package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
	pspb "github.com/brotherlogic/pstore/proto"
	"github.com/golang/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*60)
	defer cancel()

	conn, err := grpc.Dial(os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Bad dial: %v", err)
	}

	client := pspb.NewPStoreServiceClient(conn)

	keys, err := client.GetKeys(ctx, &pspb.GetKeysRequest{
		Prefix: "gramophile/taskqueue/",
	})

	if err != nil {
		log.Fatalf("Bad get keys: %v", err)
	}

	log.Printf("Found %v keys", len(keys.GetKeys()))

	tcount := make(map[string]int)
	icount := make(map[int64]int)
	for _, key := range keys.GetKeys() {
		val, err := client.Read(ctx, &pspb.ReadRequest{Key: key})
		if err != nil {
			log.Printf("Error reading key %v: %v", key, err)
		} else {
			entry := &pb.QueueElement{}
			err = proto.Unmarshal(val.GetValue().GetValue(), entry)
			if err != nil {
				log.Printf("Error unmarshalling key %v: %v", key, err)
			} else {
				tcount[fmt.Sprintf("%T", entry.GetEntry())]++
				if strings.Contains(fmt.Sprintf("%T", entry.GetEntry()), "RefreshCollectionEntry") {
					icount[entry.GetRefreshCollectionEntry().GetRefreshId()]++
				}
			}
		}
	}

	for k, v := range tcount {
		log.Printf("Type %v: %v entries", k, v)
	}

	for k, v := range icount {
		log.Printf("RefreshID %v: %v entries", k, v)
	}
}
