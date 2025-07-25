package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	discogs "github.com/brotherlogic/discogs"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	irefresh = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_intent_refresh",
		Help: "The length of the working queue I think yes",
	}, []string{"iid"})
)

func (s *Server) validateIntent(ctx context.Context, user *pb.StoredUser, i *pb.Intent) error {
	if i.GetGoalFolder() != "" {
		found := false
		for _, folder := range user.GetFolders() {
			if folder.GetName() == i.GetGoalFolder() {
				found = true
			}
		}

		if !found {
			return status.Errorf(codes.FailedPrecondition, "%v is not in the list of current user folders", i.GetGoalFolder())
		}
	}

	if i.GetSleeve() != "" {
		found := false
		for _, sleeve := range user.GetConfig().GetSleeveConfig().GetAllowedSleeves() {
			if sleeve.GetName() == i.GetSleeve() {
				found = true
			}
		}

		if !found {
			return status.Errorf(codes.FailedPrecondition, "%v is not in the list of allowed sleeves", i.GetSleeve())
		}
	}

	return nil
}

func mapDiscogsScore(score int32, config *pb.ScoreConfig) int32 {
	if config.GetBottomRange() >= config.GetTopRange() {
		return score
	}
	rangeWidth := config.GetTopRange() - config.GetBottomRange()
	return int32(math.Ceil(5 * (float64(score) / float64(rangeWidth))))
}

func (s *Server) SetIntent(ctx context.Context, req *pb.SetIntentRequest) (*pb.SetIntentResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error getting user: %w", err)
	}

	// Check that this record at least exists
	r, err := s.d.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId())
	if err != nil {
		// If we can't find the record, we may be behind - trigger a collection refresh
		if status.Code(err) == codes.NotFound {
			irefresh.With(prometheus.Labels{"iid": fmt.Sprintf("%v", req.GetInstanceId())}).Inc()
			s.qc.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Priority: pb.QueueElement_PRIORITY_HIGH,
					Force:    true,
					RunDate:  time.Now().UnixNano(),
					Auth:     user.GetAuth().GetToken(),
					Entry: &pb.QueueElement_RefreshCollectionEntry{
						RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}}}})
		}
		return nil, fmt.Errorf("error getting record: %w", err)
	}

	if req.GetIntent().GetKeep() == pb.KeepStatus_MINT_UP_KEEP && len(req.GetIntent().GetMintIds()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "You need to specify mint ids for this keep")
	}

	if req.GetIntent().GetKeep() == pb.KeepStatus_DIGITAL_KEEP {
		if discogs.ReleaseIsDigital(r.GetRelease()) {
			return nil, status.Errorf(codes.InvalidArgument, "This record is already digital")
		}
	}

	// Validate that the intent is legit
	err = s.validateIntent(ctx, user, req.GetIntent())
	if err != nil {
		return nil, err
	}

	// If this is for a backdated score, process it and exit without saving the intent
	if req.GetIntent().GetNewScoreTime() > 0 {
		log.Printf("Fast write of new score")
		discogsScore := mapDiscogsScore(req.GetIntent().GetNewScore(), user.GetConfig().GetScoreConfig())
		enumVal := req.GetIntent().GetNewScoreListen()
		if enumVal == pb.ListenStatus_LISTEN_STATUS_UNKNOWN {
			enumVal = pb.ListenStatus_LISTEN_STATUS_NO_LISTEN // We default to no listen
		}
		r.ScoreHistory = append(r.ScoreHistory, &pb.Score{
			ScoreValue:                req.GetIntent().GetNewScore(),
			ScoreMappedTo:             discogsScore,
			ListenStatus:              enumVal,
			AppliedToDiscogsTimestamp: req.GetIntent().GetNewScoreTime(),
		})
		return &pb.SetIntentResponse{}, s.d.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), r)
	}

	log.Printf("Saving intent: %v -> %v", req.GetIntent(), user)

	// Clear the score if we've moved into the listening pile
	//TODO: Turn this into a config setting on the folder
	if req.GetIntent().GetNewFolder() == 3386035 {
		req.GetIntent().NewScore = -1
		req.GetIntent().Keep = pb.KeepStatus_RESET
		req.GetIntent().Weight = 1
		req.GetIntent().Width = 0.1
	}

	if req.GetIntent().GetNewFolder() == 7651472 || req.GetIntent().GetNewFolder() == 7665013 {
		req.GetIntent().NewScore = -1
		req.GetIntent().Keep = pb.KeepStatus_RESET
	}

	ts := time.Now().UnixNano()
	err = s.d.SaveIntent(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId(), req.GetIntent(), ts)
	if err != nil {
		return nil, fmt.Errorf("error saving intent: %w", err)
	}

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:          time.Now().UnixNano(), // We want intents to run before anything else
			Priority:         pb.QueueElement_PRIORITY_HIGH,
			Auth:             user.GetAuth().GetToken(),
			BackoffInSeconds: 60,
			Entry: &pb.QueueElement_RefreshIntents{
				RefreshIntents: &pb.RefreshIntents{
					InstanceId: req.GetInstanceId(),
					Timestamp:  ts},
			},
		},
	})

	log.Printf("Saved Intent")
	if user.GetUser().GetDiscogsUserId() == 150295 {
		nerr := s.updateRecord(ctx, int32(req.GetInstanceId()), int32(r.GetRelease().GetId()))
		if nerr != nil {
			log.Printf("Error on record update: %v", nerr)
		}
	} else {
		log.Printf("Skipping: %v", user.GetUser().GetDiscogsUserId())
	}

	return &pb.SetIntentResponse{}, err
}
