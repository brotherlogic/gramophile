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
	"google.golang.org/protobuf/reflect/protoreflect"
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

func NewTestDB(cl rstore_client.RStoreClient) Database {
	return &DB{client: cl}
}

type Database interface {
	GetRecord(ctx context.Context, userid int32, iid int64) (*pb.Record, error)
	DeleteRecord(ctx context.Context, userid int32, iid int64) error
	GetRecords(ctx context.Context, userid int32) ([]int64, error)
	LoadAllRecords(ctx context.Context, userid int32) ([]*pb.Record, error)
	SaveRecord(ctx context.Context, userid int32, record *pb.Record) error
	GetUpdates(ctx context.Context, userid int32, record *pb.Record) ([]*pb.RecordUpdate, error)
	SaveUpdate(ctx context.Context, userid int32, record *pb.Record, update *pb.RecordUpdate) error

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

	GetWants(ctx context.Context, userid int32) ([]*pb.Want, error)
	SaveWant(ctx context.Context, userid int32, want *pb.Want) error
	DeleteWant(ctx context.Context, user *pb.StoredUser, want int64) error

	SaveWantlist(ctx context.Context, userid int32, wantlist *pb.Wantlist) error
	LoadWantlist(ctx context.Context, user *pb.StoredUser, name string) (*pb.Wantlist, error)
	GetWantlists(ctx context.Context, userId int32) ([]*pb.Wantlist, error)

	SaveSale(ctx context.Context, userId int32, sale *pb.SaleInfo) error
	GetSales(ctx context.Context, userId int32) ([]int64, error)
	GetSale(ctx context.Context, userId int32, saleId int64) (*pb.SaleInfo, error)

	Clean(ctx context.Context) error
}

func NewDatabase(ctx context.Context) Database {
	db := &DB{} //rcache: make(map[int32]map[int64]*pb.Record)}
	client, err := rstore_client.GetClient()
	if err != nil {
		log.Fatalf("Dial error on db -> rstore: %v", err)
	}
	db.client = client

	return db
}

func (d *DB) save(ctx context.Context, key string, message protoreflect.ProtoMessage) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	_, err = d.client.Write(ctx, &rspb.WriteRequest{
		Key:   key,
		Value: &anypb.Any{Value: data},
	})
	return err
}

func (d *DB) delete(ctx context.Context, key string) error {
	_, err := d.client.Delete(ctx, &rspb.DeleteRequest{
		Key: key,
	})

	return err
}

func (d *DB) load(ctx context.Context, key string) ([]byte, error) {
	val, err := d.client.Read(ctx, &rspb.ReadRequest{
		Key: key,
	})
	if err != nil {
		return nil, err
	}
	return val.GetValue().GetValue(), nil
}

func (d *DB) resolve(ctx context.Context, req *rspb.GetKeysRequest) ([][]byte, error) {
	keys, err := d.client.GetKeys(ctx, req)
	if err != nil {
		return nil, err
	}

	var data [][]byte
	for _, key := range keys.GetKeys() {
		val, err := d.client.Read(ctx, &rspb.ReadRequest{
			Key: key,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to read %v -> %w", key, err)
		}
		data = append(data, val.GetValue().GetValue())
	}

	return data, nil
}

func (d *DB) GetWantlists(ctx context.Context, userid int32) ([]*pb.Wantlist, error) {
	datas, err := d.resolve(ctx, &rspb.GetKeysRequest{Prefix: fmt.Sprintf("gramophile/%v/wantlist/", userid)})
	if err != nil {
		return nil, err
	}

	var lists []*pb.Wantlist
	for _, data := range datas {
		list := &pb.Wantlist{}
		err := proto.Unmarshal(data, list)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal %w", err)
		}

		lists = append(lists, list)
	}

	return lists, nil
}

func (d *DB) SaveWantlist(ctx context.Context, userid int32, wantlist *pb.Wantlist) error {
	log.Printf("Saving wantlist: %v", wantlist)
	return d.save(ctx, fmt.Sprintf("gramophile/%v/wantlist/%v", userid, wantlist.GetName()), wantlist)
}

func (d *DB) LoadWantlist(ctx context.Context, user *pb.StoredUser, wantlist string) (*pb.Wantlist, error) {
	data, err := d.load(ctx, fmt.Sprintf("gramophile/%v/wantlist/%v", user.GetUser().GetDiscogsUserId(), wantlist))
	if err != nil {
		return nil, err
	}

	wl := &pb.Wantlist{}
	err = proto.Unmarshal(data, wl)
	return wl, err
}

func (d *DB) SaveSale(ctx context.Context, userid int32, sale *pb.SaleInfo) error {
	return d.save(ctx, fmt.Sprintf("gramophile/user/%v/sale/%v", userid, sale.GetSaleId()), sale)
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
	err = proto.Unmarshal(val.GetValue().GetValue(), logins)
	if err != nil {
		return nil, err
	}

	return logins, nil
}

func (d *DB) GetWants(ctx context.Context, userid int32) ([]*pb.Want, error) {
	keys, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{
		Prefix: fmt.Sprintf("gramophile/user/%v/want/", userid),
	})
	if err != nil {
		return nil, err
	}

	var wants []*pb.Want
	for _, key := range keys.GetKeys() {
		data, err := d.load(ctx, key)
		if err != nil {
			return nil, err
		}

		want := &pb.Want{}
		err = proto.Unmarshal(data, want)
		if err != nil {
			return nil, err
		}

		wants = append(wants, want)
	}

	return wants, nil
}

