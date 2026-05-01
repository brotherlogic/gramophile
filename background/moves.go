package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/classification"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func applyMove(m *pb.RecordMove, r *pb.Record, class string, format string) string {
	log.Printf("Running move on %v and %v", class, format)
	for _, classification := range m.GetClassification() {
		if classification == class {
			for _, rformat := range m.GetFormat() {
				if rformat == format {
					return m.GetFolder()
				}
			}
		}
	}
	return ""
}

func (b *BackgroundRunner) GetFormat(ctx context.Context, record *pb.Record, fc *pb.FormatClassifier) string {
	allDescriptions := make(map[string]struct{})
	allNames := make(map[string]struct{})
	for _, rform := range record.GetRelease().GetFormats() {
		allNames[rform.GetName()] = struct{}{}
		for _, rdesc := range rform.GetDescriptions() {
			allDescriptions[rdesc] = struct{}{}
		}
	}

	for _, classifier := range fc.GetFormats() {
		description := len(classifier.GetDescription()) == 0
		names := len(classifier.GetContains()) == 0

		for _, desc := range classifier.GetDescription() {
			if _, ok := allDescriptions[desc]; ok {
				description = true
				break
			}
		}

		for _, name := range classifier.GetContains() {
			if _, ok := allNames[name]; ok {
				names = true
				break
			}
		}

		if description && names {
			return classifier.GetFormat()
		}
	}

	return fc.GetDefaultFormat()
}

func (b *BackgroundRunner) MoveRecord(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.MoveRecord, auth string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	rec, err := b.db.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), entry.GetRecordIid())
	if err != nil {
		return fmt.Errorf("unable to get record: %w", err)
	}

	fNum := int32(-1)
	for _, folder := range u.GetFolders() {
		if folder.GetName() == entry.GetMoveFolder() {
			fNum = folder.GetId()
		}
	}

	log.Printf("Moving record from %v to %v", rec.GetRelease().GetFolderId(), fNum)

	// Fast exit if we don't need to make this move
	if rec.GetRelease().GetFolderId() == fNum {
		return nil
	}

	if fNum < 0 {
		return status.Errorf(codes.NotFound, "folder %v was not found", entry.GetMoveFolder())
	}

	err = d.SetFolder(ctx, rec.GetRelease().GetInstanceId(), rec.GetRelease().GetId(), rec.GetRelease().GetFolderId(), fNum)
	if err != nil {
		return fmt.Errorf("unable to move record: %w", err)
	}

	qlog(ctx, "Setting folder: %v", fNum)

	//Update and save record
	rec.GetRelease().FolderId = int32(fNum)
	err = b.db.SaveRecordWithUpdate(ctx, u.GetUser().GetDiscogsUserId(), rec, &pb.RecordUpdate{
		Date: time.Now().UnixNano(),
	})
	if err != nil {
		return err
	}

	_, err = enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: time.Now().UnixNano(),
			Entry: &pb.QueueElement_MoveRecords{
				MoveRecords: &pb.MoveRecords{},
			},
			Auth: auth,
		}})
	return err
}

func (b *BackgroundRunner) RunMoves(ctx context.Context, user *pb.StoredUser, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	moves := user.GetConfig().GetMovingConfig().GetMoves()

	// Fast return on empty moves
	if len(moves) == 0 {
		return nil
	}

	records, err := b.db.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return fmt.Errorf("unable to get records: %v", err)
	}

	log.Printf("Running %v moves on %v records", len(moves), len(records))

	classifier := classification.CreateClassifier(user.GetConfig().GetClassificationConfig(), b.db, user.GetUser().GetDiscogsUserId())
	for _, iid := range records {
		record, err := b.db.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), iid)
		if err != nil {
			return err
		}
		// Get record classification
		class := classifier.Classify(ctx, record)

		format := b.GetFormat(ctx, record, user.GetConfig().GetMovingConfig().GetFormatClassifier())

		for _, move := range moves {
			nfolder := applyMove(move, record, class, format)
			log.Printf("MOVE: %v", nfolder)
			if nfolder != "" {
				_, err = enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						Intention: "From Run Moves",
						RunDate:   time.Now().UnixNano(),
						Auth:      user.GetAuth().GetToken(),
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
			}
		}
	}

	return nil
}
