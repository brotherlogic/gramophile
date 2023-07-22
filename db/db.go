package db

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	rstore_client "github.com/brotherlogic/rstore/client"
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

	recordLoadTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "gramophile_record_load",
		Help: "The time to load a record",
	})

	USER_PREFIX = "gramophile/user/"
)

type DB struct {
	client rstore_client.RStoreClient
}

func NewTestDB() Database {
	return &DB{client: rstore_client.GetTestClient()}
}

type Database interface {
	GetRecord(ctx context.Context, userid int32, iid int64) (*pb.Record, error)
	GetRecords(ctx context.Context, userid int32) ([]int64, error)
	SaveRecord(ctx context.Context, userid int32, record *pb.Record) error
	GetUpdates(ctx context.Context, user *pb.StoredUser, record *pb.Record) ([]*pb.RecordUpdate, error)

	GetIntent(ctx context.Context, userid int32, iid int64) (*pb.Intent, error)
	SaveIntent(ctx context.Context, userid int32, iid int64, i *pb.Intent) error

	LoadLogins(ctx context.Context) (*pb.UserLoginAttempts, error)
	SaveLogins(ctx context.Context, logins *pb.UserLoginAttempts) error
	GenerateToken(ctx context.Context, token, secret string) (*pb.GramophileAuth, error)

	SaveUser(ctx context.Context, user *pb.StoredUser) error
	DeleteUser(ctx context.Context, id string) error
	GetUser(ctx context.Context, user string) (*pb.StoredUser, error)
	GetUsers(ctx context.Context) ([]string, error)

	LoadSnapshot(ctx context.Context, user *pb.StoredUser, org string, hash string) (*pb.OrganisationSnapshot, error)
	SaveSnapshot(ctx context.Context, user *pb.StoredUser, org string, snapshot *pb.OrganisationSnapshot) error
	GetLatestSnapshot(ctx context.Context, user *pb.StoredUser, org string) (*pb.OrganisationSnapshot, error)

	Clean(ctx context.Context) error
}

func NewDatabase(ctx context.Context) Database {
	db := &DB{} //rcache: make(map[int32]map[int64]*pb.Record)}
	client, err := rstore_client.GetClient()
	if err != nil {
		log.Fatalf("Dial error on db -> rstore: %v", err)
	}
	db.client = client

	log.Printf("Connected to DB")
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

func cleanOrgString(org string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, strings.ToLower(org))
}

func (d *DB) SaveSnapshot(ctx context.Context, user *pb.StoredUser, org string, snapshot *pb.OrganisationSnapshot) error {
	data, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	_, err = d.client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/%v/org/%v/%v", user.GetUser().GetDiscogsUserId(), cleanOrgString(org), snapshot.GetDate()),
		Value: &anypb.Any{Value: data},
	})

	if snapshot.GetName() != "" {
		_, err = d.client.Write(ctx, &rspb.WriteRequest{
			Key:   fmt.Sprintf("gramophile/%v/org/%v/%v", user.GetUser().GetDiscogsUserId(), cleanOrgString(org), cleanOrgString(snapshot.GetName())),
			Value: &anypb.Any{Value: data},
		})
	}

	return err
}

func (d *DB) LoadSnapshot(ctx context.Context, user *pb.StoredUser, org string, name string) (*pb.OrganisationSnapshot, error) {
	val, err := d.client.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("gramophile/%v/org/%v/%v", user.GetUser().GetDiscogsUserId(), cleanOrgString(org), cleanOrgString(name)),
	})
	if err != nil {
		return nil, err
	}

	snapshot := &pb.OrganisationSnapshot{}
	err = proto.Unmarshal(val.GetValue().GetValue(), snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (d *DB) GetLatestSnapshot(ctx context.Context, user *pb.StoredUser, org string) (*pb.OrganisationSnapshot, error) {
	keys, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: fmt.Sprintf("gramophile/%v/org/%v/", user.GetUser().GetDiscogsUserId(), cleanOrgString(org))})
	if err != nil {
		return nil, fmt.Errorf("Cannot get keys to find latest snapshot: %w", err)
	}

	sort.Strings(keys.Keys)

	if len(keys.Keys) == 0 {
		return nil, status.Errorf(codes.NotFound, "no orgs for %v found", user.GetUser().GetDiscogsUserId())
	}

	resp, err := d.client.Read(ctx, &rspb.ReadRequest{
		Key: keys.Keys[0],
	})
	if err != nil {
		return nil, err
	}

	orgSnap := &pb.OrganisationSnapshot{}
	err = proto.Unmarshal(resp.GetValue().GetValue(), orgSnap)
	return orgSnap, err
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
		Key:   fmt.Sprintf("%v%v", USER_PREFIX, user),
		Value: &anypb.Any{Value: data},
	})

	return &pb.GramophileAuth{Token: user}, err
}

func (d *DB) SaveUser(ctx context.Context, user *pb.StoredUser) error {
	data, err := proto.Marshal(user)
	if err != nil {
		return err
	}

	_, err = d.client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("%v%v", USER_PREFIX, user.Auth.Token),
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
		Key: fmt.Sprintf("%v%v", USER_PREFIX, id),
	})

	return err
}

