package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
)

func convertList(list *pb.StoredWantlist) *pb.Wantlist {
	return &pb.Wantlist{
		Name:      list.GetName(),
		StartDate: list.GetStartDate(),
		EndDate:   list.GetEndDate(),
		Type:      list.GetType(),
		Active:    true,
	}
}

func (s *Server) handleWantslists(ctx context.Context, u *pb.StoredUser, lists []*pb.StoredWantlist) error {
	log.Printf("HANDLE HERE: %v", lists)

	savedLists, err := s.d.GetWantlists(ctx, u.GetUser().GetDiscogsUserId())
	if err != nil {
		return err
	}

	for _, list := range lists {
		// Set index on these if we need to
		foundOne := false
		for _, entry := range list.GetEntries() {
			if entry.GetIndex() != 0 {
				foundOne = true
			}
		}
		if !foundOne {
			for i, entry := range list.GetEntries() {
				entry.Index = int32(i + 1)
			}
		}

		var wlist *pb.Wantlist
		for _, savedList := range savedLists {
			if savedList.GetName() == list.GetName() {
				wlist = savedList

				savedList.Type = list.GetType()
				savedList.EndDate = list.GetEndDate()
				savedList.StartDate = list.GetStartDate()

				break
			}
		}

		if wlist == nil {
			wlist = convertList(list)

		}

		// We've found this list, no we need to update it
		maxIndex := int32(0)
		for _, entry := range list.GetEntries() {
			if entry.GetIndex() > maxIndex {
				maxIndex = entry.GetIndex()
			}

			var wentry *pb.WantlistEntry
			for _, savedEntry := range wlist.GetEntries() {
				if entry.GetIndex() == savedEntry.GetIndex() {
					wentry = savedEntry
					break
				}
			}

			if wentry == nil {
				wentry = &pb.WantlistEntry{
					Index:    entry.GetIndex(),
					Id:       entry.GetId(),
					MasterId: entry.GetMasterId(),
				}

				log.Printf("Adding")
				wlist.Entries = append(wlist.Entries, wentry)
			} else {
				wentry.Id = entry.GetId()
				wentry.MasterId = entry.GetMasterId()
			}
		}

		// And trim off any hanging entries
		var newEntries []*pb.WantlistEntry
		if maxIndex != 0 || len(list.GetEntries()) != 0 {
			for _, entry := range wlist.GetEntries() {
				if entry.GetIndex() <= maxIndex {
					newEntries = append(newEntries, entry)
				}
			}
		}
		log.Printf("%v and %v -> %v from %v --> %v", maxIndex, len(list.GetEntries()), newEntries, list.GetEntries(), newEntries)

		wlist.Entries = newEntries
		err = s.d.SaveWantlist(ctx, u, wlist)
		if err != nil {
			return err
		}
	}

	// Delete any lists removed from the master list
	for _, slist := range savedLists {
		found := false
		for _, list := range lists {
			if list.GetName() == slist.GetName() {
				found = true
			}
		}

		if !found {
			s.d.DeleteWantlist(ctx, u.GetUser().GetDiscogsUserId(), slist.GetName())
		}
	}

	return nil
}

