package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/gramophile/classification"
	pb "github.com/brotherlogic/gramophile/proto"
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
	for _, classifier := range fc.GetFormats() {
		description := false
		names := false

		for _, form := range fc.GetFormats() {
			for _, desc := range form.GetDescription() {
				for _, rform := range record.GetRelease().GetFormats() {
					for _, rdesc := range rform.GetDescriptions() {
						if rdesc == desc {
							description = true
						}
					}
				}
			}
			for _, name := range form.GetContains() {
				for _, rform := range record.GetRelease().GetFormats() {
					if rform.GetName() == name {
						names = true
					}
				}
			}
		}

		if description && names {
			return classifier.GetFormat()
		}
	}

	return fc.GetDefaultFormat()
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

	for _, iid := range records {
		record, err := b.db.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), iid)
		if err != nil {
			return err
		}
		// Get record classification
		class := classification.Classify(ctx, record, user.GetConfig().GetClassificationConfig(), user.GetConfig().GetOrganisationConfig(), b.db, user.GetUser().GetDiscogsUserId())

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
