package integration

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	pstore_client "github.com/brotherlogic/pstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSync_WantAddedToFloat(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 20, Name: "Keep"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{
			TransferList: "float",
			Existing:     pb.WantsExisting_EXISTING_LIST,
		},
	}})

	_, err = di.AddWant(context.Background(), 1234)
	if err != nil {
		t.Fatalf("Unable to add want: %v", err)
	}

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 1,
			Auth:    "123",
			Entry:   &pb.QueueElement_SyncWants{SyncWants: &pb.SyncWants{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Bad wants: %v", err)
	}

	if len(wants.GetWants()) != 1 {
		t.Fatalf("Want was not live: %v", wants)
	}
	found := false
	for _, listy := range wants.GetWants()[0].GetWant().GetFromWantlist() {
		if listy == "float" {
			found = true
		}
	}
	if !found {
		t.Errorf("Float was not listed: %v", wants)
	}

	list, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "float"})
	if err != nil {
		t.Fatalf("Unable to get list: %v", err)
	}

	if len(list.List.GetEntries()) != 1 {
		t.Errorf("Could not get entries: %v", list)
	}

	// Now if we get user config, do we get the float list as we expect?
	user, err := s.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("Unable to get user: %v", err)
	}

	var floatList *pb.StoredWantlist
	for _, list := range user.GetUser().GetConfig().GetWantsListConfig().GetWantlists() {
		if list.GetName() == "float" {
			floatList = list
		}
	}

	if floatList == nil {
		t.Fatalf("Could not find float list: %v", user)
	}

	if len(floatList.GetEntries()) != 1 {
		t.Errorf("Floatlist does not have 1 entry: %v", floatList)
	}

}
