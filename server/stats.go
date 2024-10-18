package server

import (
	"context"
	"log"
	"math"
	"time"

	dpb "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) getCollectionStats(ctx context.Context, userid int32) (*pb.CollectionStats, error) {
	recs, err := s.d.LoadAllRecords(ctx, userid)
	if err != nil {
		return nil, err
	}

	cs := &pb.CollectionStats{FolderToCount: make(map[int32]int32)}
	for _, r := range recs {
		cs.FolderToCount[r.GetRelease().GetFolderId()]++
	}

	return cs, nil
}

func (s *Server) getSalesStats(ctx context.Context, userid int32) (*pb.SaleStats, error) {
	sales, err := s.d.GetSales(ctx, userid)
	if err != nil {
		return nil, err
	}

	ss := &pb.SaleStats{YearTotals: make(map[int32]int32), StateCount: make(map[string]int32), LastUpdate: map[int64]int64{}}
	totals := int32(0)
	ss.OldestLastUpdate = math.MaxInt64
	for _, sl := range sales {
		sale, err := s.d.GetSale(ctx, userid, sl)
		ss.LastUpdate[sale.GetSaleId()] = int64(time.Since(time.Unix(0, sale.GetLastPriceUpdate())).Seconds())
		if err != nil {
			return nil, err
		}

		if sale.GetSaleState() == dpb.SaleStatus_SOLD {
			ss.YearTotals[int32(time.Unix(0, sale.GetSoldDate()).Year())] += sale.GetCurrentPrice().GetValue()
		}

		if sale.GetSaleState() == dpb.SaleStatus_FOR_SALE {
			if sale.GetLastPriceUpdate() < ss.GetOldestLastUpdate() {
				ss.OldestLastUpdate = sale.GetLastPriceUpdate()
				log.Printf("Oldest update: %v", sale)
			}

			totals += sale.GetCurrentPrice().GetValue()
			if sale.GetTimeAtStale() > 0 {
				ss.StateCount["STALE"]++
			} else if sale.GetTimeAtLow() > 0 {
				ss.StateCount["TO_STALE"]++
			} else if sale.GetTimeAtMedian() > 0 {
				ss.StateCount["TO_LOW"]++
			} else {
				ss.StateCount["TO_MEDIAN"]++
			}
		}
	}
	ss.TotalSales = totals

	return ss, nil
}

func (s *Server) GetStats(ctx context.Context, _ *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	cStats, err := s.getCollectionStats(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	sStats, err := s.getSalesStats(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	return &pb.GetStatsResponse{
		CollectionStats: cStats,
		SaleStats:       sStats,
	}, nil
}
