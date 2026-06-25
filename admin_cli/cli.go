package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*60)
	defer cancel()

	conn, err := grpc.Dial(os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Bad dial: %v", err)
	}

	sconn, serr := grpc.Dial(os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if serr != nil {
		log.Fatalf("Bad dial: %v", err)
	}

	client := pb.NewQueueServiceClient(conn)
	sclient := pb.NewGramophileServiceClient(sconn)

	switch os.Args[2] {
	case "upgrade_user":
		resp, err := sclient.UpgradeUser(ctx, &pb.UpgradeUserRequest{
			Username: os.Args[3],
			NewState: pb.StoredUser_USER_STATE_LIVE,
		})
		if err != nil {
			log.Fatalf("Unable to upgrade user: %v", err)
		}
		fmt.Printf("Upgrded: %v", resp)
	case "tdrain":
		resp, err := client.Drain(ctx, &pb.DrainRequest{
			DrainType: pb.DrainRequest_JUST_RELEASE_DATES,
		})
		if err != nil {
			log.Fatalf("Unable to drain queue: %v", err)
		}
		fmt.Printf("Drained %v items\n", resp.GetCount())
	case "rdrain":
		resp, err := client.Drain(ctx, &pb.DrainRequest{
			DrainType: pb.DrainRequest_JUST_REFRESH,
		})
		if err != nil {
			log.Fatalf("Unable to drain queue: %v", err)
		}
		fmt.Printf("Drained %v items\n", resp.GetCount())
	case "wdrain":
		resp, err := client.Drain(ctx, &pb.DrainRequest{
			DrainType: pb.DrainRequest_JUST_WANTS,
		})
		if err != nil {
			log.Fatalf("Unable to drain queue: %v", err)
		}
		fmt.Printf("Drained %v items\n", resp.GetCount())
	case "sdrain":
		resp, err := client.Drain(ctx, &pb.DrainRequest{
			DrainType: pb.DrainRequest_JUST_SALES,
		})
		if err != nil {
			log.Fatalf("Unable to drain queue: %v", err)
		}
		fmt.Printf("Drained %v items\n", resp.GetCount())
	case "drain":
		resp, err := client.Drain(ctx, &pb.DrainRequest{})
		if err != nil {
			log.Fatalf("Unable to drain queue: %v", err)
		}
		fmt.Printf("Drained %v items\n", resp.GetCount())
	case "users":
		users, err := sclient.GetUsers(ctx, &pb.GetUsersRequest{})
		if err != nil {
			log.Fatalf("Error getting users: %v", err)
		}
		fmt.Printf("%v users\n", len(users.GetUsers()))
		for _, user := range users.GetUsers() {
			fmt.Printf("%v\n", user)
		}
	case "waitlist":
		users, err := sclient.GetWaitlistStatus(ctx, &pb.GetWaitlistStatusRequest{})
		if err != nil {
			log.Fatalf("Error getting waitlist status: %v", err)
		}
		fmt.Print(formatWaitlist(users))
	case "refresh":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshUser{RefreshUser: &pb.RefreshUserEntry{Auth: os.Args[3]}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "collection":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Priority: pb.QueueElement_PRIORITY_HIGH, Force: true, RunDate: time.Now().Add(time.Hour * 24 * -1).UnixNano(), Auth: os.Args[3], Entry: &pb.QueueElement_RefreshCollectionEntry{RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refreshcollection":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Force: true, RunDate: 1234567, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshCollection{RefreshCollection: &pb.RefreshCollection{}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "clean":
		_, err := sclient.Clean(ctx, &pb.CleanRequest{})
		if err != nil {
			log.Fatalf("Error in clean: %v", err)
		}
		log.Printf("Cleaned: %v", err)
	case "clean-wantupdates":
		_, err := sclient.Clean(ctx, &pb.CleanRequest{})
		if err != nil {
			log.Fatalf("Error in clean: %v", err)
		}
		log.Printf("Cleaned: %v", err)
	case "list":
		items, err := client.List(context.Background(), &pb.ListRequest{})
		if err != nil {
			log.Fatalf("Bad list: %v", err)
		}
		for _, item := range items.GetElements() {
			fmt.Printf("%v\n", item)
		}
	case "syncsales":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refresh_release":
		iid, err := strconv.ParseInt(os.Args[4], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse %v -> %v", os.Args[4], err)
		}
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{RunDate: 1222, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshRelease{RefreshRelease: &pb.RefreshRelease{Iid: iid, Intention: "from-cli"}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refresh_wantlists":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Priority: pb.QueueElement_PRIORITY_HIGH, Force: true, RunDate: 1015, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshWantlists{}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refresh_wants":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Force: true, RunDate: 10, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshWants{}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refresh_want":
		id, err := strconv.ParseInt(os.Args[4], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse %v -> %v", os.Args[4], err)
		}
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Priority: pb.QueueElement_PRIORITY_HIGH, Force: true, RunDate: 10, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshWant{RefreshWant: &pb.RefreshWant{Want: &pb.Want{Id: id}}}},
		})
		fmt.Printf("%v and %v\n", a, b)

	case "refresh_release_date":
		iid, err := strconv.ParseInt(os.Args[4], 10, 64)
		mid, err := strconv.ParseInt(os.Args[5], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse %v -> %v", os.Args[4], err)
		}
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Force: true, RunDate: 123456, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshEarliestReleaseDates{RefreshEarliestReleaseDates: &pb.RefreshEarliestReleaseDates{Iid: iid, MasterId: mid}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refresh_release_dates":
		iid, err := strconv.ParseInt(os.Args[4], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse %v -> %v", os.Args[4], err)
		}
		rid, err := strconv.ParseInt(os.Args[5], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse %v -> %v", os.Args[5], err)
		}
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Force: true, RunDate: 10, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshEarliestReleaseDate{RefreshEarliestReleaseDate: &pb.RefreshEarliestReleaseDate{Iid: iid, OtherRelease: rid, UpdateDigitalWantlist: true}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refresh_master":
		iid, err := strconv.ParseInt(os.Args[4], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse %v -> %v", os.Args[4], err)
		}
		mid, err := strconv.ParseInt(os.Args[5], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse")
		}
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshEarliestReleaseDates{RefreshEarliestReleaseDates: &pb.RefreshEarliestReleaseDates{Iid: iid, MasterId: mid}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "adjustsales":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Force: true, RunDate: 1718597532322472889, Auth: os.Args[3], Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	}
}

func formatWaitlist(res *pb.GetWaitlistStatusResponse) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v users in waitlist\n", len(res.GetUsers())))
	sb.WriteString(fmt.Sprintf("%-20s | %-16s | %-10s | %-10s\n", "User", "Progress", "ETA", "Status"))
	sb.WriteString(strings.Repeat("-", 65) + "\n")
	
	for _, u := range res.GetUsers() {
		username := ""
		if u.GetUser() != nil && u.GetUser().GetUser() != nil {
			username = u.GetUser().GetUser().GetUsername()
		}
		
		status := "Partially Synced"
		if u.GetFullySynced() {
			status = "Fully Synced"
		} else if u.GetIsStuck() {
			status = "STUCK"
		}
		
		totalExpected := int32(0)
		if u.GetUser() != nil {
			totalExpected = u.GetUser().GetExpectedCollectionSize() + u.GetUser().GetExpectedWantlistSize()
		}
		totalSynced := u.GetSyncedCollectionSize() + u.GetSyncedWantlistSize()
		progress := fmt.Sprintf("%d/%d", totalSynced, totalExpected)
		
		eta := fmt.Sprintf("%ds", u.GetEtaSeconds())
		if u.GetEtaSeconds() > 3600 {
			eta = fmt.Sprintf("%dh%dm", u.GetEtaSeconds()/3600, (u.GetEtaSeconds()%3600)/60)
		} else if u.GetEtaSeconds() > 60 {
			eta = fmt.Sprintf("%dm%ds", u.GetEtaSeconds()/60, u.GetEtaSeconds()%60)
		} else if u.GetEtaSeconds() == 0 {
			eta = "-"
		}
		
		sb.WriteString(fmt.Sprintf("%-20s | %-16s | %-10s | %-10s\n", username, progress, eta, status))
	}
	return sb.String()
}

