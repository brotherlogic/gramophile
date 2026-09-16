package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetLocate() *CLIModule {
	return &CLIModule{
		Command: "locate",
		Help:    "Locate a record by ID",
		Execute: executeLocate,
	}
}

func calculatePercentage(beforeCount, afterCount int) float64 {
	total := beforeCount + afterCount + 1
	targetIndex := beforeCount + 1
	return float64(targetIndex) / float64(total) * 100.0
}

func resolveTargetRecord(ctx context.Context, client pb.GramophileEServiceClient, location *pb.Location, releaseId int64, cacheManager *RecordCacheManager) (string, error) {
	if cacheManager != nil && releaseId > 0 {
		if cacheManager.HasRelease(releaseId) {
			val, _ := cacheManager.ResolveRelease(ctx, client, releaseId)
			if val != "" {
				return val, nil
			}
		} else if client != nil {
			val, err := cacheManager.ResolveRelease(ctx, client, releaseId)
			if err == nil && val != "" && !strings.HasPrefix(val, "[Unresolved") {
				return val, nil
			}
		} else if location != nil && location.GetRecord() != "" && !strings.HasPrefix(location.GetRecord(), "Unknown") {
			cacheManager.mu.Lock()
			if cacheManager.data.GetReleaseCache() == nil {
				cacheManager.data.ReleaseCache = make(map[int64]*pb.RecordCacheEntry)
			}
			cacheManager.data.GetReleaseCache()[releaseId] = &pb.RecordCacheEntry{
				ArtistTitle:       location.GetRecord(),
				ResolvedTimestamp: time.Now().Unix(),
			}
			cacheManager.mu.Unlock()
			_ = cacheManager.Save()
		}
	}
	if location != nil && location.GetRecord() != "" {
		return location.GetRecord(), nil
	}
	if releaseId > 0 {
		return FormatUnresolved(releaseId), nil
	}
	return "", nil
}

func resolveContextRecord(ctx context.Context, client pb.GramophileEServiceClient, c *pb.Context, cacheManager *RecordCacheManager) (string, error) {
	if c == nil {
		return "", nil
	}
	if cacheManager != nil && c.GetIid() > 0 {
		if cacheManager.HasInstance(c.GetIid()) {
			val, _ := cacheManager.ResolveInstance(ctx, client, c.GetIid())
			if val != "" {
				return val, nil
			}
		} else if client != nil {
			val, err := cacheManager.ResolveInstance(ctx, client, c.GetIid())
			if err == nil && val != "" && !strings.HasPrefix(val, "[Unresolved") {
				return val, nil
			}
		} else if c.GetRecord() != "" && !strings.HasPrefix(c.GetRecord(), "Unknown") {
			cacheManager.mu.Lock()
			if cacheManager.data.GetInstanceCache() == nil {
				cacheManager.data.InstanceCache = make(map[int64]*pb.RecordCacheEntry)
			}
			cacheManager.data.GetInstanceCache()[c.GetIid()] = &pb.RecordCacheEntry{
				ArtistTitle:       c.GetRecord(),
				ResolvedTimestamp: time.Now().Unix(),
			}
			cacheManager.mu.Unlock()
			_ = cacheManager.Save()
		}
	}
	if c.GetRecord() != "" {
		return c.GetRecord(), nil
	}
	if c.GetIid() > 0 {
		return FormatUnresolved(c.GetIid()), nil
	}
	return "", nil
}

func formatLocationOutput(location *pb.Location, args ...interface{}) string {
	if location == nil {
		return ""
	}

	var cacheManager *RecordCacheManager
	var releaseId int64

	for _, arg := range args {
		switch v := arg.(type) {
		case *RecordCacheManager:
			cacheManager = v
		case int64:
			releaseId = v
		case int:
			releaseId = int64(v)
		}
	}

	targetRecord, _ := resolveTargetRecord(context.Background(), nil, location, releaseId, cacheManager)
	percentage := calculatePercentage(len(location.GetBefore()), len(location.GetAfter()))
	out := fmt.Sprintf("%v is in %v, Slot %v (%.0f %%):\n\n", targetRecord, location.GetLocationName(), location.GetSlot(), percentage)

	for i := len(location.GetBefore()) - 1; i >= 0; i-- {
		b := location.GetBefore()[i]
		bRec, _ := resolveContextRecord(context.Background(), nil, b, cacheManager)
		out += fmt.Sprintf("%v\n", bRec)
	}

	out += fmt.Sprintf("%v\n", targetRecord)

	for _, a := range location.GetAfter() {
		aRec, _ := resolveContextRecord(context.Background(), nil, a, cacheManager)
		out += fmt.Sprintf("%v\n", aRec)
	}
	out += "\n"
	return out
}

