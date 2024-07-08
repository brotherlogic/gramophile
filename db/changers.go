package db

import (
	"context"
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MoveChanger struct {
	d Database
}

func (m *MoveChanger) buildLocation(ctx context.Context, org *pb.Organisation, s *pb.OrganisationSnapshot, index int32, nc int32) *pb.Location {
	var before []*pb.Context
	var after []*pb.Context

	for i := index - 1; i > max(0, index-nc); i-- {
		before = append(before, &pb.Context{
			Index: i,
			Iid:   s.GetPlacements()[i].GetIid(),
		})
	}

	for i := index + 1; i < min(int32(len(s.GetPlacements())), index+nc); i++ {
		after = append(after, &pb.Context{
			Index: i,
			Iid:   s.GetPlacements()[i].GetIid(),
		})
	}

	return &pb.Location{
		LocationName: org.GetName(),
		Before:       before,
		After:        after,
	}
}

func (m *MoveChanger) Name() string {
	return "move_changer"
}

func (m *MoveChanger) getLocation(ctx context.Context, userId int32, r *pb.Record, config *pb.GramophileConfig) (*pb.Location, error) {
	for _, org := range config.GetOrganisationConfig().GetOrganisations() {
		found := false
		for _, folder := range org.GetFoldersets() {
			if folder.GetFolder() == r.GetRelease().GetFolderId() {
				found = true
			}
		}

		if found {
			snapshot, err := m.d.GetLatestSnapshot(ctx, userId, org.GetName())
			if err != nil {
				return nil, err
			}

			index := -1
			for i, val := range snapshot.GetPlacements() {
				if val.GetIid() == r.GetRelease().GetInstanceId() {
					index = i
					break
				}
			}

			if index < 0 {
				return nil, status.Errorf(codes.Internal, "Record %v is listed to be in %v but does not appear in latest snapshot", r.GetRelease().GetInstanceId(), org.GetName())
			}

			return m.buildLocation(ctx, org, snapshot, int32(index), config.GetPrintMoveConfig().GetContext()), nil
		}
	}

	return nil, status.Errorf(codes.FailedPrecondition, "Unable to locate %v in an org", r.GetRelease().GetInstanceId())
}

func (m *MoveChanger) ProcessChange(ctx context.Context, c *pb.DBChange, config *pb.GramophileConfig) error {
	// We only care about this change if it's a change record
	if c.GetType() != pb.DBChange_CHANGE_RECORD {
		return nil
	}

	// We only care about records that have moved folders
	if c.GetOldRecord().GetRelease().GetFolderId() == c.GetNewRecord().GetRelease().GetFolderId() {
		return nil
	}

	oldLoc, err := m.getLocation(ctx, c.GetUserId(), c.GetOldRecord(), config)
	if err != nil {
		return err
	}
	newLoc, err := m.getLocation(ctx, c.GetUserId(), c.GetNewRecord(), config)
	if err != nil {
		return err
	}

	// This is just for out of location moves
	if oldLoc.GetLocationName() == newLoc.GetLocationName() {
		return nil
	}

	return m.d.SavePrintMove(ctx, c.GetUserId(), &pb.PrintMove{
		Timestamp:   time.Now().UnixNano(),
		Iid:         c.GetOldRecord().GetRelease().GetInstanceId(),
		Origin:      oldLoc,
		Destination: newLoc,
		Record:      fmt.Sprintf("%v - %v", c.GetOldRecord().GetRelease().GetArtists()[0].GetName(), c.GetOldRecord().GetRelease().GetTitle()),
	})
}
