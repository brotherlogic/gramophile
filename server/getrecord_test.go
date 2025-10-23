package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/grpc/metadata"

	pbd "github.com/brotherlogic/discogs/proto"
)

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "auth-token", fmt.Sprintf("%v", userid))
}

func TestReverse(t *testing.T) {
	recs := []*pb.Record{
		{
			Release: &pbd.Release{InstanceId: 1},
		},
		{
			Release: &pbd.Release{InstanceId: 2},
		},
	}

	nrecs := reverse(recs)

	if nrecs[0].GetRelease().GetInstanceId() != 2 {
		t.Errorf("Bad reverse")
	}
}

func TestRetrieveUpdates(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
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

	if len(r.GetRecords()[0].GetUpdates()) > 0 {
		t.Errorf("Updates retrieved when record is empty: %v", r.GetRecords()[0].GetRecord())
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

	if len(r.GetRecords()[0].GetUpdates()) > 0 {
		t.Errorf("Updates retrieved when not requested: %v", r.GetRecords()[0].GetRecord())
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

	if len(r.GetRecords()[0].GetUpdates()) == 0 {
		t.Errorf("No updates retreived, expected 1: %v", r.GetRecords()[0].GetRecord())
	}
}

func TestGetSale(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	err = d.SaveSale(ctx, 123, &pb.SaleInfo{SaleId: 12345, CurrentPrice: &pbd.Price{Value: 12345}})

	s := Server{d: d}

	sale, err := s.GetSale(ctx, &pb.GetSaleRequest{Id: 12345})
	if err != nil {
		t.Fatalf("Bad sale return %v", err)
	}

	if sale.GetSales()[0].GetCurrentPrice().GetValue() != 12345 {
		t.Errorf("Bad sale return: %v", sale)
	}
}

func TestGetByLabel(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
	err := d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1, Labels: []*pbd.Label{{Id: 12}}}})
	if err != nil {
		t.Fatalf("Can't save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 2, Labels: []*pbd.Label{{Id: 12}}}})
	if err != nil {
		t.Fatalf("Can't save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 3, Labels: []*pbd.Label{{Id: 13}}}})
	if err != nil {
		t.Fatalf("Can't save record: %v", err)
	}

	s := Server{d: d}

	rs, err := s.GetRecord(ctx, &pb.GetRecordRequest{Request: &pb.GetRecordRequest_GetRecordWithId{
		GetRecordWithId: &pb.GetRecordWithId{LabelId: 12},
	}})
	if err != nil {
		t.Fatalf("Bad sale return %v", err)
	}

	if len(rs.GetRecords()) != 2 {
		t.Errorf("Wrong number of records returned: %v; should have been 2", len(rs.GetRecords()))
	}

	for _, r := range rs.GetRecords() {
		if r.GetRecord().GetRelease().GetInstanceId() > 2 {
			t.Errorf("Bad record returned: %v", r)
		}
	}
}
