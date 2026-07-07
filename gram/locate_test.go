package main

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
)

type testClient struct {
	pb.GramophileEServiceClient
	withArtist bool
}

func (t *testClient) GetRecord(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
	artists := []*pbd.Artist{}
	if t.withArtist {
		artists = append(artists, &pbd.Artist{Name: "The Beatles"})
	}

	return &pb.GetRecordResponse{
		Records: []*pb.RecordResponse{
			{
				Record: &pb.Record{
					Release: &pbd.Release{
						Title:   "Abbey Road",
						Artists: artists,
					},
				},
			},
		},
	}, nil
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

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		before int
		after  int
		want   float64
	}{
		{0, 0, 100.0},
		{1, 1, 66.66666666666666},
		{0, 1, 50.0},
		{1, 0, 100.0},
		{9, 0, 100.0},
		{0, 9, 10.0},
		{4, 5, 50.0},
	}

	for _, tt := range tests {
		got := calculatePercentage(tt.before, tt.after)
		if got != tt.want {
			t.Errorf("calculatePercentage(%v, %v) = %v, want %v", tt.before, tt.after, got, tt.want)
		}
	}
}

func TestGetTitle_WithArtist(t *testing.T) {
	client := &testClient{withArtist: true}
	title := getTitle(context.Background(), client, 123)
	expected := "The Beatles - Abbey Road"
	if title != expected {
		t.Errorf("Expected %v, got %v", expected, title)
	}
}

func TestGetTitle_WithoutArtist(t *testing.T) {
	client := &testClient{withArtist: false}
	title := getTitle(context.Background(), client, 123)
	expected := "Abbey Road"
	if title != expected {
		t.Errorf("Expected %v, got %v", expected, title)
	}
}

func TestGetTitleFromRelease_WithArtist(t *testing.T) {
	client := &testClient{withArtist: true}
	title := getTitleFromRelease(context.Background(), client, 123)
	expected := "The Beatles - Abbey Road"
	if title != expected {
		t.Errorf("Expected %v, got %v", expected, title)
	}
}

func TestGetTitleFromRelease_WithoutArtist(t *testing.T) {
	client := &testClient{withArtist: false}
	title := getTitleFromRelease(context.Background(), client, 123)
	expected := "Abbey Road"
	if title != expected {
		t.Errorf("Expected %v, got %v", expected, title)
	}
}
