package background

import (
	"context"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"

	dpb "github.com/brotherlogic/discogs/proto"
	pbd "github.com/brotherlogic/discogs/proto"

	pstore_client "github.com/brotherlogic/pstore/client"
)

func GetTestBackgroundRunner() *BackgroundRunner {
	return &BackgroundRunner{
		db: db.NewTestDB(pstore_client.GetTestClient()),
	}
}

func TestGetCollectionPage_WithNewRecord(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2})

	_, err := b.ProcessCollectionPage(context.Background(), d, 1, 123)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	record, err := b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	if record.GetRelease().GetRating() != 2 {
		t.Errorf("Stored record is not quite right: %v", record)
	}
}

func TestGetCollectionPage_WithDeletion(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2})

	_, err := b.ProcessCollectionPage(context.Background(), d, 1, 123)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	_, err = b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	d = &discogs.TestDiscogsClient{}
	_, err = b.ProcessCollectionPage(context.Background(), d, 1, 1234)
	if err != nil {
		t.Fatalf("Bad collection pull (2); %v", err)
	}

	err = b.db.SaveUser(context.Background(), &pb.StoredUser{
		Auth: &pb.GramophileAuth{Token: "test-token-deletion"},
		UserToken: "test-token-deletion",
		User: &pbd.User{DiscogsUserId: d.GetUserId()},
	})
	if err != nil {
		t.Fatalf("Unable to save user: %v", err)
	}

	var enqueuedIid int64
	err = b.CleanCollection(context.Background(), d, 1234, "test-token-deletion", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueuedIid = req.GetElement().GetDeleteRecord().GetIid()
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if enqueuedIid != 100 {
		t.Errorf("Should have enqueued deletion of 100, got %v", enqueuedIid)
	}

	err = b.DeleteRecord(context.Background(), d, enqueuedIid)
	if err != nil {
		t.Fatalf("Delete record failed: %v", err)
	}

	rec, err := b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err == nil && rec.GetRelease().GetRating() == 2 {
		t.Errorf("Refresh pull did not delete record")
	}
}

func TestGetCollectionPage_WithFieldUpdates(t *testing.T) {
	b := GetTestBackgroundRunner()

	ti := time.Date(2012, time.April, 10, 0, 0, 0, 0, time.UTC)

	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2, Notes: map[int32]string{10: ti.Format("2006-01-02")}})

	_, err := b.ProcessCollectionPage(context.Background(), d, 1, 123)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	record, err := b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	if record.GetRelease().GetRating() != 2 {
		t.Errorf("Stored record is not quite right: %v", record)
	}

	if record.GetLastCleanTime() != ti.UnixNano() {
		t.Errorf("Unable to retrieve clean time: %v (%v vs %v)", record, time.Unix(0, record.GetLastCleanTime()), ti)
	}

	if len(record.GetRelease().GetNotes()) > 0 {
		t.Errorf("Effective hanging notes here: %v", record.GetRelease())
	}
}

func TestGetCollectionPage_NotClobberingDateAdded(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{UserId: 123}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2, Labels: []*pbd.Label{{Name: "testing"}}})
	b.db.SaveRecord(context.Background(), 123, &pb.Record{Release: &pbd.Release{Id: 1, DateAdded: 1234, Labels: []*pbd.Label{{Name: "testing"}}, InstanceId: 100}}, &db.SaveOptions{})

	_, err := b.ProcessCollectionPage(context.Background(), d, 1, 123)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	record, err := b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	if record.GetRelease().GetRating() != 2 {
		t.Errorf("Stored record is not quite right: %v", record)
	}

	if record.GetRelease().GetDateAdded() != 1234 {
		t.Errorf("Date has been clobbered: %v", record)
	}

	if len(record.GetRelease().GetLabels()) != 1 {
		t.Errorf("Overadded the labels: %v", record)
	}
}

func TestGetCollectionPage_WithArrivedUpdates(t *testing.T) {
	b := GetTestBackgroundRunner()

	ti := time.Date(2012, time.April, 10, 0, 0, 0, 0, time.UTC)

	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2, Notes: map[int32]string{10: ti.Format("2006-01-02")}})

	_, err := b.ProcessCollectionPage(context.Background(), d, 1, 123)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	record, err := b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	if record.GetArrived() != ti.UnixNano() {
		t.Errorf("Unable to retrieve arrived time: %v (%v vs %v)", record, time.Unix(0, record.GetArrived()), ti)
	}

	if record.GetLastCleanTime() != 0 {
		t.Errorf("Last clean time should be zero, but is: %v", record.GetLastCleanTime())
	}
}

