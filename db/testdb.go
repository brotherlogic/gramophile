package db

import (
	"context"
	"fmt"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TestDatabase struct {
	rMap map[int64]*pb.Record
	iMap map[int64]*pb.Intent
}

func (td *TestDatabase) SaveRecord(ctx context.Context, userid int32, r *pb.Record) error {
	if td.rMap == nil {
		td.rMap = make(map[int64]*pb.Record)
	}
	td.rMap[r.GetRelease().GetInstanceId()] = r
	return nil
}

func (td *TestDatabase) GetRecords(ctx context.Context, userid int32) ([]string, error) {
	if td.rMap == nil {
		return nil, status.Errorf(codes.NotFound, "Empty td")
	}
	var keys []string
	for key := range td.rMap {
		keys = append(keys, fmt.Sprintf("%v", key))
	}
	return keys, nil
}

func (td *TestDatabase) GetRecord(ctx context.Context, userid int32, iid int64) (*pb.Record, error) {
	if td.rMap == nil {
		return nil, status.Errorf(codes.NotFound, "Unable to locate %v in %v", iid, td.rMap)
	}
	return td.rMap[iid], nil
}

func (td *TestDatabase) GetIntent(ctx context.Context, userid int32, iid int64) (*pb.Intent, error) {
	if td.iMap == nil {
		return nil, status.Errorf(codes.NotFound, "Unable to locate %v in %v", iid, td.rMap)
	}
	return td.iMap[iid], nil
}

func (td *TestDatabase) SaveIntent(ctx context.Context, userid int32, iid int64, i *pb.Intent) error {
	return nil
}

func (td *TestDatabase) LoadLogins(ctx context.Context) (*pb.UserLoginAttempts, error) {
	return nil, nil
}
func (td *TestDatabase) SaveLogins(ctx context.Context, logins *pb.UserLoginAttempts) error {
	return nil
}
func (td *TestDatabase) GenerateToken(ctx context.Context, token, secret string) (*pb.GramophileAuth, error) {
	return nil, nil
}

func (td *TestDatabase) SaveUser(ctx context.Context, user *pb.StoredUser) error {
	return nil
}
func (td *TestDatabase) DeleteUser(ctx context.Context, id string) error {
	return nil
}
func (td *TestDatabase) GetUser(ctx context.Context, user string) (*pb.StoredUser, error) {
	return nil, nil
}
func (td *TestDatabase) GetUsers(ctx context.Context) ([]string, error) {
	return nil, nil
}
