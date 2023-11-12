package background

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) loadMoveQuota(ctx context.Context, userid int32) (*pb.MoveQuota, error) {
	quota, err := b.db.LoadMoveQuota(ctx, userid)
	if err != nil {
		return nil, err
	}

	var mh []*pb.MoveHistory
	for _, move := range quota.GetPastMoves() {
		if time.Since(time.Unix(move.GetTime(), 0)) < time.Hour {
			mh = append(mh, move)
		}
	}

	return &pb.MoveQuota{PastMoves: mh}, nil
}

func filter(c *pb.MoveCriteria, r *pb.Record) bool {
	if c.GetHasSaleId() != pb.Bool_UNKNOWN {
		if c.GetHasSaleId() == pb.Bool_TRUE && r.GetSaleId() == 0 {
			return false
		}

		if c.GetHasSaleId() == pb.Bool_FALSE && r.GetSaleId() > 0 {
			return false
		}
	}

	if c.GetArrived() != pb.Bool_UNKNOWN {
		if c.GetArrived() == pb.Bool_TRUE && r.GetArrived() == 0 {
			return false
		}
		if c.GetArrived() == pb.Bool_FALSE && r.GetArrived() > 0 {
			return false
		}
	}

	if c.GetListened() != pb.Bool_UNKNOWN {
		if c.GetListened() == pb.Bool_TRUE && r.GetLastListenTime() == 0 {
			return false
		}
		if c.GetListened() == pb.Bool_FALSE && r.GetLastListenTime() > 0 {
			return false
		}
	}

	return true
}

func applyMove(m *pb.FolderMove, r *pb.Record) string {
	if m.GetMoveState() == pb.MoveState_BLOCKED_BECAUSE_OF_LOOP {
		return ""
	}

	if filter(m.GetCriteria(), r) {
		if m.GetMoveToGoalFolder() {
			return r.GetGoalFolder()
		}
		return m.GetMoveFolder()
	}

	return ""
}

const (
	MaxMoves = 3
)

func (b *BackgroundRunner) updateQuota(ctx context.Context, quota *pb.MoveQuota, uid int32, iid int64, fm *pb.FolderMove) bool {
	count := 0
	log.Printf("MOVE: %v (%v)", quota, fm.GetName())
	for _, q := range quota.GetPastMoves() {
		if q.GetIid() == iid && q.GetMove() == fm.GetName() {
			count++
		}
	}

	if count > MaxMoves {
		fm.MoveState = pb.MoveState_BLOCKED_BECAUSE_OF_LOOP
		return false
	}

	quota.PastMoves = append(quota.PastMoves, &pb.MoveHistory{
		Iid:  iid,
		Move: fm.GetName(),
		Time: time.Now().Unix(),
	})
	err := b.db.SaveMoveQuota(ctx, uid, quota)
	if err != nil {
		return false
	}
	return true
}

func (b *BackgroundRunner) RunMoves(ctx context.Context, user *pb.StoredUser, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	moves := user.GetMoves()
	quota, err := b.loadMoveQuota(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			quota = &pb.MoveQuota{}
		} else {
			return err
		}
	}

	records, err := b.db.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return fmt.Errorf("unablet to get records: %v", err)
	}

	log.Printf("Running %v moves on %v records", len(moves), len(records))

	for _, iid := range records {
		record, err := b.db.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), iid)
		if err != nil {
			return err
		}

		for _, move := range moves {
			nfolder := applyMove(move, record)
			log.Printf("MOVE: %v", nfolder)
			if nfolder != "" {
				if b.updateQuota(ctx, quota, user.GetUser().GetDiscogsUserId(), iid, move) {
					_, err = enqueue(ctx, &pb.EnqueueRequest{
						Element: &pb.QueueElement{
							RunDate: time.Now().UnixNano(),
							Auth:    user.GetAuth().GetToken(),
							Entry: &pb.QueueElement_MoveRecord{
								MoveRecord: &pb.MoveRecord{
									RecordIid:  iid,
									MoveFolder: nfolder,
									Rule:       move.GetName(),
								}}},
					})
					log.Printf("enqueued move: %v", err)
					if err != nil {
						return err
					}
				} else {
					log.Printf("OVER QUOTA")
					return b.db.SaveUser(ctx, user)
				}
			}
		}
	}

	return b.db.SaveMoveQuota(ctx, user.GetUser().GetDiscogsUserId(), quota)
}