func TestCleanCollection_SuccessStateTransition(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{UserId: 123}
	
	// Create user with ExpectedCollectionSize = 5
	err := b.db.SaveUser(context.Background(), &pb.StoredUser{
		Auth: &pb.GramophileAuth{Token: "test-token"},
		UserToken: "test-token",
		User: &pbd.User{DiscogsUserId: 123},
		ExpectedCollectionSize: 5,
		State: pb.StoredUser_USER_STATE_REFRESHING,
	})
	if err != nil {
		t.Fatalf("Save user failed: %v", err)
	}

	// Add 5 records
	for i := int64(1); i <= 5; i++ {
		b.db.SaveRecord(context.Background(), 123, &pb.Record{
			Release: &pbd.Release{InstanceId: i},
			RefreshId: 1234,
		}, &db.SaveOptions{})
	}

	err = b.CleanCollection(context.Background(), d, 1234, "test-token", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	user, err := b.db.GetUser(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("Get user failed: %v", err)
	}

	if user.GetState() != pb.StoredUser_USER_STATE_IN_WAITLIST {
		t.Errorf("User state did not transition: %v", user.GetState())
	}
}

func TestCleanCollection_FailureRetryTransition(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{UserId: 123}
	
	err := b.db.SaveUser(context.Background(), &pb.StoredUser{
		Auth: &pb.GramophileAuth{Token: "test-token"},
		UserToken: "test-token",
		User: &pbd.User{DiscogsUserId: 123},
		ExpectedCollectionSize: 5,
		State: pb.StoredUser_USER_STATE_REFRESHING,
	})
	if err != nil {
		t.Fatalf("Save user failed: %v", err)
	}

	// Add 3 records (less than 5)
	for i := int64(1); i <= 3; i++ {
		b.db.SaveRecord(context.Background(), 123, &pb.Record{
			Release: &pbd.Release{InstanceId: i},
			RefreshId: 1234,
		}, &db.SaveOptions{})
	}

	enqueuedRetry := false
	err = b.CleanCollection(context.Background(), d, 1234, "test-token", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		if req.GetElement().GetRefreshCollectionEntry() != nil && req.GetElement().GetRefreshCollectionEntry().GetPage() == 1 {
			enqueuedRetry = true
		}
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if !enqueuedRetry {
		t.Errorf("Retry was not enqueued")
	}

	user, err := b.db.GetUser(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("Get user failed: %v", err)
	}

	if user.GetState() != pb.StoredUser_USER_STATE_REFRESHING {
		t.Errorf("User state should not have transitioned: %v", user.GetState())
	}
}
func TestProcessNotes_PackageField(t *testing.T) {
	b := GetTestBackgroundRunner()
	ctx := context.Background()

	fields := []*pbd.Field{
		{Id: 10, Name: "Package"},
	}

	// 1. Asserts note "4" sets r.PackageScore = 4.
	r1 := &pb.Record{
		Release: &dpb.Release{
			Notes: map[int32]string{10: "4"},
		},
	}
	res1, err := b.processNotes(ctx, fields, r1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.GetPackageScore() != 4 {
		t.Errorf("expected PackageScore to be 4, got %v", res1.GetPackageScore())
	}

	// 1b. Asserts note with whitespace " 4 " sets r.PackageScore = 4.
	r1b := &pb.Record{
		Release: &dpb.Release{
			Notes: map[int32]string{10: " 4 "},
		},
	}
	res1b, err := b.processNotes(ctx, fields, r1b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1b.GetPackageScore() != 4 {
		t.Errorf("expected PackageScore to be 4 for whitespace padded note, got %v", res1b.GetPackageScore())
	}

	// 2. Asserts note "0" sets r.PackageScore = 0.
	r2 := &pb.Record{
		PackageScore: 3,
		Release: &dpb.Release{
			Notes: map[int32]string{10: "0"},
		},
	}
	res2, err := b.processNotes(ctx, fields, r2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.GetPackageScore() != 0 {
		t.Errorf("expected PackageScore to be 0, got %v", res2.GetPackageScore())
	}

	// 3. Asserts non-numeric note ("VG+"), empty string (""), and out-of-bounds notes ("-1", "6") are safely ignored, leaving existing score untouched.
	testCases := []struct {
		val           string
		initialScore  int32
		expectedScore int32
	}{
		{val: "VG+", initialScore: 2, expectedScore: 2},
		{val: "", initialScore: 3, expectedScore: 3},
		{val: "-1", initialScore: 4, expectedScore: 4},
		{val: "6", initialScore: 5, expectedScore: 5},
	}

	for _, tc := range testCases {
		r := &pb.Record{
			PackageScore: tc.initialScore,
			Release: &dpb.Release{
				Notes: map[int32]string{10: tc.val},
			},
		}
		res, err := b.processNotes(ctx, fields, r)
		if err != nil {
			t.Fatalf("unexpected error for val %q: %v", tc.val, err)
		}
		if res.GetPackageScore() != tc.expectedScore {
			t.Errorf("for val %q, expected score %d to be untouched, got %d", tc.val, tc.expectedScore, res.GetPackageScore())
		}
	}
}
