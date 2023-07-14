package server

import (
	"context"
	"crypto/sha1"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (s *Server) GetRecords(ctx context.Context, user *pb.StoredUser) ([]*pb.Record, error) {
	ids, err := s.d.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, fmt.Errorf("Unable to load record ids: %w", err)
	}

	var records []*pb.Record
	for _, id := range ids {
		rec, err := s.d.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), id)
		if err != nil {
			return nil, fmt.Errorf("unable to load record %v -> %w", id, err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (s *Server) getArtistYear(ctx context.Context, r *pb.Record) string {
	return fmt.Sprintf("%v", r.GetRelease().GetInstanceId())
}

func (s *Server) getLabelCatno(ctx context.Context, r *pb.Record) string {
	if len(r.GetRelease().GetLabels()) > 0 {
		return fmt.Sprintf("%v-%v", strings.ToLower(r.GetRelease().GetLabels()[0].GetName()), strings.ToLower(r.GetRelease().GetLabels()[0].GetCatno()))
	}

	return ""
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func (s *Server) buildSnapshot(ctx context.Context, user *pb.StoredUser, org *pb.Organisation) (*pb.OrganisationSnapshot, error) {
	allRecords, err := s.GetRecords(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("unable to load records: %w", err)
	}

	// First sort the records into order
	var records []*pb.Record
	for _, folderset := range org.GetFoldersets() {
		var recs []*pb.Record
		for _, record := range allRecords {
			if record.GetRelease().GetFolderId() == folderset.GetFolder() {
				recs = append(recs, record)
			}
		}

		switch folderset.GetSort() {
		case pb.Sort_ARTIST_YEAR:
			sort.SliceStable(recs, func(i, j int) bool {
				return s.getArtistYear(ctx, recs[i]) < s.getArtistYear(ctx, recs[j])
			})
		case pb.Sort_LABEL_CATNO:
			sort.SliceStable(recs, func(i, j int) bool {
				return s.getLabelCatno(ctx, recs[i]) < s.getLabelCatno(ctx, recs[j])
			})
		}

		records = append(records, recs...)
	}

	// Now lay out the records in the units
	var placements []*pb.Placement
	rc := int32(0)

	for _, slot := range org.GetSpaces() {
		if slot.GetLayout() == pb.Layout_TIGHT {
			for i := int32(1); i <= (slot.GetUnits()); i++ {
				if slot.GetRecordsWidth() > 0 {
					for _, r := range records[rc:min(rc+(slot.GetRecordsWidth()), int32(len(records)))] {
						placements = append(placements, &pb.Placement{
							Iid:   r.GetRelease().GetInstanceId(),
							Space: slot.GetName(),
							Unit:  i,
							Index: rc + 1,
						})
						rc++
					}
				}
			}
		} else if slot.GetLayout() == pb.Layout_LOOSE {
			if slot.GetRecordsWidth() > 0 {
				count := min(int32(math.Ceil(float64(len(records[rc:]))/float64(slot.GetUnits()))), slot.GetRecordsWidth())
				for i := int32(1); i <= slot.GetUnits(); i++ {
					for _, r := range records[rc : rc+count] {
						placements = append(placements, &pb.Placement{
							Iid:   r.GetRelease().GetInstanceId(),
							Space: slot.GetName(),
							Unit:  i,
							Index: rc + 1,
						})
						rc++
					}
				}
			}
		}
	}

	return &pb.OrganisationSnapshot{
		Date:       time.Now().Unix(),
		Placements: placements,
		Hash:       getHash(placements),
	}, nil
}

func getHash(placements []*pb.Placement) string {
	bytes, _ := proto.Marshal(&pb.OrganisationSnapshot{Placements: placements})
	return fmt.Sprintf("%x", sha1.Sum(bytes))
}

func getSnapshotDiff(start, end *pb.OrganisationSnapshot) []*pb.Move {
	return []*pb.Move{}
}

func (s *Server) GetOrg(ctx context.Context, req *pb.GetOrgRequest) (*pb.GetOrgResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load user: %w", err)
	}

	var o *pb.Organisation
	for _, org := range user.GetConfig().GetOrganisationConfig().GetOrganisations() {
		if org.GetName() == req.GetOrgName() {
			o = org
		}
	}

	if o == nil {
		return nil, status.Errorf(codes.NotFound, "unable to locate org called %v", req.GetOrgName())
	}

	snapshot, err := s.buildSnapshot(ctx, user, o)
	if err != nil {
		return nil, fmt.Errorf("unable to build snapshot: %w", err)
	}

	latest, err := s.d.GetLatestSnapshot(ctx, user, req.GetOrgName())
	if err != nil && status.Code(err) != codes.NotFound {
		return nil, fmt.Errorf("unable to load previous snapshot: %w", err)
	}

	if latest == nil || latest.GetHash() != snapshot.GetHash() {
		err = s.d.SaveSnapshot(ctx, user, req.GetOrgName(), snapshot)
		if err != nil {
			return nil, fmt.Errorf("unable to save new snapshot: %w", err)
		}
	}

	return &pb.GetOrgResponse{Snapshot: snapshot}, nil
}
