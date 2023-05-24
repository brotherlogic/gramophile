package main

import "testing"
import "context"
import rstore_client "github.com/brotherlogic/rstore/client"

func GetTestQueue() *queue {
	tc, err := rstore_client.GetTestClient()
	if err != nil {
		panic(err)
	}
	return &queue{
		rstore: tc,
	}
}

func TestRunWithEmptyQueue(t *testing.T) {
	q := GetTestQueue()

	elem, err := q.getNextEntry(context.Background())
	if err == nil {
		t.Errorf("Should have failed: %v, %v", elem, err)
	}
}
