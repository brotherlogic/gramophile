package classification

import (
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func ValidateRule(r *pb.ClassificationRule) error {

	switch r.GetSelector().(type) {
	case *pb.ClassificationRule_BooleanSelector:
		return ValidateBooleanSelector(r.GetBooleanSelector())
	}

	return status.Errorf(codes.NotFound, "Validator for %T not found", r.GetSelector())
}

func ValidateBooleanSelector(s *pb.BooleanSelector) error {
	r := &pb.Record{}
	fields := r.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.TextName() == s.GetName() {
			if field.Kind() != protoreflect.BoolKind {
				return status.Errorf(codes.FailedPrecondition, "Field %v is not a boolean", s.GetName())
			}
			return nil
		}
	}

	return status.Errorf(codes.NotFound, "Boolean field %v not found in record proto", s.GetName())
}
