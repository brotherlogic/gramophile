package classification

import (
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testCases = []struct {
	name   string
	rule   *pb.ClassificationRule
	result codes.Code
}{
	{
		name: "Valid Boolean",
		rule: &pb.ClassificationRule{Selector: &pb.ClassificationRule_BooleanSelector{
			BooleanSelector: &pb.BooleanSelector{Name: "madeupboolean"},
		}},
		result: codes.NotFound,
	},
}

func TestClassifications(t *testing.T) {
	for _, tc := range testCases {
		err := ValidateRule(tc.rule)
		if status.Code(err) != tc.result {
			t.Errorf("Failure in %v: expected %v, got %v (%v)", tc.name, tc.result, status.Code(err), err)
		}
	}
}