func (d *DB) GetUser(ctx context.Context, user string) (*pb.StoredUser, error) {
	resp, err := d.client.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("%v%v", USER_PREFIX, user),
	})
	if err != nil {
		return nil, err
	}

	su := &pb.StoredUser{}
	err = proto.Unmarshal(resp.GetValue().GetValue(), su)
	return su, err
}

func (d *DB) SaveRecord(ctx context.Context, userid int32, record *pb.Record) error {
	data, err := proto.Marshal(record)
	if err != nil {
		return err
	}

	old, err := d.client.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("gramophile/user/%v/release/%v", userid, record.GetRelease().GetInstanceId()),
	})
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return err
		}
		old = &rspb.ReadResponse{}
		old.Value = &anypb.Any{Value: []byte{}}
	}

	oldRecord := &pb.Record{}
	err = proto.Unmarshal(old.GetValue().GetValue(), oldRecord)
	if err != nil {
		return err
	}

	err = d.saveUpdate(ctx, userid, oldRecord, record)
	if err != nil {
		return err
	}

	_, err = d.client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v/release/%v", userid, record.GetRelease().GetInstanceId()),
		Value: &anypb.Any{Value: data},
	})

	return err
}

func (d *DB) saveUpdate(ctx context.Context, userid int32, old, new *pb.Record) error {
	update := &pb.RecordUpdate{
		Date:   time.Now().Unix(),
		Before: old,
		After:  new,
	}
	data, err := proto.Marshal(new)
	if err != nil {
		return err
	}

	_, err = d.client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v/release/%v-%v.update", userid, new.GetRelease().GetInstanceId(), update.GetDate()),
		Value: &anypb.Any{Value: data},
	})
	return err

}

func (d *DB) GetRecord(ctx context.Context, userid int32, iid int64) (*pb.Record, error) {
	t := time.Now()
	resp, err := d.client.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("gramophile/user/%v/release/%v", userid, iid),
	})
	if err != nil {
		return nil, err
	}

	su := &pb.Record{}
	err = proto.Unmarshal(resp.GetValue().GetValue(), su)

	if err == nil {
		recordLoadTime.Observe(float64(time.Since(t).Milliseconds()))
	}

	return su, err
}

func (d *DB) GetUpdates(ctx context.Context, u *pb.StoredUser, r *pb.Record) ([]*pb.RecordUpdate, error) {
	resp, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: fmt.Sprintf("gramophile/user/%v/release/%v", u.GetUser().GetDiscogsUserId(), r.GetRelease().GetInstanceId())})
	if err != nil {
		return nil, err
	}

	log.Printf("UPDATES: %v", resp)

	var updates []*pb.RecordUpdate
	for _, key := range resp.GetKeys() {
		if strings.HasSuffix(key, ".update") {
			update := &pb.RecordUpdate{}
			resp, err := d.client.Read(ctx, &rspb.ReadRequest{
				Key: key,
			})
			if err != nil {
				return nil, err
			}
			err = proto.Unmarshal(resp.GetValue().GetValue(), update)
			if err != nil {
				return nil, err
			}
			updates = append(updates, update)
		}
	}
	return updates, nil
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

func (d *DB) SaveIntent(ctx context.Context, userid int32, iid int64, i *pb.Intent) error {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}

	data, err := proto.Marshal(i)
	if err != nil {
		return err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v/release/intent-%v", userid, iid),
		Value: &anypb.Any{Value: data},
	})

	return err
}

func (d *DB) GetRecords(ctx context.Context, userid int32) ([]int64, error) {
	resp, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: fmt.Sprintf("gramophile/user/%v/release/", userid)})
	if err != nil {
		return nil, err
	}

	var ret []int64
	for _, key := range resp.GetKeys() {
		if !strings.Contains(key, "intent") && !strings.Contains(key, "update") {
			pieces := strings.Split(key, "/")
			val, _ := strconv.ParseInt(pieces[len(pieces)-1], 10, 64)
			ret = append(ret, val)
		}
	}

	return ret, nil
}

func (d *DB) GetUsers(ctx context.Context) ([]string, error) {
	resp, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{
		Prefix: USER_PREFIX,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("USER KEYS: %v", resp.GetKeys())

	users.Set(float64(len(resp.GetKeys())))

	// Trim out the prefix from the returned keys
	var rusers []string
	for _, key := range resp.GetKeys() {
		rusers = append(rusers, key[len(USER_PREFIX):])
	}

	log.Printf("TO: %v", rusers)

	return rusers, err
}

func (d *DB) Clean(ctx context.Context) error {
	//Clean out bad users
	users, err := d.GetUsers(ctx)
	if err != nil {
		return err
	}

	for _, u := range users {
		user, err := d.GetUser(ctx, u)
		if err != nil {
			return err
		}

		if u != user.GetAuth().GetToken() {
			conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
			if err != nil {
				return err
			}

			client := rspb.NewRStoreServiceClient(conn)
			_, err = client.Delete(ctx, &rspb.DeleteRequest{Key: fmt.Sprintf("%v%v", USER_PREFIX, u)})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
