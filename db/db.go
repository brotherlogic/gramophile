package db

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	pb "github.com/brotherlogic/gramophile/proto"
	rspb "github.com/brotherlogic/rstore/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	users = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_users",
		Help: "The size of the user list",
	})
)

type DB struct{}

type Database interface {
	GetRecord(ctx context.Context, userid int32, iid int64) (*pb.Record, error)
	GetIntent(ctx context.Context, userid int32, iid int64) (*pb.Intent, error)
	GetRecords(ctx context.Context, userid int32) ([]string, error)
	SaveRecord(ctx context.Context, userid int32, record *pb.Record) error

	LoadLogins(ctx context.Context) (*pb.UserLoginAttempts, error)
	SaveLogins(ctx context.Context, logins *pb.UserLoginAttempts) error
	GenerateToken(ctx context.Context, token, secret string) (*pb.GramophileAuth, error)

	SaveUser(ctx context.Context, user *pb.StoredUser) error
	DeleteUser(ctx context.Context, id string) error
	GetUser(ctx context.Context, user string) (*pb.StoredUser, error)
	GetUsers(ctx context.Context) ([]string, error)
}

func NewDatabase(ctx context.Context) Database {
	db := &DB{}
	b, err := db.GetUsers(ctx)
	log.Printf("WHAT %v / %v", b, err)
	return db
}

func (d *DB) LoadLogins(ctx context.Context) (*pb.UserLoginAttempts, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	val, err := client.Read(ctx, &rspb.ReadRequest{
		Key: "gramophile/logins",
	})
	if err != nil {
		// OutOfRange indicates that the key was not found
		if status.Code(err) == codes.NotFound {
			return &pb.UserLoginAttempts{}, nil
		}
		return nil, err
	}

	logins := &pb.UserLoginAttempts{}
	log.Printf("Unmarshal: %v -> %v", logins, val.GetValue().GetValue())
	err = proto.Unmarshal(val.GetValue().GetValue(), logins)
	if err != nil {
		return nil, err
	}

	return logins, nil
}

func (d *DB) SaveLogins(ctx context.Context, logins *pb.UserLoginAttempts) error {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}

	data, err := proto.Marshal(logins)
	if err != nil {
		return err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Write(ctx, &rspb.WriteRequest{
		Key:   "gramophile/logins",
		Value: &anypb.Any{Value: data},
	})

	return err
}

func (d *DB) GenerateToken(ctx context.Context, token, secret string) (*pb.GramophileAuth, error) {
	user := fmt.Sprintf("%v-%v", time.Now().UnixNano(), rand.Int63())
	log.Printf("GERENATING %v and %v", token, secret)
	su := &pb.StoredUser{
		Auth:       &pb.GramophileAuth{Token: user},
		UserToken:  token,
		UserSecret: secret,
	}

	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	data, err := proto.Marshal(su)
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v", user),
		Value: &anypb.Any{Value: data},
	})

	return &pb.GramophileAuth{Token: user}, err
}

func (d *DB) SaveUser(ctx context.Context, user *pb.StoredUser) error {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}

	data, err := proto.Marshal(user)
	if err != nil {
		return err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v", user.Auth.Token),
		Value: &anypb.Any{Value: data},
	})

	return err
}

func (d *DB) DeleteUser(ctx context.Context, id string) error {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Delete(ctx, &rspb.DeleteRequest{
		Key: fmt.Sprintf("gramophile/user/%v", id),
	})

	return err
}

func (d *DB) GetUser(ctx context.Context, user string) (*pb.StoredUser, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	resp, err := client.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("gramophile/user/%v", user),
	})
	if err != nil {
		return nil, err
	}

	su := &pb.StoredUser{}
	err = proto.Unmarshal(resp.GetValue().GetValue(), su)
	return su, err
}

func (d *DB) SaveRecord(ctx context.Context, userid int32, record *pb.Record) error {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}

	data, err := proto.Marshal(record)
	if err != nil {
		return err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v/release/%v", userid, record.GetRelease().GetInstanceId()),
		Value: &anypb.Any{Value: data},
	})

	return err
}

func (d *DB) GetRecord(ctx context.Context, userid int32, iid int64) (*pb.Record, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	resp, err := client.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("gramophile/user/%v/release/%v", userid, iid),
	})
	if err != nil {
		return nil, err
	}

	su := &pb.Record{}
	err = proto.Unmarshal(resp.GetValue().GetValue(), su)
	return su, err
}

func (d *DB) GetIntent(ctx context.Context, userid int32, iid int64) (*pb.Intent, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	resp, err := client.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("gramophile/user/%v/release/intent-%v", userid, iid),
	})
	if err != nil {
		return nil, err
	}

	in := &pb.Intent{}
	err = proto.Unmarshal(resp.GetValue().GetValue(), in)
	return in, err
}

func (d *DB) GetRecords(ctx context.Context, userid int32) ([]string, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	resp, err := client.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: fmt.Sprintf("gramophile/user/%v/release/", userid)})
	if err != nil {
		return nil, err
	}
	return resp.GetKeys(), nil
}

func (d *DB) GetUsers(ctx context.Context) ([]string, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	resp, err := client.GetKeys(ctx, &rspb.GetKeysRequest{
		Prefix: "gramophile/user",
	})
	if err != nil {
		return nil, err
	}

	users.Set(float64(len(resp.GetKeys())))

	// Trim out the prefix from the returned keys
	var rusers []string
	for _, key := range resp.GetKeys() {
		rusers = append(rusers, key[len("gramophile/user/"):])
	}

	return rusers, err
}