func (s *Server) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	u, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get user: %w", err)
	}

	log.Printf("Got user: %v", u.GetUser())
	log.Printf("Down to: %v", s.di.ForUser(u.GetUser()))

	fields, err := s.di.ForUser(u.GetUser()).GetFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to promote di to user (or get fields): %w", err)
	}

	log.Printf("got these fields: %v", fields)

	u.Config = req.GetConfig()
	folders, verr := config.ValidateConfig(ctx, u, fields, u)
	if verr != nil {
		return nil, fmt.Errorf("bad validate: %w", verr)
	}

	log.Printf("got folders: %v", folders)

	for _, folder := range folders {
		s.qc.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention:        "From New Config",
				RunDate:          time.Now().UnixNano(),
				Auth:             u.GetAuth().GetToken(),
				BackoffInSeconds: 60,
				Entry: &pb.QueueElement_AddFolderUpdate{
					AddFolderUpdate: &pb.AddFolderUpdate{FolderName: folder.GetName()},
				},
			}})
	}

	u.LastConfigUpdate = time.Now().UnixNano()

	log.Printf("Updated user: %v", u)

	if req.GetConfig().GetWantsConfig().GetMintUpWantList() {
		// Inject into the fields if not present
		found := false
		for _, list := range u.GetConfig().GetWantsListConfig().GetWantlists() {
			if list.GetName() == "mint_up_wantlist" {
				found = true
			}
		}
		if !found {
			if u.GetConfig().GetWantsListConfig() == nil {
				u.GetConfig().WantsListConfig = &pb.WantslistConfig{}
			}
			u.GetConfig().GetWantsListConfig().Wantlists = append(u.GetConfig().GetWantsListConfig().GetWantlists(), &pb.StoredWantlist{Name: "mint_up_wantlist"})
		} else {
			var nlist []*pb.StoredWantlist
			for _, list := range u.GetConfig().GetWantsListConfig().GetWantlists() {
				if list.GetName() != "mint_up_wantlist" {
					nlist = append(nlist, list)
				}
			}
			u.GetConfig().GetWantsListConfig().Wantlists = nlist
		}
	}

	if req.GetConfig().GetWantsConfig().GetExisting() == pb.WantsExisting_EXISTING_LIST {
		// Inject into the fields if not present
		found := false
		for _, list := range u.GetConfig().GetWantsListConfig().GetWantlists() {
			if list.GetName() == "float" {
				found = true
			}
		}
		if !found {
			if u.GetConfig().GetWantsListConfig() == nil {
				u.GetConfig().WantsListConfig = &pb.WantslistConfig{}
			}
			u.GetConfig().GetWantsListConfig().Wantlists = append(u.GetConfig().GetWantsListConfig().GetWantlists(), &pb.StoredWantlist{Name: "float"})
		} else {
			var nlist []*pb.StoredWantlist
			for _, list := range u.GetConfig().GetWantsListConfig().GetWantlists() {
				if list.GetName() != "float" {
					nlist = append(nlist, list)
				}
			}
			u.GetConfig().GetWantsListConfig().Wantlists = nlist
		}
	}

	if req.GetConfig().GetWantsConfig().GetDigitalWantList() {
		// Inject into the fields if not present
		found := false
		for _, list := range u.GetConfig().GetWantsListConfig().GetWantlists() {
			if list.GetName() == "digital_wantlist" {
				found = true
			}
		}
		if !found {
			if u.GetConfig().GetWantsListConfig() == nil {
				u.GetConfig().WantsListConfig = &pb.WantslistConfig{}
			}
			u.GetConfig().GetWantsListConfig().Wantlists = append(u.GetConfig().GetWantsListConfig().GetWantlists(), &pb.StoredWantlist{Name: "digital_wantlist", Type: pb.WantlistType_EN_MASSE})
		}

		log.Printf("Added digital wantlist: %v", u.GetConfig().GetWantsListConfig())
	}

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention:        "From new config",
			RunDate:          time.Now().UnixNano(),
			Auth:             u.GetAuth().GetToken(),
			BackoffInSeconds: 60,
			Entry: &pb.QueueElement_MoveRecords{
				MoveRecords: &pb.MoveRecords{},
			},
		}})
	if err != nil {
		return nil, fmt.Errorf("unable to enqueue: %w", err)
	}

	s.handleWantslists(ctx, u, u.GetConfig().WantsListConfig.GetWantlists())
	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention:        "From new config",
			RunDate:          time.Now().UnixNano(),
			Auth:             u.GetAuth().GetToken(),
			BackoffInSeconds: 60,
			Entry: &pb.QueueElement_RefreshWantlists{
				RefreshWantlists: &pb.RefreshWantlists{},
			},
		}})
	if err != nil {
		return nil, fmt.Errorf("unable to enqueue: %w", err)
	}

	return &pb.SetConfigResponse{}, s.d.SaveUser(ctx, u)
}
