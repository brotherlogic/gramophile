package server

import (
	"context"
	"fmt"

	orglogic "github.com/brotherlogic/gramophile/org"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func (s *Server) SetOrgSnapshot(ctx context.Context, req *pb.SetOrgSnapshotRequest) (*pb.SetOrgSnapshotResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load user: %w", err)
	}

	org, err := s.d.LoadSnapshot(ctx, user, req.GetOrgName(), fmt.Sprintf("%v", req.GetDate()))
	if err != nil {
		return nil, fmt.Errorf("unable to load snapshot: %w", err)
	}

	org.Name = req.GetName()
	err = s.d.SaveSnapshot(ctx, user, req.GetOrgName(), org)
	if err != nil {
		return nil, fmt.Errorf("unable to save snapshot: %w", err)
	}

	return &pb.SetOrgSnapshotResponse{}, nil
}

func (s *Server) GetOrg(ctx context.Context, req *pb.GetOrgRequest) (*pb.GetOrgResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetHash() != "" {
		snapshot, err := s.d.LoadSnapshotHash(ctx, user, req.GetOrgName(), req.GetHash())
		if err != nil {
			return nil, fmt.Errorf("Unable to load snapshot hash: %w", err)
		}

		return &pb.GetOrgResponse{Snapshot: snapshot}, nil
	}

	if req.GetName() != "" {
		snapshot, err := s.d.LoadSnapshot(ctx, user, req.GetOrgName(), req.GetName())
		if err != nil {
			return nil, fmt.Errorf("Unable to load snapshot: %w", err)
		}

		return &pb.GetOrgResponse{Snapshot: snapshot}, nil
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

	org := orglogic.GetOrg(s.d)

	snapshot, err := org.BuildSnapshot(ctx, user, o)
	if err != nil {
		return nil, fmt.Errorf("unable to build snapshot: %w", err)
	}

	latest, err := s.d.GetLatestSnapshot(ctx, user.GetUser().GetDiscogsUserId(), req.GetOrgName())
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

type place struct {
	iid   int64
	unit  int32
	space string
	next  *place
}

func getSnapshotDiff(start, end *pb.OrganisationSnapshot) []*pb.Move {
	mapper := make(map[int32]*pb.Placement)
	for _, place := range start.GetPlacements() {
		mapper[place.GetIndex()] = proto.Clone(place).(*pb.Placement)
	}
	var cplace *place
	for i := int32(len(mapper)); i > 0; i-- {
		nplace := &place{
			iid:   mapper[i].GetIid(),
			unit:  mapper[i].GetUnit(),
			space: mapper[i].GetSpace(),
		}
		if cplace != nil {
			nplace.next = cplace
		}
		cplace = nplace
	}

	emapper := make(map[int32]*pb.Placement)
	for _, place := range end.GetPlacements() {
		emapper[place.GetIndex()] = place
	}

	var moves []*pb.Move
	curr := cplace
	var prev *place
	for index := 1; index <= len(end.GetPlacements()); index++ {
		if curr.iid != emapper[int32(index)].GetIid() {
			// Search forwards and move this record to this slot
			sstart := curr
			cIndex := int32(index)
			for {
				if sstart.iid == emapper[int32(index)].GetIid() {
					moves = append(moves, &pb.Move{
						Start: &pb.Placement{
							Iid:   sstart.iid,
							Space: sstart.space,
							Unit:  sstart.unit,
							Index: cIndex,
						},
						End: &pb.Placement{
							Iid:   sstart.iid,
							Space: curr.space,
							Unit:  curr.unit,
							Index: int32(index),
						},
					})
					if prev != nil {
						prev.next = sstart
						sstart.next = curr
					}
					break
				} else {
					sstart = sstart.next
					cIndex++
				}
			}

		}
	}

	return moves
}
