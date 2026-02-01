package background

import (
	"context"
	"log"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) FanoutHistory(ctx context.Context, typ pb.UpdateType, user *pb.StoredUser, auth string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	log.Printf("FANNING OUT: %v for %v", user, typ)
	records, err := b.db.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return err
	}

	for _, record := range records {
		enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention: "From Fanout History",
				Auth:      auth,
				RunDate:   time.Now().UnixNano(),
				Entry: &pb.QueueElement_RecordHistory{
					RecordHistory: &pb.RecordHistory{
						InstanceId: record,
						Userid:     int64(user.GetUser().GetDiscogsUserId()),
						Type:       typ,
					},
				}}})
	}

	user.GetUpdates().LastBackfill[typ.String()] = time.Now().UnixNano()
	return b.db.SaveUser(ctx, user)
}

func (b *BackgroundRunner) RecordHistory(ctx context.Context, typ pb.UpdateType, userid, iid int64) error {
	log.Printf("Processing %v, %v -> %v", userid, iid, typ)
	rids, err := b.db.GetRecordHistory(ctx, int32(userid), iid)
	if err != nil {
		return err
	}

	for i := 0; i < len(rids)-1; i++ {
		r1, err := b.db.GetHistoricalRecord(ctx, userid, iid, rids[i])
		if err != nil {
			return err
		}

		r2, err := b.db.GetHistoricalRecord(ctx, userid, iid, rids[i+1])
		if err != nil {
			return err
		}
		diff := b.db.GetDiff(r1, r2, typ)
		err = b.db.SaveUpdate(ctx, int32(userid), r2, diff)
		if err != nil {
			return err
		}
	}

	return nil
}
