package background

import (
	"context"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

type fanoutHistoryHandler struct {
	b *BackgroundRunner
}

func (h *fanoutHistoryHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.FanoutHistory(ctx, entry.GetFanoutHistory().GetType(), u, entry.GetAuth(), enqueue)
}

func (h *fanoutHistoryHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *fanoutHistoryHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

type recordHistoryHandler struct {
	b *BackgroundRunner
}

func (h *recordHistoryHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.RecordHistory(ctx, entry.GetRecordHistory().GetType(), int64(u.GetUser().GetDiscogsUserId()), entry.GetRecordHistory().GetInstanceId())
}

func (h *recordHistoryHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *recordHistoryHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

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
