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

type Database struct{}

func NewDatabase(ctx context.Context) *Database {
	db := &Database{}
	_, err := db.GetUsers(ctx)
	if err != nil {
		log.Printf("AHA: %v", err)
	}
	return db
}

func (d *Database) LoadLogins(ctx context.Context) (*pb.UserLoginAttempts, error) {
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

func (d *Database) SaveLogins(ctx context.Context, logins *pb.UserLoginAttempts) error {
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

func (d *Database) GenerateToken(ctx context.Context, token, secret string) (*pb.GramophileAuth, error) {
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

func (d *Database) SaveUser(ctx context.Context, user *pb.StoredUser) error {
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

func (d *Database) GetUser(ctx context.Context, user string) (*pb.StoredUser, error) {
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

func (d *Database) GetUsers(ctx context.Context) ([]string, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	resp, err := client.GetKeys(ctx, &rspb.GetKeysRequest{
		Suffix: "gramophile/user",
	})
	if err != nil {
		return nil, err
	}

	users.Set(float64(len(resp.GetKeys())))

	return resp.GetKeys(), err
}
