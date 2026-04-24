package server

import (
	"context"
	"fmt"
	"log"
	"time"

	orglogic "github.com/brotherlogic/gramophile/org"
	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	orgLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gramophile_org_latency",
	}, []string{"stage"})
	overallLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "gramophile_overall_org_latency",
	})
)

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func (s *Server) SetOrgSnapshot(ctx context.Context, req *pb.SetOrgSnapshotRequest) (*pb.SetOrgSnapshotResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load user: %w", err)
	}

	org, err := s.d.LoadSnapshot(ctx, user, req.GetOrgName(), fmt.Sprintf("%v", req.GetDate()))
	if err != nil {
		return nil, fmt.Errorf("unable to load snapshot: %w", err)
	}

	org.Name = req.GetName()
	err = s.d.SaveSnapshot(ctx, user, req.GetOrgName(), org)
	if err != nil {
		return nil, fmt.Errorf("unable to save snapshot: %w", err)
	}

	return &pb.SetOrgSnapshotResponse{}, nil
}

func (s *Server) GetOrg(ctx context.Context, req *pb.GetOrgRequest) (*pb.GetOrgResponse, error) {
	t := time.Now()
	defer func() {
		overallLatency.Observe(float64(time.Since(t).Milliseconds()))
	}()

	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	orgLatency.With(prometheus.Labels{"stage": "getUser"}).Observe(float64(time.Since(t).Milliseconds()))

	if req.GetHash() != "" {
		t := time.Now()
		snapshot, err := s.d.LoadSnapshotHash(ctx, user, req.GetOrgName(), req.GetHash())
		if err != nil {
			return nil, fmt.Errorf("Unable to load snapshot hash: %w", err)
		}
		orgLatency.With(prometheus.Labels{"stage": "snapshothasg"}).Observe(float64(time.Since(t).Milliseconds()))

		return &pb.GetOrgResponse{Snapshot: snapshot}, nil
	}

	if req.GetName() != "" {
		t := time.Now()
		snapshot, err := s.d.LoadSnapshot(ctx, user, req.GetOrgName(), req.GetName())
		if err != nil {
			return nil, fmt.Errorf("Unable to load snapshot: %w", err)
		}
		orgLatency.With(prometheus.Labels{"stage": "loadsnapshot"}).Observe(float64(time.Since(t).Milliseconds()))

		return &pb.GetOrgResponse{Snapshot: snapshot}, nil
	}

	var o *pb.Organisation
	for _, org := range user.GetConfig().GetOrganisationConfig().GetOrganisations() {
		if org.GetName() == req.GetOrgName() {
			o = org
		}
	}

	if o == nil {
		return nil, status.Errorf(codes.NotFound, "unable to locate org called %v", req.GetOrgName())
	}

	t = time.Now()
	org := orglogic.GetOrgSwallow(s.d)
	orgLatency.With(prometheus.Labels{"stage": "getorg"}).Observe(float64(time.Since(t).Milliseconds()))

	t = time.Now()
	snapshot, err := org.BuildSnapshot(ctx, user, o, user.Config.GetOrganisationConfig())
	if err != nil {
		return nil, fmt.Errorf("unable to build snapshot: %w", err)
	}
	orgLatency.With(prometheus.Labels{"stage": "buildsnapshot"}).Observe(float64(time.Since(t).Milliseconds()))

	t = time.Now()
	latest, err := s.d.GetLatestSnapshot(ctx, user.GetUser().GetDiscogsUserId(), req.GetOrgName())
	if err != nil && status.Code(err) != codes.NotFound {
		return nil, fmt.Errorf("unable to load previous snapshot: %w", err)
	}
	orgLatency.With(prometheus.Labels{"stage": "getlatestsnapshot"}).Observe(float64(time.Since(t).Milliseconds()))

	log.Printf("LATEST %v vs RECENT %v", time.Unix(0, latest.GetDate()), time.Unix(0, snapshot.GetDate()))

	if latest == nil || latest.GetHash() != snapshot.GetHash() {
		t = time.Now()
		err = s.d.SaveSnapshot(ctx, user, req.GetOrgName(), snapshot)
		if err != nil {
			return nil, fmt.Errorf("unable to save new snapshot: %w", err)
		}
		orgLatency.With(prometheus.Labels{"stage": "savesnapshot"}).Observe(float64(time.Since(t).Milliseconds()))

	}

	return &pb.GetOrgResponse{Snapshot: snapshot}, nil
}

func getSnapshotDiff(start, end *pb.OrganisationSnapshot) []*pb.Move {
	startMap := make(map[int64]*pb.Placement)
	for _, p := range start.GetPlacements() {
		startMap[p.GetIid()] = p
	}

	endMap := make(map[int64]*pb.Placement)
	for _, p := range end.GetPlacements() {
		endMap[p.GetIid()] = p
	}

	var moves []*pb.Move
	// Check for moves and deletions
	for iid, startP := range startMap {
		endP, ok := endMap[iid]
		if !ok {
			// Deletion
			moves = append(moves, &pb.Move{
				Start: startP,
				End:   nil,
			})
		} else if !proto.Equal(startP, endP) {
			// Move
			moves = append(moves, &pb.Move{
				Start: startP,
				End:   endP,
			})
		}
	}

	// Check for additions
	for iid, endP := range endMap {
		if _, ok := startMap[iid]; !ok {
			moves = append(moves, &pb.Move{
				Start: nil,
				End:   endP,
			})
		}
	}

	return moves
}