func renderLocation(ctx context.Context, client pb.GramophileEServiceClient, location *pb.Location, releaseId int64, cacheManager *RecordCacheManager, renderer *TerminalRenderer) (string, error) {
	if location == nil {
		return "", nil
	}
	if renderer == nil {
		renderer = NewTerminalRenderer(nil)
	}

	percentage := calculatePercentage(len(location.GetBefore()), len(location.GetAfter()))

	var targetRecord string
	var err error

	targetCached := cacheManager != nil && releaseId > 0 && cacheManager.HasRelease(releaseId)
	if renderer.IsTTY() && !targetCached && client != nil && releaseId > 0 {
		session := renderer.StartResolution("")
		targetRecord, err = resolveTargetRecord(ctx, client, location, releaseId, cacheManager)
		if err != nil {
			session.Cancel()
			return "", err
		}
		session.Finish(fmt.Sprintf("%v is in %v, Slot %v (%.0f %%):\n", targetRecord, location.GetLocationName(), location.GetSlot(), percentage))
	} else {
		targetRecord, err = resolveTargetRecord(ctx, client, location, releaseId, cacheManager)
		if err != nil {
			return "", err
		}
		renderer.mu.Lock()
		fmt.Fprintf(renderer.out, "%v is in %v, Slot %v (%.0f %%):\n\n", targetRecord, location.GetLocationName(), location.GetSlot(), percentage)
		renderer.mu.Unlock()
	}

	var lines []string
	for i := len(location.GetBefore()) - 1; i >= 0; i-- {
		b := location.GetBefore()[i]
		isCached := cacheManager != nil && b.GetIid() > 0 && cacheManager.HasInstance(b.GetIid())
		var bRec string
		if renderer.IsTTY() && !isCached && client != nil && b.GetIid() > 0 {
			session := renderer.StartResolution("")
			bRec, err = resolveContextRecord(ctx, client, b, cacheManager)
			if err != nil {
				session.Cancel()
				return "", err
			}
			session.Finish(bRec)
		} else {
			bRec, err = resolveContextRecord(ctx, client, b, cacheManager)
			if err != nil {
				return "", err
			}
			renderer.mu.Lock()
			fmt.Fprintf(renderer.out, "%s\n", bRec)
			renderer.mu.Unlock()
		}
		lines = append(lines, bRec)
	}

	renderer.mu.Lock()
	fmt.Fprintf(renderer.out, "%s\n", targetRecord)
	renderer.mu.Unlock()

	for _, a := range location.GetAfter() {
		isCached := cacheManager != nil && a.GetIid() > 0 && cacheManager.HasInstance(a.GetIid())
		var aRec string
		if renderer.IsTTY() && !isCached && client != nil && a.GetIid() > 0 {
			session := renderer.StartResolution("")
			aRec, err = resolveContextRecord(ctx, client, a, cacheManager)
			if err != nil {
				session.Cancel()
				return "", err
			}
			session.Finish(aRec)
		} else {
			aRec, err = resolveContextRecord(ctx, client, a, cacheManager)
			if err != nil {
				return "", err
			}
			renderer.mu.Lock()
			fmt.Fprintf(renderer.out, "%s\n", aRec)
			renderer.mu.Unlock()
		}
		lines = append(lines, aRec)
	}

	renderer.mu.Lock()
	fmt.Fprintf(renderer.out, "\n")
	renderer.mu.Unlock()

	out := fmt.Sprintf("%v is in %v, Slot %v (%.0f %%):\n\n", targetRecord, location.GetLocationName(), location.GetSlot(), percentage)
	for i := 0; i < len(location.GetBefore()); i++ {
		out += fmt.Sprintf("%v\n", lines[i])
	}
	out += fmt.Sprintf("%v\n", targetRecord)
	for i := 0; i < len(location.GetAfter()); i++ {
		out += fmt.Sprintf("%v\n", lines[len(location.GetBefore())+i])
	}
	out += "\n"

	return out, nil
}

func runLocate(ctx context.Context, client pb.GramophileEServiceClient, args []string, cacheManager *RecordCacheManager, renderer *TerminalRenderer) error {
	idSet := flag.NewFlagSet("ids", flag.ExitOnError)
	var id = idSet.Int("id", 0, "Id of record to locate")

	if err := idSet.Parse(args); err != nil {
		return err
	}

	res, err := client.LocateRecord(ctx, &pb.LocateRecordRequest{
		ReleaseId: int64(*id),
	})
	if err != nil {
		return err
	}

	for _, location := range res.GetLocations() {
		if _, err := renderLocation(ctx, client, location, int64(*id), cacheManager, renderer); err != nil {
			return err
		}
	}

	return nil
}

func executeLocate(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewGramophileEServiceClient(conn)
	cacheManager, err := NewRecordCacheManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: unable to initialize record cache: %v\n", err)
	}
	renderer := NewTerminalRenderer(nil)

	return runLocate(ctx, client, args, cacheManager, renderer)
}
