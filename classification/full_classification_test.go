package classification

import (
	"testing"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
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
		classification := Classify(tc.record, classificationConfig)
		if classification != tc.result {
			t.Errorf("Failure in %v: expected %v, got %v", tc.name, tc.result, classification)
		}
	}
}
