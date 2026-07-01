package server

import (
	"context"
	"sort"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) LocateRecord(ctx context.Context, req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	records, err := s.d.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	var iids []int64
	for _, recId := range records {
		rec, err := s.d.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), recId)
		if err != nil {
			return nil, err
		}
		if rec.GetRelease().GetId() == req.GetReleaseId() {
			iids = append(iids, rec.GetRelease().GetInstanceId())
		}
	}

	var locations []*pb.Location
	for _, org := range user.GetConfig().GetOrganisationConfig().GetOrganisations() {
		snapshot, err := s.d.GetLatestSnapshot(ctx, user.GetUser().GetDiscogsUserId(), org.GetName())
		if err != nil {
			continue // Ignore error if snapshot is not found
		}
		if snapshot == nil {
			continue
		}

		// Ensure placements are sorted by index just in case
		placements := snapshot.GetPlacements()
		sort.Slice(placements, func(i, j int) bool {
			return placements[i].GetIndex() < placements[j].GetIndex()
		})

		for idx, p := range placements {
			for _, iid := range iids {
				if p.GetIid() == iid {
					loc := &pb.Location{
						LocationName: org.GetName(),
						Shelf:        p.GetSpace(),
						Slot:         p.GetUnit(),
					}

					// Get before context (up to 2)
					beforeCount := 0
					for i := idx - 1; i >= 0 && beforeCount < 2; i-- {
						beforeP := placements[i]
						loc.Before = append(loc.Before, &pb.Context{
							Index: beforeP.GetIndex(),
							Iid:   beforeP.GetIid(),
						})
						beforeCount++
					}

					// Get after context (up to 2)
					afterCount := 0
					for i := idx + 1; i < len(placements) && afterCount < 2; i++ {
						afterP := placements[i]
						loc.After = append(loc.After, &pb.Context{
							Index: afterP.GetIndex(),
							Iid:   afterP.GetIid(),
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
