package server

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"math"
	"sort"
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

func (s *Server) getLabelCatno(ctx context.Context, r *pb.Record, c *pb.Organisation) string {
	// Release has no labels
	if len(r.GetRelease().GetLabels()) == 0 {
		return ""
	}

	bestWeight := float32(0.5)
	bestLabel := r.GetRelease().GetLabels()[0]

	for _, label := range r.GetRelease().GetLabels()[1:] {
		for _, weight := range c.GetGrouping().GetLabelWeights() {
			if weight.GetLabelId() == label.GetId() && (weight.GetWeight()) > bestWeight {
				bestLabel = label
				bestWeight = weight.GetWeight()
			}
		}
	}

	return bestLabel.GetName() + "-" + bestLabel.GetCatno()
}

func (s *Server) getLabel(ctx context.Context, r *pb.Record, c *pb.Organisation) string {
	// Release has no labels
	if len(r.GetRelease().GetLabels()) == 0 {
		return ""
	}

	bestWeight := float32(0.5)
	bestLabel := r.GetRelease().GetLabels()[0]

	for _, label := range r.GetRelease().GetLabels()[1:] {
		for _, weight := range c.GetGrouping().GetLabelWeights() {
			if weight.GetLabelId() == label.GetId() && (weight.GetWeight()) > bestWeight {
				bestLabel = label
				bestWeight = weight.GetWeight()
			}
		}
	}

	return bestLabel.GetName()
}

func oneIfZero(in float32) float32 {
	if in == 0 {
		return 1
	}
	return in
}

