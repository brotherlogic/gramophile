package main

import (
	"context"
	"testing"

	discogs "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
)

type mockClient struct {
	pb.GramophileEServiceClient
	records []*pb.RecordResponse
}

func (m *mockClient) GetRecord(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
	return &pb.GetRecordResponse{Records: m.records}, nil
}

func TestGetTitle_WithArtist(t *testing.T) {
	client := &mockClient{
		records: []*pb.RecordResponse{
			{
				Record: &pb.Record{
					Release: &discogs.Release{
						Title: "Some Title",
						Artists: []*discogs.Artist{
							{Name: "Some Artist"},
						},
					},
				},
			},
		},
	}

	title := getTitle(context.Background(), client, 123)
	expected := "Some Artist - Some Title"
	if title != expected {
		t.Errorf("Expected %v, got %v", expected, title)
	}
}

func TestGetTitle_WithoutArtist(t *testing.T) {
	client := &mockClient{
		records: []*pb.RecordResponse{
			{
				Record: &pb.Record{
					Release: &discogs.Release{
						Title: "Some Title",
					},
				},
			},
		},
	}

	title := getTitle(context.Background(), client, 123)
	expected := "Some Title"
	if title != expected {
		t.Errorf("Expected %v, got %v", expected, title)
	}
}

func TestGetLocate(t *testing.T) {
	module := GetLocate()
	if module == nil {
		t.Fatalf("GetLocate returned nil")
	}
	if module.Command != "locate" {
		t.Errorf("Expected command 'locate', got %v", module.Command)
	}
}
