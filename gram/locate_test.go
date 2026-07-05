package main

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs/proto"
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

func TestGetLocate(t *testing.T) {
	module := GetLocate()
	if module == nil {
		t.Fatalf("GetLocate returned nil")
	}
	if module.Command != "locate" {
		t.Errorf("Expected command 'locate', got %v", module.Command)
	}
}

func TestGetTitleWithArtist(t *testing.T) {
	client := &mockClient{
		records: []*pb.RecordResponse{
			{
				Record: &pb.Record{
					Release: &proto.Release{
						Title: "Some Title",
						Artists: []*proto.Artist{
							{Name: "Some Artist"},
						},
					},
				},
			},
		},
	}

	title := getTitle(context.Background(), client, 123)
	if title != "Some Artist - Some Title" {
		t.Errorf("Expected 'Some Artist - Some Title', got %v", title)
	}

	titleRelease := getTitleFromRelease(context.Background(), client, 123)
	if titleRelease != "Some Artist - Some Title" {
		t.Errorf("Expected 'Some Artist - Some Title', got %v", titleRelease)
	}
}

func TestGetTitleWithoutArtist(t *testing.T) {
	client := &mockClient{
		records: []*pb.RecordResponse{
			{
				Record: &pb.Record{
					Release: &proto.Release{
						Title: "Some Title",
					},
				},
			},
		},
	}

	title := getTitle(context.Background(), client, 123)
	if title != "Some Title" {
		t.Errorf("Expected 'Some Title', got %v", title)
	}

	titleRelease := getTitleFromRelease(context.Background(), client, 123)
	if titleRelease != "Some Title" {
		t.Errorf("Expected 'Some Title', got %v", titleRelease)
	}
}
