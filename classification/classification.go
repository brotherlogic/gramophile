package classification

import (
	"context"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func Classify(ctx context.Context, r *pb.Record, config *pb.ClassificationConfig, org *pb.OrganisationConfig, db db.Database, uid int32) string {
	rules := config.GetClassifiers()
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].GetPriority() < rules[j].GetPriority()
	})
	for _, ruleSet := range rules {
		pass := true
		for _, rule := range ruleSet.GetRules() {
			if !ApplyRule(ctx, rule, r, db, uid) {
				pass = false
				continue
			}
		}
		if pass {
			return ruleSet.GetClassification()
		}
	}

	return ""
}

func validateDuration(d string) error {
	_, err := time.ParseDuration(d)
	if err != nil {
		// We also support days and years
		if d[len(d)-1] == 'd' || d[len(d)-1] == 'y' {
			_, err := strconv.ParseInt(d[:len(d)-1], 10, 64)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "unable to parse duration: %v", err)
			}
			return nil
		}

		return status.Errorf(codes.InvalidArgument, "Unable to parse duration: %v", err)
	}
	return nil
}

func validateLocation(l string, org *pb.OrganisationConfig) error {
	for _, o := range org.GetOrganisations() {
		if o.GetName() == l {
			return nil
		}
	}

	return status.Errorf(codes.NotFound, "Unable to find location called %v", l)
}

func ValidateRule(r *pb.ClassificationRule, org *pb.OrganisationConfig) error {

	switch r.GetSelector().(type) {
	case *pb.ClassificationRule_BooleanSelector:
		return ValidateSelector(r.GetBooleanSelector().GetName(), protoreflect.BoolKind)
	case *pb.ClassificationRule_IntSelector:
		return ValidateSelector(r.GetIntSelector().GetName(), protoreflect.Int32Kind)
	case *pb.ClassificationRule_DateSinceSelector:
		if err := ValidateSelector(r.GetDateSinceSelector().GetName(), protoreflect.Int64Kind); err == nil {
			return validateDuration(r.GetDateSinceSelector().GetDuration())
		} else {
			return err
		}
	case *pb.ClassificationRule_LocationSelector:
		return validateLocation(r.GetLocationSelector().GetLocation(), org)
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

func ApplyRule(ctx context.Context, rule *pb.ClassificationRule, record *pb.Record, db db.Database, uid int32) bool {
	switch rule.GetSelector().(type) {
	case *pb.ClassificationRule_BooleanSelector:
		return ApplyBooleanSelector(rule.GetBooleanSelector(), record)
	case *pb.ClassificationRule_IntSelector:
		return ApplyIntSelector(rule.GetIntSelector(), record)
	case *pb.ClassificationRule_DateSinceSelector:
		return ApplyDateSinceSelector(rule.GetDateSinceSelector(), record)
	case *pb.ClassificationRule_LocationSelector:
		return ApplyLocationSelector(ctx, rule.GetLocationSelector().GetLocation(), record, db, uid)
	}
	return false
}

func ApplyLocationSelector(ctx context.Context, l string, record *pb.Record, db db.Database, userid int32) bool {
	org, err := db.GetLatestSnapshot(ctx, userid, l)
	if err != nil {
		log.Printf("Unable to get latest snapshot for %v -> %v", l, err)
	}

	log.Printf("Placements: %v", org.GetPlacements())
	for _, placement := range org.GetPlacements() {
		if placement.GetIid() == record.GetRelease().GetInstanceId() {
			return true
		}
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

func ApplyDateSinceSelector(d *pb.DateSinceSelector, r *pb.Record) bool {
	fields := r.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.TextName() == d.GetName() {
			dv := r.ProtoReflect().Get(field).Int()
			dt := time.Unix(0, dv)
			du, err := time.ParseDuration(d.GetDuration())
			if err != nil {
				// We also support days and years
				if d.GetDuration()[len(d.GetDuration())-1] == 'd' || d.GetDuration()[len(d.GetDuration())-1] == 'y' {
					value, err := strconv.ParseInt(d.GetDuration()[:len(d.GetDuration())-1], 10, 64)
					if err != nil {
						return false
					}
					if d.GetDuration()[len(d.GetDuration())-1] == 'd' {
						du = time.Hour * 24 * time.Duration(value)
					} else {
						du = time.Hour * 24 * 365 * time.Duration(value)
					}
				}
			}
			log.Printf("HERE %v and %v -> %v but %v", time.Now(), dt, du, time.Since(dt))
			return time.Since(dt) > du
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
