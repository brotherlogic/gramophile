package classification

import (
	"context"
	"testing"
	"time"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

var classificationConfig = &pb.ClassificationConfig{
	Classifiers: []*pb.Classifier{
		{
			ClassifierName: "cleaning pile",
			Classification: "cleaning_pile",
			Priority:       1,
			Rules: []*pb.ClassificationRule{
				{
					Selector: &pb.ClassificationRule_DateSinceSelector{
						DateSinceSelector: &pb.DateSinceSelector{Name: "last_clean_time", Duration: "3y"},
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
}

func TestClassification(t *testing.T) {
	for _, tc := range classificationTestCases {
		classification := Classify(context.Background(), tc.record, classificationConfig, &pb.OrganisationConfig{}, db.NewTestDB(pstore_client.GetTestClient()), 12)
		if classification != tc.result {
			t.Errorf("Failure in %v: expected %v, got %v", tc.name, tc.result, classification)
		}
	}
}
