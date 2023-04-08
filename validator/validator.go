package main

import (
	"context"
	"log"
)

func validateUsers(ctx context.Context) error {

	return nil
}

func main() {
	ctx := context.Background()

	err := validateUsers(ctx)
	if err != nil {
		log.Fatalf("Cannot validate users: %v", err)
	}
}
