package org

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/protobuf/proto"
)

func GetOrg(d db.Database) *Org {
	return &Org{d: d}
}

type Org struct {
	d db.Database
}

type groupingElement struct {
	records []*pb.Record
	id      int64
}

type sortingElement struct {
	record *pb.Record
	sort   pb.Sort
}

func (o *Org) getLabel(ctx context.Context, r *pb.Record, c *pb.Organisation, ws []*pb.LabelWeight) string {
	// Release has no labels
	if len(r.GetRelease().GetLabels()) == 0 {
		return ""
	}

	bestWeight := float32(0.5)
	bestLabel := r.GetRelease().GetLabels()[0]

	for _, label := range r.GetRelease().GetLabels()[1:] {
		for _, weight := range ws {
			if weight.GetLabelId() == label.GetId() && (weight.GetWeight()) > bestWeight {
				bestLabel = label
				bestWeight = weight.GetWeight()
			}
		}
	}

	return strings.ToLower(bestLabel.GetName())
}

func (o *Org) getArtistYear(ctx context.Context, r *pb.Record) string {
	return fmt.Sprintf("%v", r.GetRelease().GetInstanceId())
}

func (o *Org) getLabelCatno(ctx context.Context, r *pb.Record, c *pb.Organisation, ws []*pb.LabelWeight) string {
	// Release has no labels
	if len(r.GetRelease().GetLabels()) == 0 {
		return ""
	}

	// Sort labels alphabetically
	labels := r.GetRelease().GetLabels()
	sort.SliceStable(labels, func(i, j int) bool {
		return strings.ToLower(labels[i].GetName()) < strings.ToLower(labels[j].GetName())
	})

	bestWeight := float32(0.5)
	bestLabel := r.GetRelease().GetLabels()[0]

	for _, label := range r.GetRelease().GetLabels()[1:] {
		for _, weight := range ws {
			if weight.GetLabelId() == label.GetId() && (weight.GetWeight()) > bestWeight {
				bestLabel = label
				bestWeight = weight.GetWeight()
			}
		}
	}

	return strings.ToLower(bestLabel.GetName() + "-" + bestLabel.GetCatno())
}

func (o *Org) getRecords(ctx context.Context, user *pb.StoredUser) ([]*pb.Record, error) {
	ids, err := o.d.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, fmt.Errorf("Unable to load record ids: %w", err)
	}

	var records []*pb.Record
	for _, id := range ids {
		rec, err := o.d.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), id)
		if err != nil {
			return nil, fmt.Errorf("unable to load record %v -> %w", id, err)
		}
		records = append(records, rec)
	}

	return records, nil
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

func getHash(placements []*pb.Placement) string {
	sort.SliceStable(placements, func(i, j int) bool {
		return placements[i].GetIndex() < placements[j].GetIndex()
	})

	val := &pb.OrganisationSnapshot{Placements: placements}
	bytes, _ := proto.Marshal(val)
	log.Printf("HASH %v -> %x", val, (sha1.Sum(bytes)))
	return fmt.Sprintf("%x", sha1.Sum(bytes))
}

func (o *Org) BuildSnapshot(ctx context.Context, user *pb.StoredUser, org *pb.Organisation, c *pb.OrganisationConfig) (*pb.OrganisationSnapshot, error) {
	log.Printf("Building Snapshot for %v", org)
	allRecords, err := o.getRecords(ctx, user)
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
		case pb.Sort_ADDITION_DATE:
			sort.SliceStable(recs, func(i, j int) bool {
				return recs[i].record.GetRelease().GetDateAdded() < recs[j].record.GetRelease().GetDateAdded()
			})
		case pb.Sort_ARTIST_YEAR:
			sort.SliceStable(recs, func(i, j int) bool {
				return o.getArtistYear(ctx, recs[i].record) < o.getArtistYear(ctx, recs[j].record)
			})
		case pb.Sort_LABEL_CATNO:
			sort.SliceStable(recs, func(i, j int) bool {
				return o.getLabelCatno(ctx, recs[i].record, org, c.GetLabelRanking()) < o.getLabelCatno(ctx, recs[j].record, org, c.GetLabelRanking())
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
				if o.getLabel(ctx, r.record, org, c.GetLabelRanking()) == currLabel {
					currElement.records = append(currElement.records, r.record)
				} else {
					if len(currElement.records) > 0 {
						ordList = append(ordList, currElement)
					}
					currLabel = o.getLabel(ctx, r.record, org, c.GetLabelRanking())
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
				edge = min(edge, len(ordList))
				log.Printf("Looking ahead: %v -> %v", i+1, edge)
				for _, tr := range ordList[i+1 : edge] {
					if tr != nil {
					width := getWidth(tr, org.GetDensity(), sleeveMap, defaultWidth)
					log.Printf("Trying %v", width)
					if currSlotWidth+width <= org.GetSpaces()[currSlot].GetWidth() {
						str := fmt.Sprintf("Placing %v -> %v + %v / %v", tr.id, currSlotWidth, width, org.GetSpaces()[currSlot].GetWidth())
						placed[tr.id] = true
						placements = append(placements, &pb.Placement{
							Iid:           tr.id,
							Space:         org.GetSpaces()[currSlot].GetName(),
							Unit:          currUnit,
							Index:         index + 1,
							Width:         width,
							OriginalIndex: int32(ogMap[tr.id]),
							Observations:  str,
						})
						index += int32(len(tr.records))
						currSlotWidth += width
					}
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
			Observations:  "regular",
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
				Observations:  entry.GetObservations(),
			})
		}
	}

	return &pb.OrganisationSnapshot{
		Hash:       getHash(nplacements),
		Placements: nplacements,
		Date:       time.Now().UnixNano(),
	}, nil
}
