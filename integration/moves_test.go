package integration

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/server"

	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	pstore_client "github.com/brotherlogic/pstore/client"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestMoveApplied(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{
			&pbd.Folder{Name: "Listening Pile", Id: 123},
			&pbd.Folder{Name: "Limbo", Id: 125},
		},
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Add a record that needs to be moved
	d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{FolderId: 125, InstanceId: 1234},
		Arrived: time.Now().UnixNano(),
	}, &db.SaveOptions{})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			ArrivedConfig: &pb.ArrivedConfig{Mandate: pb.Mandate_REQUIRED},
			MovingConfig: &pb.MovingConfig{
				FormatClassifier: &pb.FormatClassifier{DefaultFormat: "UNKNOWN"},
				Moves: []*pb.RecordMove{
					{
						Classification: []string{"arrived"},
						Format:         []string{"UNKNOWN"},
						Folder:         "Listening Pile",
					},
				},
			},
			ClassificationConfig: &pb.ClassificationConfig{
				Classifiers: []*pb.Classifier{
					{
						ClassifierName: "arrived",
						Classification: "arrived",
						Rules: []*pb.ClassificationRule{
							{
								RuleName: "arrived",
								Selector: &pb.ClassificationRule_IntSelector{
									IntSelector: &pb.IntSelector{
										Name:      "arrived",
										Threshold: 0,
										Comp:      pb.Comparator_COMPARATOR_GREATER_THAN,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}
	qc.FlushQueue(ctx)

	r, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})

	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetRecords()[0].GetRecord() == nil || r.GetRecords()[0].GetRecord().GetRelease().GetFolderId() != 123 {
		t.Errorf("Record was not moved: %v", r.GetRecords()[0].GetRecord())
	}
}

func TestRandomMoveHappensPostIntent(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{
			&pbd.Folder{Name: "Listening Pile", Id: 123},
			&pbd.Folder{Name: "Limbo", Id: 125},
		},
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Add a record that needs to be moved
	d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{FolderId: 125, InstanceId: 1234},
		Arrived: time.Now().UnixNano(),
	}, &db.SaveOptions{})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			ArrivedConfig: &pb.ArrivedConfig{Mandate: pb.Mandate_REQUIRED},
			MovingConfig: &pb.MovingConfig{
				FormatClassifier: &pb.FormatClassifier{DefaultFormat: "UNKNOWN"},
				Moves: []*pb.RecordMove{
					{
						Classification: []string{"arrived"},
						Format:         []string{"UNKNOWN"},
						Folder:         "Listening Pile",
					},
				},
			},
			ClassificationConfig: &pb.ClassificationConfig{
				Classifiers: []*pb.Classifier{
					{
						ClassifierName: "arrived",
						Classification: "arrived",
						Rules: []*pb.ClassificationRule{
							{
								RuleName: "arrived",
								Selector: &pb.ClassificationRule_IntSelector{
									IntSelector: &pb.IntSelector{
										Name:      "arrived",
										Threshold: 0,
										Comp:      pb.Comparator_COMPARATOR_GREATER_THAN,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}
	qc.FlushQueue(ctx)

	r, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})

	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetRecords()[0].GetRecord() == nil || r.GetRecords()[0].GetRecord().GetRelease().GetFolderId() != 123 {
		t.Errorf("Record was not moved: %v", r.GetRecords()[0].GetRecord())
	}
}
