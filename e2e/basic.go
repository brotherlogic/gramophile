package e2e

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

// Basic test:
//
//  1. Create a record
//  2. Set the width of the record
//  3. Delete the user data
//  4. Run a user update
//  5. Confirm record retrieval
//  6. Delete record
func RunBasicTest(ctx context.Context, gclient *pb.GramophileServiceClient, qclient *pb.QueueServiceClient) {

}
