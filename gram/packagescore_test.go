package main

import (
	"context"
	"strings"
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockPackageScoreClient struct {
	pb.GramophileEServiceClient
	lastReq *pb.SetIntentRequest
	err     error
}

func (m *mockPackageScoreClient) SetIntent(ctx context.Context, in *pb.SetIntentRequest, opts ...grpc.CallOption) (*pb.SetIntentResponse, error) {
	m.lastReq = in
	return &pb.SetIntentResponse{}, m.err
}

func TestGetPackageScore(t *testing.T) {
	module := GetPackageScore()
	if module == nil {
		t.Fatalf("GetPackageScore returned nil")
	}
	if module.Command != "packagescore" {
		t.Errorf("Expected command 'packagescore', got %q", module.Command)
	}
	if module.Execute == nil {
		t.Errorf("Expected Execute function to be set")
	}
}

func TestRunPackageScore_Success(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedIid   int64
		expectedScore int32
	}{
		{
			name:          "score 0",
			args:          []string{"12345", "0"},
			expectedIid:   12345,
			expectedScore: 0,
		},
		{
			name:          "score 5",
			args:          []string{"99999", "5"},
			expectedIid:   99999,
			expectedScore: 5,
		},
		{
			name:          "score 3",
			args:          []string{"45678", "3"},
			expectedIid:   45678,
			expectedScore: 3,
		},
		{
			name:          "score -1 (reset)",
			args:          []string{"12345", "-1"},
			expectedIid:   12345,
			expectedScore: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPackageScoreClient{}
			err := runPackageScore(context.Background(), mock, tt.args)
			if err != nil {
				t.Fatalf("runPackageScore failed: %v", err)
			}
			if mock.lastReq == nil {
				t.Fatalf("SetIntent was not called")
			}
			if mock.lastReq.GetInstanceId() != tt.expectedIid {
				t.Errorf("Expected instance ID %d, got %d", tt.expectedIid, mock.lastReq.GetInstanceId())
			}
			if mock.lastReq.GetIntent().GetPackageScore() != tt.expectedScore {
				t.Errorf("Expected package score %d, got %d", tt.expectedScore, mock.lastReq.GetIntent().GetPackageScore())
			}
		})
	}
}

func TestRunPackageScore_ValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedCode codes.Code
	}{
		{
			name:         "too few args (0)",
			args:         []string{},
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "too few args (1)",
			args:         []string{"12345"},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "invalid iid",
			args: []string{"not-a-number", "3"},
		},
		{
			name: "invalid score string",
			args: []string{"12345", "not-a-number"},
		},
		{
			name:         "score below -1",
			args:         []string{"12345", "-2"},
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "score above 5",
			args:         []string{"12345", "6"},
			expectedCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPackageScoreClient{}
			err := runPackageScore(context.Background(), mock, tt.args)
			if err == nil {
				t.Fatalf("Expected error for args %v, got nil", tt.args)
			}
			if tt.expectedCode != codes.OK && status.Code(err) != tt.expectedCode {
				t.Errorf("Expected status code %v for args %v, got %v", tt.expectedCode, tt.args, status.Code(err))
			}
			if strings.HasPrefix(tt.name, "too few args") {
				if !strings.Contains(err.Error(), "usage: gram packagescore <iid> <-1-5>") {
					t.Errorf("Expected usage 'usage: gram packagescore <iid> <-1-5>', got %v", err)
				}
			}
		})
	}
}

func TestRunPackageScore_ClientError(t *testing.T) {
	mock := &mockPackageScoreClient{
		err: status.Errorf(codes.Internal, "rpc failure"),
	}
	err := runPackageScore(context.Background(), mock, []string{"12345", "4"})
	if err == nil {
		t.Fatalf("Expected error from rpc failure, got nil")
	}
	if !strings.Contains(err.Error(), "rpc failure") {
		t.Errorf("Expected error to contain 'rpc failure', got %v", err)
	}
}
