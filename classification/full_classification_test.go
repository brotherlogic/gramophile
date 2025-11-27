package classification

import (
	"context"
	"testing"
	"time"

	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/org"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	pstore_client "github.com/brotherlogic/pstore/client"
)

var orgConfig = &pb.OrganisationConfig{
	Organisations: []*pb.Organisation{
		{
			Name: "Listening Pile",
			Foldersets: []*pb.FolderSet{
				{
					Folder: 123,
				},
			},
		},
	},
}

var classificationConfig = &pb.ClassificationConfig{
	Classifiers: []*pb.Classifier{
		{
			ClassifierName: "cleaning pile",
			Classification: "cleaning_pile",
			Priority:       2,
			Rules: []*pb.ClassificationRule{
				{
					Selector: &pb.ClassificationRule_DateSinceSelector{
						DateSinceSelector: &pb.DateSinceSelector{Name: "last_clean_time", Duration: "3y"},
					},
				},
			},
		},
		{
			ClassifierName: "deep_cleaning pile",
			Classification: "deep_cleaning_pile",
			Priority:       1,
			Rules: []*pb.ClassificationRule{
				{
					Selector: &pb.ClassificationRule_DateSinceSelector{
						DateSinceSelector: &pb.DateSinceSelector{Name: "last_clean_time", Duration: "3y"},
					},
				},
				{
					Selector: &pb.ClassificationRule_LocationSelector{
						LocationSelector: &pb.LocationSelector{Location: "Listening Pile"},
					},
				},
			},
		},
	},
}

var classificationTestCases = []struct {
	name   string
	record *pb.Record
	result string
}{
	{
		name:   "Needs Clean",
		record: &pb.Record{LastCleanTime: time.Now().Add(-time.Hour * 24 * 365 * 5).UnixNano()},
		result: "cleaning_pile",
	},
	{
		name:   "Does not need Clean",
		record: &pb.Record{LastCleanTime: time.Now().UnixNano()},
		result: "",
	},
	{
		name:   "Needs Clean and Ready To Clean",
		record: &pb.Record{LastCleanTime: time.Now().Add(-time.Hour * 24 * 365 * 5).UnixNano(), Release: &pbd.Release{FolderId: 123}},
		result: "deep_cleaning_pile",
	},
}

func TestClassification(t *testing.T) {
	for _, tc := range classificationTestCases {
		db := db.NewTestDB(pstore_client.GetTestClient())
		lpOrg := &pb.Organisation{
			Name: "Listening Pile",
			Foldersets: []*pb.FolderSet{
				{
					Folder: 123,
				},
			},
		}
		orgConfig := &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				lpOrg,
			},
		}
		user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 12}}
		db.SaveRecord(context.Background(), 12, tc.record)

		o, _ := org.GetOrg(db)
		snap, err := o.BuildSnapshot(context.Background(), user, lpOrg, orgConfig)
		if err != nil {
			t.Fatalf("Bad org: %v", err)
		}
		db.SaveSnapshot(context.Background(), user, "Listening Pile", snap)

		classification := Classify(context.Background(), tc.record, classificationConfig, orgConfig, db, 12)
		if classification != tc.result {
			t.Errorf("Failure in %v: expected %v, got %v", tc.name, tc.result, classification)
		}
	}
}
