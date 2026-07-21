package server

import (
	"context"
	"fmt"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

func formatRecordTitle(iid int64, rec *pb.Record) string {
	if rec != nil && rec.GetRelease() != nil {
		artists := rec.GetRelease().GetArtists()
		title := rec.GetRelease().GetTitle()
		if len(artists) > 0 && artists[0].GetName() != "" {
			if title != "" {
				return artists[0].GetName() + " - " + title
			}
			return artists[0].GetName()
		}
		if title != "" {
			return title
		}
	}
	return fmt.Sprintf("Unknown (%v)", iid)
}

func (s *Server) LocateRecord(ctx context.Context, req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	records, err := s.d.LoadAllRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	iidToRecord := make(map[int64]*pb.Record)
	var matchingIids []int64
	for _, rec := range records {
		if rec.GetRelease() != nil {
			iidToRecord[rec.GetRelease().GetInstanceId()] = rec
			if rec.GetRelease().GetId() == req.GetReleaseId() {
				matchingIids = append(matchingIids, rec.GetRelease().GetInstanceId())
			}
		}
	}

	if len(matchingIids) == 0 {
		return nil, status.Errorf(codes.NotFound, "release %v not found in user collection", req.GetReleaseId())
	}

	var locations []*pb.Location
	for _, org := range user.GetConfig().GetOrganisationConfig().GetOrganisations() {
		snapshot, err := s.d.GetLatestSnapshot(ctx, user.GetUser().GetDiscogsUserId(), org.GetName())
		if err != nil || snapshot == nil {
			continue // Ignore error if snapshot is not found
		}

		// Ensure placements are sorted by index just in case
		placements := snapshot.GetPlacements()
		sort.Slice(placements, func(i, j int) bool {
			return placements[i].GetIndex() < placements[j].GetIndex()
		})

		for idx, p := range placements {
			for _, iid := range matchingIids {
				if p.GetIid() == iid {
					loc := &pb.Location{
						LocationName: org.GetName(),
						Shelf:        p.GetSpace(),
						Slot:         p.GetUnit(),
						Record:       formatRecordTitle(p.GetIid(), iidToRecord[p.GetIid()]),
					}

					// Get before context (up to 2)
					beforeCount := 0
					for i := idx - 1; i >= 0 && beforeCount < 2; i-- {
						beforeP := placements[i]
						loc.Before = append(loc.Before, &pb.Context{
							Index:  beforeP.GetIndex(),
							Iid:    beforeP.GetIid(),
							Record: formatRecordTitle(beforeP.GetIid(), iidToRecord[beforeP.GetIid()]),
						})
						beforeCount++
					}

					// Get after context (up to 2)
					afterCount := 0
					for i := idx + 1; i < len(placements) && afterCount < 2; i++ {
						afterP := placements[i]
						loc.After = append(loc.After, &pb.Context{
							Index:  afterP.GetIndex(),
							Iid:    afterP.GetIid(),
							Record: formatRecordTitle(afterP.GetIid(), iidToRecord[afterP.GetIid()]),
						})
						afterCount++
					}
					locations = append(locations, loc)
				}
			}
		}
	}

	return &pb.LocateRecordResponse{Locations: locations}, nil
}


