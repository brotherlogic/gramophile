package server

import (
	"context"
	"sync"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) Clean(ctx context.Context, req *pb.CleanRequest) (*pb.CleanResponse, error) {
	return &pb.CleanResponse{}, s.d.Clean(ctx, req.GetType())
}

func (s *Server) GetWaitlistStatus(ctx context.Context, req *pb.GetWaitlistStatusRequest) (*pb.GetWaitlistStatusResponse, error) {
	users, err := s.d.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	var waitlistUsers []*pb.StoredUser
	for _, userToken := range users {
		su, err := s.d.GetUser(ctx, userToken)
		if err != nil {
			return nil, err
		}
		if su.GetState() == pb.StoredUser_USER_STATE_IN_WAITLIST {
			waitlistUsers = append(waitlistUsers, su)
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var resUsers []*pb.WaitlistUser

	for _, user := range waitlistUsers {
		wg.Add(1)
		go func(u *pb.StoredUser) {
			defer wg.Done()

			uid := u.GetUser().GetDiscogsUserId()
			recs, err := s.d.GetRecords(ctx, uid)
			syncedCollection := int32(0)
			if err == nil {
				syncedCollection = int32(len(recs))
			}

			wants, err := s.d.GetWants(ctx, uid)
			syncedWantlist := int32(0)
			if err == nil {
				syncedWantlist = int32(len(wants))
			}

			lastSynced := time.Unix(0, u.GetLastItemSyncedTime())
			isStuck := time.Since(lastSynced) > time.Hour

			fullySynced := syncedCollection >= u.GetExpectedCollectionSize() && syncedWantlist >= u.GetExpectedWantlistSize()

			remCollection := u.GetExpectedCollectionSize() - syncedCollection
			if remCollection < 0 {
				remCollection = 0
			}
			remWantlist := u.GetExpectedWantlistSize() - syncedWantlist
			if remWantlist < 0 {
				remWantlist = 0
			}

			etaSeconds := int64(remCollection + remWantlist)

			mu.Lock()
			resUsers = append(resUsers, &pb.WaitlistUser{
				User:                 u,
				SyncedCollectionSize: syncedCollection,
				SyncedWantlistSize:   syncedWantlist,
				IsStuck:              isStuck,
				FullySynced:          fullySynced,
				EtaSeconds:           etaSeconds,
			})
			mu.Unlock()
		}(user)
	}

	wg.Wait()

	return &pb.GetWaitlistStatusResponse{Users: resUsers}, nil
}
