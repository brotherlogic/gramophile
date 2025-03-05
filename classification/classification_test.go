package classification

import (
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var validationTestCases = []struct {
	name   string
	rule   *pb.ClassificationRule
	result codes.Code
}{
	{
		name: "Invalid Boolean",
		rule: &pb.ClassificationRule{Selector: &pb.ClassificationRule_BooleanSelector{
			BooleanSelector: &pb.BooleanSelector{Name: "madeupboolean"},
		}},
		result: codes.NotFound,
	},
	{
		name: "Valid Int",
		rule: &pb.ClassificationRule{Selector: &pb.ClassificationRule_BooleanSelector{
			BooleanSelector: &pb.BooleanSelector{Name: "num_plays"},
		}},
		result: codes.OK,
	},
	{
		name: "Invalid Int",
		rule: &pb.ClassificationRule{Selector: &pb.ClassificationRule_BooleanSelector{
			BooleanSelector: &pb.BooleanSelector{Name: "made_up_int"},
		}},
		result: codes.NotFound,
	},
}

func TestClassificationValidation(t *testing.T) {
	for _, tc := range validationTestCases {
		err := ValidateRule(tc.rule)
		if status.Code(err) != tc.result {
			t.Errorf("Failure in %v: expected %v, got %v (%v)", tc.name, tc.result, status.Code(err), err)
		}
	}
}
