package background

import (
	"context"
	"fmt"
	"log"

	"github.com/brotherlogic/gramophile/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type BackgroundRunner struct {
	db                    db.Database
	key, secret, callback string
	ReleaseRefresh        int64
}

func GetBackgroundRunner(db db.Database, key, secret, callback string) *BackgroundRunner {
	return &BackgroundRunner{db: db, key: key, secret: secret, callback: callback}
}

func getRefKey(ctx context.Context) (string, error) {
	md, found := metadata.FromIncomingContext(ctx)
	if found {
		if _, ok := md["queue-key"]; ok {
			idt := md["queue-key"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	md, found = metadata.FromOutgoingContext(ctx)
	if found {
		if _, ok := md["queue-key"]; ok {
			idt := md["queue-key"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	return "", status.Errorf(codes.NotFound, "Could not extract token from incoming or outgoing")
}

func qlog(ctx context.Context, str string, v ...any) {
	key, err := getRefKey(ctx)
	if err != nil {
		log.Printf("Unable to get ref key: %v", err)
		log.Printf(str, v...)
		return
	}

	prefix := fmt.Sprintf("%v: ", key)
	log.Printf(prefix+str, v...)
}
