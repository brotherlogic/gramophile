package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pbd "github.com/brotherlogic/discogs/proto"
)

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "auth-token", fmt.Sprintf("%v", userid))
}

func getTestContextBeta(userid int) context.Context {
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

func TestGetByDate(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
	err := d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{DateAdded: 100, InstanceId: 1, Labels: []*pbd.Label{{Id: 12}}}})
	if err != nil {
		t.Fatalf("Can't save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{DateAdded: 1000, InstanceId: 2, Labels: []*pbd.Label{{Id: 12}}}})
	if err != nil {
		t.Fatalf("Can't save record: %v", err)
	}

	s := Server{d: d}

	rs, err := s.GetRecord(ctx, &pb.GetRecordRequest{Request: &pb.GetRecordRequest_GetRecordsPurchasedBetween{
		GetRecordsPurchasedBetween: &pb.GetRecordsPurchasedBetween{
			StartDate: 100,
			EndDate:   999,
		},
	}})
	if err != nil {
		t.Fatalf("Bad sale return %v", err)
	}

	if len(rs.GetRecords()) != 1 {
		t.Errorf("Wrong number of records returned: %v; should have been 2", len(rs.GetRecords()))
	}

	for _, r := range rs.GetRecords() {
		if r.GetRecord().GetRelease().GetInstanceId() > 1 {
			t.Errorf("Bad record returned: %v", r)
		}
	}
}

type failLoadAllRecordsDB struct {
	db.Database
}

func (f *failLoadAllRecordsDB) LoadAllRecords(ctx context.Context, userid int32) ([]*pb.Record, error) {
	return nil, status.Errorf(codes.Internal, "injected LoadAllRecords failure")
}

func TestGetRecord_GetAllRecords(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
	err := d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 101, Title: "Record 1"}})
	if err != nil {
		t.Fatalf("Can't save record 1: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 102, Title: "Record 2"}})
	if err != nil {
		t.Fatalf("Can't save record 2: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 103, Title: "Record 3"}})
	if err != nil {
		t.Fatalf("Can't save record 3: %v", err)
	}

	s := Server{d: d}

	res, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetAllRecords{
			GetAllRecords: true,
		},
	})
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if len(res.GetRecords()) != 3 {
		t.Fatalf("Expected 3 records, got %v", len(res.GetRecords()))
	}

	found := make(map[int64]bool)
	for _, r := range res.GetRecords() {
		found[r.GetRecord().GetRelease().GetInstanceId()] = true
	}
	for _, expectedId := range []int64{101, 102, 103} {
		if !found[expectedId] {
			t.Errorf("Expected record instance id %v to be returned", expectedId)
		}
	}
}

func TestGetRecord_GetAllRecords_Error(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
	err := d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	failDB := &failLoadAllRecordsDB{Database: d}
	s := Server{d: failDB}

	_, err = s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetAllRecords{
			GetAllRecords: true,
		},
	})
	if err == nil {
		t.Fatalf("Expected error from LoadAllRecords failure, got nil")
	}
	if status.Code(err) != codes.Internal {
		t.Errorf("Expected Internal error code, got %v (err: %v)", status.Code(err), err)
	}
}

func TestGetRecord_GetAllRecords_IncludeHistory(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
	err := d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 201, FolderId: 10}, SaleId: 999})
	if err != nil {
		t.Fatalf("Can't save initial record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 201, FolderId: 11}, SaleId: 999})
	if err != nil {
		t.Fatalf("Can't update record: %v", err)
	}

	err = d.SaveSale(ctx, 123, &pb.SaleInfo{SaleId: 999, CurrentPrice: &pbd.Price{Value: 2500}})
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	s := Server{d: d}

	// 1. Without IncludeHistory: expensive history and sales lookups must be bypassed
	resNoHist, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetAllRecords{
			GetAllRecords: true,
		},
		IncludeHistory: false,
	})
	if err != nil {
		t.Fatalf("GetRecord (no history) failed: %v", err)
	}
	if len(resNoHist.GetRecords()) != 1 {
		t.Fatalf("Expected 1 record, got %v", len(resNoHist.GetRecords()))
	}
	if resNoHist.GetRecords()[0].GetSaleInfo() != nil {
		t.Errorf("Expected SaleInfo to be bypassed without IncludeHistory, got %v", resNoHist.GetRecords()[0].GetSaleInfo())
	}
	if len(resNoHist.GetRecords()[0].GetUpdates()) > 0 {
		t.Errorf("Expected Updates to be empty without IncludeHistory, got %v", resNoHist.GetRecords()[0].GetUpdates())
	}

	// 2. With IncludeHistory: history and sales lookups are populated
	resWithHist, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetAllRecords{
			GetAllRecords: true,
		},
		IncludeHistory: true,
	})
	if err != nil {
		t.Fatalf("GetRecord (with history) failed: %v", err)
	}
	if len(resWithHist.GetRecords()) != 1 {
		t.Fatalf("Expected 1 record, got %v", len(resWithHist.GetRecords()))
	}
	if resWithHist.GetRecords()[0].GetSaleInfo() == nil || resWithHist.GetRecords()[0].GetSaleInfo().GetSaleId() != 999 {
		t.Errorf("Expected SaleInfo to be populated with IncludeHistory, got %v", resWithHist.GetRecords()[0].GetSaleInfo())
	}
	if len(resWithHist.GetRecords()[0].GetUpdates()) == 0 {
		t.Errorf("Expected Updates to be populated with IncludeHistory, got 0 updates")
	}
}


