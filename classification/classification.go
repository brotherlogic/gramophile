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
		return ValidateSelector(r.GetBooleanSelector().GetName(), protoreflect.BoolKind)
	case *pb.ClassificationRule_IntSelector:
		return ValidateSelector(r.GetIntSelector().GetName(), protoreflect.Int32Kind)
	}

	return status.Errorf(codes.NotFound, "Validator for %T not found", r.GetSelector())
}

func ValidateSelector(s string, ty protoreflect.Kind) error {
	r := &pb.Record{}
	fields := r.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.TextName() == s {
			if field.Kind() != ty {
				return status.Errorf(codes.FailedPrecondition, "Field %v is not a %v", s, ty)
			}
			return nil
		}
	}

	return status.Errorf(codes.NotFound, "Boolean field %v not found in record proto", s)
}

func ApplyRule(rule *pb.ClassificationRule, record *pb.Record) bool {
	switch rule.GetSelector().(type) {
	case *pb.ClassificationRule_BooleanSelector:
		return ApplyBooleanSelector(rule.GetBooleanSelector(), record)
	case *pb.ClassificationRule_IntSelector:
		return ApplyIntSelector(rule.GetIntSelector(), record)
	}
	return false
}

func ApplyBooleanSelector(b *pb.BooleanSelector, r *pb.Record) bool {
	fields := r.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.TextName() == b.GetName() {
			return r.ProtoReflect().Get(field).Bool()
		}
	}

	return false
}

func compare(v, t int64, comp pb.Comparator) bool {
	switch comp {
	case pb.Comparator_COMPARATOR_GREATER_THAN:
		return v > t
	case pb.Comparator_COMPARATOR_LESS_THAN:
		return v < t
	case pb.Comparator_COMPARATOR_LESS_THAN_OR_EQUALS:
		return v <= t
	case pb.Comparator_COMPARATOR_GREATER_THAN_OR_EQUALS:
		return v >= t
	}

	return false
}

func ApplyIntSelector(is *pb.IntSelector, r *pb.Record) bool {
	fields := r.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.TextName() == is.GetName() {
			val := r.ProtoReflect().Get(field).Int()
			return compare(val, is.GetThreshold(), is.Comp)
		}
	}

	return false
}
