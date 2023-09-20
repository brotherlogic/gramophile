package server

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
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

func getWidth(r *pb.Record, d pb.Density, sleeveMap map[string]*pb.Sleeve) float32 {
	log.Printf("WIDTH: %v", r)
	switch d {
	case pb.Density_COUNT:
		return 1
	case pb.Density_DISKS:
		return 1 //return r.GetDisks()
	case pb.Density_WIDTH:
		return r.GetWidth() * sleeveMap[r.GetSleeve()].GetWidthMultiplier()
	}

	log.Printf("Unknown Width Calculation: %v", d)
	return -1
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

	// Build out the width map
	sleeveMap := make(map[string]*pb.Sleeve)
	for _, sleeve := range user.GetConfig().GetSleeveConfig().GetAllowedSleeves() {
		sleeveMap[sleeve.GetName()] = sleeve
	}

	// Now lay out the records in the units
	var placements []*pb.Placement
	rc := int32(0)
	totalWidth := float32(0)

	for _, slot := range org.GetSpaces() {
		if slot.GetLayout() == pb.Layout_TIGHT {
			for i := int32(1); i <= (slot.GetUnits()); i++ {
				if slot.GetRecordsWidth() > 0 {
					for _, r := range records[rc:min(rc+(slot.GetRecordsWidth()), int32(len(records)))] {
						width := getWidth(r, org.GetDensity(), sleeveMap)

						placements = append(placements, &pb.Placement{
							Iid:   r.GetRelease().GetInstanceId(),
							Space: slot.GetName(),
							Unit:  i,
							Index: rc + 1,
							Width: width,
						})
						rc++
						totalWidth += width
					}
				}
			}
		} else if slot.GetLayout() == pb.Layout_LOOSE {
			if slot.GetRecordsWidth() > 0 {
				count := min(int32(math.Ceil(float64(len(records[rc:]))/float64(slot.GetUnits()))), slot.GetRecordsWidth())
				for i := int32(1); i <= slot.GetUnits(); i++ {
					for _, r := range records[rc : rc+count] {
						width := getWidth(r, org.GetDensity(), sleeveMap)
						log.Printf("Got width: %v", width)
						placements = append(placements, &pb.Placement{
							Iid:   r.GetRelease().GetInstanceId(),
							Space: slot.GetName(),
							Unit:  i,
							Index: rc + 1,
							Width: width,
						})
						rc++
						totalWidth += width
					}
				}
			}
		}
	}

	return &pb.OrganisationSnapshot{
		Hash:       getHash(placements),
		Placements: placements,
		Date:       time.Now().Unix(),
	}, nil
}

func getHash(placements []*pb.Placement) string {
	sort.SliceStable(placements, func(i, j int) bool {
		return placements[i].GetIndex() < placements[j].GetIndex()
	})

	bytes, _ := proto.Marshal(&pb.OrganisationSnapshot{Placements: placements})
	return fmt.Sprintf("%x", sha1.Sum(bytes))
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