func (d *DB) SaveWant(ctx context.Context, userid int32, want *pb.Want) error {
	log.Printf("SAVING: %v", want)
	return d.save(ctx, fmt.Sprintf("gramophile/user/%v/want/%v", userid, want.GetId()), want)
}

func (d *DB) DeleteWant(ctx context.Context, user *pb.StoredUser, want int64) error {
	return d.delete(ctx, fmt.Sprintf("gramophile/user/%v/want/%v", user.GetUser().GetDiscogsUserId(), want))
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

func ResolveDiff(update *pb.RecordUpdate) []string {
	var diff []string
	if update.GetBefore().GetGoalFolder() != update.GetAfter().GetGoalFolder() {
		if update.GetBefore().GetGoalFolder() == "" {
			diff = append(diff, fmt.Sprintf("Goal Folder was set to %v", update.GetAfter().GetGoalFolder()))
		}
	}
	return diff
}

func (d *DB) saveUpdate(ctx context.Context, userid int32, old, new *pb.Record) error {
	update := &pb.RecordUpdate{
		Date:   time.Now().Unix(),
		Before: old,
		After:  new,
	}
	return d.SaveUpdate(ctx, userid, new, update)
}

func (d *DB) SaveUpdate(ctx context.Context, userid int32, r *pb.Record, update *pb.RecordUpdate) error {
	data, err := proto.Marshal(update)
	if err != nil {
		return err
	}
	_, err = d.client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v/release/%v-%v.update", userid, r.GetRelease().GetInstanceId(), update.GetDate()),
		Value: &anypb.Any{Value: data},
	})
	return err
}

func (d *DB) DeleteRecord(ctx context.Context, userid int32, iid int64) error {
	_, err := d.client.Delete(ctx, &rspb.DeleteRequest{
		Key: fmt.Sprintf("gramophile/user/%v/release/%v", userid, iid),
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

func (d *DB) GetUpdates(ctx context.Context, userid int32, r *pb.Record) ([]*pb.RecordUpdate, error) {
	resp, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: fmt.Sprintf("gramophile/user/%v/release/%v", userid, r.GetRelease().GetInstanceId())})
	if err != nil {
		return nil, err
	}

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
	resp, err := d.client.Read(ctx, &rspb.ReadRequest{
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
	data, err := proto.Marshal(i)
	if err != nil {
		return err
	}

	_, err = d.client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v/release/intent-%v", userid, iid),
		Value: &anypb.Any{Value: data},
	})

	return err
}

func (d *DB) GetRecords(ctx context.Context, userid int32) ([]int64, error) {
	resp, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{
		Prefix:      fmt.Sprintf("gramophile/user/%v/release/", userid),
		AvoidSuffix: []string{"update"},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting keys: %w", err)
	}

	var ret []int64
	for _, key := range resp.GetKeys() {
		if !strings.Contains(key, "intent") {
			pieces := strings.Split(key, "/")
			val, _ := strconv.ParseInt(pieces[len(pieces)-1], 10, 64)
			ret = append(ret, val)
		}
	}

	return ret, nil
}

func (d *DB) LoadAllRecords(ctx context.Context, userid int32) ([]*pb.Record, error) {
	iids, err := d.GetRecords(ctx, userid)
	if err != nil {
		fmt.Errorf("unable to get records: %w", err)
	}

	var records []*pb.Record
	for _, iid := range iids {
		rec, err := d.GetRecord(ctx, userid, iid)
		if err != nil {
			return nil, fmt.Errorf("unable to read (%v) -> %w", iid, err)
		}

		records = append(records, rec)
	}

	return records, nil
}

func (d *DB) GetSales(ctx context.Context, userid int32) ([]int64, error) {
	log.Printf("LOADING %v", fmt.Sprintf("gramophile/user/%v/sale/", userid))
	resp, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{
		Prefix: fmt.Sprintf("gramophile/user/%v/sale/", userid),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting keys: %w", err)
	}

	var ret []int64
	for _, key := range resp.GetKeys() {
		pieces := strings.Split(key, "/")
		val, _ := strconv.ParseInt(pieces[len(pieces)-1], 10, 64)
		ret = append(ret, val)
	}

	return ret, nil
}

func (d *DB) GetSale(ctx context.Context, userid int32, saleid int64) (*pb.SaleInfo, error) {
	data, err := d.load(ctx, fmt.Sprintf("gramophile/user/%v/sale/%v", userid, saleid))
	if err != nil {
		return nil, err
	}

	ret := &pb.SaleInfo{}
	err = proto.Unmarshal(data, ret)
	return ret, err
}

func (d *DB) GetUsers(ctx context.Context) ([]string, error) {
	resp, err := d.client.GetKeys(ctx, &rspb.GetKeysRequest{
		Prefix: USER_PREFIX,
	})
	if err != nil {
		return nil, err
	}

	users.Set(float64(len(resp.GetKeys())))

	// Trim out the prefix from the returned keys
	var rusers []string
	for _, key := range resp.GetKeys() {
		rusers = append(rusers, key[len(USER_PREFIX):])
	}

	return rusers, err
}

func (d *DB) Clean(ctx context.Context) error {

	return nil
}
