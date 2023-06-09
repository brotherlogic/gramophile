package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/metadata"

	pbd "github.com/brotherlogic/discogs/proto"
)

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "auth-token", fmt.Sprintf("%v", userid))
}

func TestRetrieveUpdates(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB()
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d}

	r, err := s.GetRecord(ctx, &pb.GetRecordRequest{Request: &pb.GetRecordRequest_GetRecordWithId{
		GetRecordWithId: &pb.GetRecordWithId{
			InstanceId: int64(1234),
		},
	}})
	if err != nil {
		t.Fatalf("Bad get: %v", err)
	}

	if len(r.GetRecord().GetUpdates()) > 0 {
		t.Errorf("Updates retrieved when record is empty: %v", r.GetRecord())
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 13}})
	if err != nil {
		t.Fatalf("Bad save: %v", err)
	}

	r, err = s.GetRecord(ctx, &pb.GetRecordRequest{Request: &pb.GetRecordRequest_GetRecordWithId{
		GetRecordWithId: &pb.GetRecordWithId{
			InstanceId: int64(1234),
		},
	}})
	if err != nil {
		t.Fatalf("Bad get: %v", err)
	}

	if len(r.GetRecord().GetUpdates()) > 0 {
		t.Errorf("Updates retrieved when not requested: %v", r.GetRecord())
	}

	r, err = s.GetRecord(ctx, &pb.GetRecordRequest{
		IncludeHistory: true,
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{
				InstanceId: int64(1234),
			},
		}})
	if err != nil {
		t.Fatalf("Bad get: %v", err)
	}

	if len(r.GetRecord().GetUpdates()) == 0 {
		t.Errorf("No updates retreived, expected 1: %v", r.GetRecord())
	}
}
