package server

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		}

		records = append(records, recs...)
	}

	// Now lay out the records in the units
	var placements []*pb.Placement
	rc := int32(0)
	for _, slot := range org.GetSpaces() {
		for i := int32(1); i <= (slot.GetUnits()); i++ {
			if slot.GetRecordsWidth() > 0 {
				for _, r := range records[rc:min(rc+(slot.GetRecordsWidth()), int32(len(records)))] {
					placements = append(placements, &pb.Placement{
						Iid:   r.GetRelease().GetInstanceId(),
						Space: slot.GetName(),
						Unit:  i,
						Index: rc,
					})
					rc++
				}
			}
		}
	}

	return &pb.OrganisationSnapshot{
		Date:       time.Now().Unix(),
		Placements: placements,
	}, nil
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
		return nil, fmt.Errorf("Unable to build snapshot: %w", err)
	}

	return &pb.GetOrgResponse{Snapshot: snapshot}, nil
}