func getWidth(r *groupingElement, d pb.Density, sleeveMap map[string]*pb.Sleeve, defaultWidth float32) float32 {
	log.Printf("PROC %+v", r)
	switch d {
	case pb.Density_COUNT:
		return float32(len(r.records))
	case pb.Density_DISKS:
		return float32(len(r.records))
	case pb.Density_WIDTH:
		twidth := float32(0)
		for _, r := range r.records {
			if r.GetWidth() == 0 {
				twidth += defaultWidth * oneIfZero(sleeveMap[r.GetSleeve()].GetWidthMultiplier())
			} else {
				twidth += r.GetWidth() * oneIfZero(sleeveMap[r.GetSleeve()].GetWidthMultiplier())
			}
		}
		return twidth
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

type groupingElement struct {
	records []*pb.Record
	id      int64
}

type sortingElement struct {
	record *pb.Record
	sort   pb.Sort
}

func (s *Server) buildSnapshot(ctx context.Context, user *pb.StoredUser, org *pb.Organisation) (*pb.OrganisationSnapshot, error) {
	allRecords, err := s.GetRecords(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("unable to load records: %w", err)
	}

	// First sort the records into order
	var records []*sortingElement
	for _, folderset := range org.GetFoldersets() {
		var recs []*sortingElement
		for _, record := range allRecords {
			if record.GetRelease().GetFolderId() == folderset.GetFolder() {
				recs = append(recs, &sortingElement{record: record, sort: folderset.GetSort()})
			}
		}

		switch folderset.GetSort() {
		case pb.Sort_ARTIST_YEAR:
			sort.SliceStable(recs, func(i, j int) bool {
				return s.getArtistYear(ctx, recs[i].record) < s.getArtistYear(ctx, recs[j].record)
			})
		case pb.Sort_LABEL_CATNO:
			sort.SliceStable(recs, func(i, j int) bool {
				return s.getLabelCatno(ctx, recs[i].record, org) < s.getLabelCatno(ctx, recs[j].record, org)
			})
		case pb.Sort_RELEASE_YEAR:
			sort.SliceStable(recs, func(i, j int) bool {
				return recs[i].record.GetRelease().GetReleaseDate() < recs[j].record.GetRelease().GetReleaseDate()
			})
		case pb.Sort_EARLIEST_RELEASE_YEAR:
			sort.SliceStable(recs, func(i, j int) bool {
				if recs[i].record.GetEarliestReleaseDate() == recs[j].record.GetEarliestReleaseDate() {
					return recs[i].record.GetRelease().GetReleaseDate() < recs[j].record.GetRelease().GetReleaseDate()
				}
				return recs[i].record.GetEarliestReleaseDate() < recs[j].record.GetEarliestReleaseDate()
			})

		}

		records = append(records, recs...)
	}

	ogMap := make(map[int64]int)
	for i, r := range records {
		log.Printf("POST SORT %v -> %v", i, r.record.GetRelease().GetInstanceId())
		ogMap[r.record.GetRelease().GetInstanceId()] = i
	}

	// Build out the width map
	sleeveMap := make(map[string]*pb.Sleeve)
	for _, sleeve := range user.GetConfig().GetSleeveConfig().GetAllowedSleeves() {
		sleeveMap[sleeve.GetName()] = sleeve
	}

	// Now lay out the records in the units
	var placements []*pb.Placement

	currSlot := 0
	currSlotWidth := float32(0)
	index := int32(0)
	currUnit := int32(1)

	// Add an infinite spill slot to the end of the slots
	org.Spaces = append(org.Spaces, &pb.Space{
		Name:  "Spill",
		Width: math.MaxFloat32,
		Units: 1,
	})

	var ordList []*groupingElement
	if org.GetGrouping().GetType() == pb.GroupingType_GROUPING_GROUP {
		currLabel := ""
		currElement := &groupingElement{records: make([]*pb.Record, 0)}
		for _, r := range records {
			if r.sort == pb.Sort_LABEL_CATNO {
				if s.getLabel(ctx, r.record, org) == currLabel {
					currElement.records = append(currElement.records, r.record)
				} else {
					if len(currElement.records) > 0 {
						ordList = append(ordList, currElement)
					}
					currLabel = s.getLabel(ctx, r.record, org)
					currElement = &groupingElement{records: []*pb.Record{r.record}, id: time.Now().UnixNano()}
				}
			} else {
				if len(currElement.records) > 0 {
					ordList = append(ordList, currElement)
					currElement = &groupingElement{records: make([]*pb.Record, 0)}
				}
				ordList = append(ordList, &groupingElement{records: []*pb.Record{r.record}, id: time.Now().UnixNano()})
			}
		}

		if len(currElement.records) > 0 {
			ordList = append(ordList, currElement)
		}
	} else {
		for _, r := range records {
			ordList = append(ordList, &groupingElement{
				records: []*pb.Record{r.record},
				id:      time.Now().UnixNano(),
			})
		}
	}

	defaultWidth := float32(0.0)
	defaultCount := float32(0.0)
	for _, element := range ordList {
		for _, r := range element.records {
			defaultWidth += r.GetWidth()
			if r.GetWidth() > 0 {
				defaultCount++
			}
		}
	}
	if defaultCount > 0 && org.GetMissingWidthHandling() == pb.MissingWidthHandling_MISSING_WIDTH_AVERAGE {
		defaultWidth /= defaultCount
	}

	// Let's run a check that these groups will actually fit on the shelves
	var nordList []*groupingElement
	for _, element := range ordList {
		fits := false
		for _, shelves := range org.GetSpaces() {
			if getWidth(element, org.GetDensity(), sleeveMap, defaultWidth) <= shelves.GetWidth() && shelves.GetName() != "Spill" {
				fits = true
			}
		}

		log.Printf("%v fits %v", element, fits)

		// Break out the group if it can't possible fit anywhere
		if !fits {
			for _, r := range element.records {
				nordList = append(nordList, &groupingElement{records: []*pb.Record{r}, id: r.GetRelease().GetInstanceId()})
			}
		} else {
			nordList = append(nordList, element)
		}
	}
	ordList = nordList

	ordMap := make(map[int64]*groupingElement)
	for _, entry := range ordList {
		log.Printf("ENTRY: %+v", entry)
		ordMap[entry.id] = entry
	}

	placed := make(map[int64]bool)
	i := 0
	for i < len(ordList) {
		log.Printf("Running %v", i)
		r := ordList[i]
		tripped := false

		// Skip records we've already placed
		if _, ok := placed[r.id]; ok {
			i++
			continue
		}

		width := getWidth(r, org.GetDensity(), sleeveMap, defaultWidth)

		log.Printf("Current %v with %v taken from %v", width, currSlotWidth, org.GetSpaces()[currSlot].GetWidth())

		// Current slot is full - move on to the next unit
		if currSlotWidth+width > org.GetSpaces()[currSlot].GetWidth() {
			log.Printf("Slot is full")

			if org.GetSpill().GetType() == pb.GroupSpill_SPILL_BREAK_ORDERING {
				// Fill out the slot
				edge := i + 1 + int(org.GetSpill().GetLookAhead())
				if org.GetSpill().GetLookAhead() < 0 {
					edge = len(ordList)
				}
				log.Printf("Looking ahead: %v -> %v", i+1, edge)
				for _, tr := range ordList[i+1 : edge] {
					width := getWidth(tr, org.GetDensity(), sleeveMap, defaultWidth)
					log.Printf("Trying %v", width)
					if currSlotWidth+width <= org.GetSpaces()[currSlot].GetWidth() {
						placed[tr.id] = true
						placements = append(placements, &pb.Placement{
							Iid:           tr.id,
							Space:         org.GetSpaces()[currSlot].GetName(),
							Unit:          currUnit,
							Index:         index + 1,
							Width:         width,
							OriginalIndex: int32(ogMap[tr.id]),
						})
						index += int32(len(tr.records))
						currSlotWidth += width
					}
				}
			}

			tripped = true
			currUnit++
			currSlotWidth = 0
		}

		if currUnit > org.GetSpaces()[currSlot].GetUnits() {
			currUnit = 1
			currSlot++
			currSlotWidth = 0

			// If we're starting a new slot, let's start afresh
			tripped = true
		}

		// If we've moved to a new slot or a new unit, let's start afresh to avoid misplacing
		if tripped {
			continue
		}

		placed[r.id] = true
		placements = append(placements, &pb.Placement{
			Iid:           r.id,
			Space:         org.GetSpaces()[currSlot].GetName(),
			Unit:          currUnit,
			Index:         index + 1,
			Width:         width,
			OriginalIndex: int32(ogMap[r.id]),
		})
		index += int32(len(r.records))
		currSlotWidth += width
		i++
	}

	log.Printf("PLACEMENTS %v", len(placements))

	// Expand the placements
	var nplacements []*pb.Placement
	for _, entry := range placements {
		log.Printf("EXPANDING: %v -> %+v", entry, ordMap[entry.GetIid()])
		for ri, r := range ordMap[entry.GetIid()].records {
			nplacements = append(nplacements, &pb.Placement{
				Iid:           r.GetRelease().GetInstanceId(),
				Space:         entry.GetSpace(),
				Unit:          entry.GetUnit(),
				Index:         entry.GetIndex() + int32(ri),
				Width:         getWidth(&groupingElement{records: []*pb.Record{r}}, org.GetDensity(), sleeveMap, defaultWidth),
				OriginalIndex: int32(ogMap[r.GetRelease().GetInstanceId()]),
			})
		}
	}

	return &pb.OrganisationSnapshot{
		Hash:       getHash(nplacements),
		Placements: nplacements,
		Date:       time.Now().UnixNano(),
	}, nil
}

func getHash(placements []*pb.Placement) string {
	sort.SliceStable(placements, func(i, j int) bool {
		return placements[i].GetIndex() < placements[j].GetIndex()
	})

	val := &pb.OrganisationSnapshot{Placements: placements}
	bytes, _ := proto.Marshal(val)
	log.Printf("HASH %v -> %x", val, (sha1.Sum(bytes)))
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
