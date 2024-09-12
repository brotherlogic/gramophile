package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ghb_client "github.com/brotherlogic/githubridge/client"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestAddIntent_FailOnBadUser(t *testing.T) {
	ctx := getTestContext(12345)

	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	qc := queuelogic.GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())
	s := Server{d: d, di: di, qc: qc}

	r, err := s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{GoalFolder: "12 Inches"},
		InstanceId: 1234,
	})
	if err == nil {
		t.Errorf("should have failed: %v (%v)", r, err)
	}
}
func TestAddIntent_FailOnBadRecord(t *testing.T) {
	ctx := getTestContext(123)

	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())
	s := Server{d: d, di: di, qc: qc}

	r, err := s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{GoalFolder: "12 Inches"},
		InstanceId: 1234,
	})
	if err == nil || !strings.Contains(fmt.Sprintf("%v", err), "record") {
		t.Errorf("should have failed with bad record: %v (%v)", r, err)
	}
}

func TestGoalFolderAddsIntent_Success(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}

	qc := queuelogic.GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{GoalFolder: "12 Inches"},
		InstanceId: 1234,
	})
	if err != nil {
		t.Fatalf("Error setting intent: %v", err)
	}

	//Run the intent
	qc.FlushQueue(ctx)

	rec, err := d.GetRecord(ctx, 123, 1234)
	if err != nil {
		t.Fatalf("Bad record retrieve: %v", err)
	}

	if rec.GetGoalFolder() != "12 Inches" {
		t.Errorf("Goal folder was not set: %v", rec)
	}
}

func TestGoalFolderAddsIntent_FailMissingFolder(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)

	err := d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{GoalFolder: "12 Inches"},
		InstanceId: 1234,
	})
	if err == nil || status.Code(err) == codes.NotFound {
		t.Errorf("Intent did not fail with missing folder: %v", err)
	}
}

func TestGoalFolderAddsIntent_FailNoSleeve(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)

	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Config: &pb.GramophileConfig{SleeveConfig: &pb.SleeveConfig{
			AllowedSleeves: []*pb.Sleeve{{Name: "MadeUpSleeve"}},
		}},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{Sleeve: "DifferenetSleeve"},
		InstanceId: 1234,
	})
	if err == nil || status.Code(err) == codes.NotFound {
		t.Errorf("Intent did not fail with missing sleeve: %v", err)
	}
}

func TestGoalFolderAddsIntent_SuccessOnSleeve(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Config: &pb.GramophileConfig{SleeveConfig: &pb.SleeveConfig{
			AllowedSleeves: []*pb.Sleeve{{Name: "MadeUpSleeve"}},
		}},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{Sleeve: "MadeUpSleeve"},
		InstanceId: 1234,
	})
	if err != nil {
		t.Errorf("Intent was not set: %v", err)
	}
}

func TestKeepIntent_FailWithNoMint(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{
		User:   &pbd.User{DiscogsUserId: 123},
		Config: &pb.GramophileConfig{KeepConfig: &pb.KeepConfig{Mandate: pb.Mandate_REQUIRED}},
		Auth:   &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{Keep: pb.KeepStatus_MINT_UP_KEEP},
		InstanceId: 1234,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Wrong error in returning: %v", err)
	}
}
