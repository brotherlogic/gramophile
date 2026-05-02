package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) CullWants(ctx context.Context, d discogs.Discogs, sid int64) error {
	wants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return err
	}

	for _, swant := range wants {
		if swant.GetSyncId() != sid && swant.GetState() == pb.WantState_WANTED {
			swant.IntendedState = pb.WantState_RETIRED
			err = b.db.SaveWant(ctx, d.GetUserId(), swant, "Determined to be culled")
		}
	}

	return nil
}

type syncWantsHandler struct {
	b *BackgroundRunner
}

func (h *syncWantsHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.ProcessSyncWants(ctx, d, u, entry, enqueue)
}

func (h *syncWantsHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *syncWantsHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

func (b *BackgroundRunner) ProcessSyncWants(ctx context.Context, d discogs.Discogs, user *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	// Only refresh every 24 hours
	if time.Since(time.Unix(0, user.GetLastWantRefresh())) < time.Hour*24 {
		qlog(ctx, "Needs more time to sync")
		return nil
	}

	if entry.GetSyncWants().GetPage() == 1 {
		entry.GetSyncWants().RefreshId = time.Now().UnixNano()
	}
	pages, err := b.PullWants(ctx, d, entry.GetSyncWants().GetPage(), entry.GetSyncWants().GetRefreshId(), user.GetConfig().GetWantsConfig())
	if err != nil {
		return err
	}
	if entry.GetSyncWants().GetPage() == 1 {
		for i := int32(2); i <= pages; i++ {
			enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate: time.Now().UnixNano() + int64(i),
					Entry: &pb.QueueElement_SyncWants{
						SyncWants: &pb.SyncWants{Page: i, RefreshId: entry.GetSyncWants().GetRefreshId()},
					},
					Auth: entry.GetAuth(),
				},
			})
		}
	}

	// If this is the final sync, let's run the alignment
	if entry.GetSyncWants().GetPage() >= pages {
		err = b.AlignWants(ctx, user, user.GetConfig().GetWantsConfig())
		if err != nil {
			return err
		}

		// Save any dirty wants
		wants, err := b.db.GetWants(ctx, user.GetUser().GetDiscogsUserId())
		if err != nil {
			return err
		}
		for _, want := range wants {
			if !want.GetClean() {
				_, err = enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:          time.Now().UnixNano(),
						Auth:             user.GetAuth().GetToken(),
						BackoffInSeconds: 60,
						Entry: &pb.QueueElement_RefreshWant{
							RefreshWant: &pb.RefreshWant{
								Want: want,
							},
						},
					},
				})
			}
		}

		_, err = enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate:          time.Now().UnixNano(),
				Auth:             user.GetAuth().GetToken(),
				BackoffInSeconds: 60,
				Entry: &pb.QueueElement_RefreshWants{
					RefreshWants: &pb.RefreshWants{},
				},
			},
		})

		_, err = enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention:        "From SyncWants",
				RunDate:          time.Now().UnixNano(),
				Auth:             user.GetAuth().GetToken(),
				BackoffInSeconds: 60,
				Entry: &pb.QueueElement_RefreshWantlists{
					RefreshWantlists: &pb.RefreshWantlists{},
				},
			},
		})

		user.LastWantRefresh = time.Now().UnixNano()
		return b.db.SaveUser(ctx, user)
	}

	return nil
}

func (b *BackgroundRunner) PullWants(ctx context.Context, d discogs.Discogs, page int32, sid int64, wc *pb.WantsConfig) (int32, error) {
	wants, pag, err := d.GetWants(ctx, page)
	log.Printf("GET_WANTS: %v", wants)

	if err != nil {
		return -1, fmt.Errorf("bad get wants: %w", err)
	}

	swants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return -1, err
	}
	for _, want := range wants {
		found := false
		for _, swant := range swants {
			if want.GetId() == swant.GetId() {
				found = true
				swant.SyncId = sid
				err := b.db.SaveWant(ctx, d.GetUserId(), swant, fmt.Sprintf("Updating on refresh (%v)", swant.GetState()))
				if err != nil {
					return -1, fmt.Errorf("error on save in pull: %w", err)
				}

				continue
			}
		}

		if !found {
			state := pb.WantState_WANTED
			if wc.GetOrigin() == pb.WantsBasis_WANTS_GRAMOPHILE {
				state = pb.WantState_RETIRED
			}
			err := b.db.SaveWant(ctx, d.GetUserId(), &pb.Want{
				Id:            want.GetId(),
				WantAddedDate: time.Now().UnixNano(),
				State:         state,
				SyncId:        sid,
			}, "Creating in sync")

			if err != nil {
				return -1, fmt.Errorf("error on new want in pull: %w", err)
			}
		}
	}

	return pag.GetPages(), nil
}
