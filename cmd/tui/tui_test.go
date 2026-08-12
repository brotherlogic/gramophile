package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
)

type mockClient struct {
	getURLFunc    func() (*pb.GetURLResponse, error)
	getLoginFunc  func() (*pb.GetLoginResponse, error)
	getUserFunc   func() (*pb.GetUserResponse, error)
	getStateFunc  func() (*pb.GetStateResponse, error)
	getOrgFunc    func(*pb.GetOrgRequest) (*pb.GetOrgResponse, error)
	getRecordFunc func(*pb.GetRecordRequest) (*pb.GetRecordResponse, error)
}

func (m *mockClient) GetURL(ctx context.Context, in *pb.GetURLRequest, opts ...grpc.CallOption) (*pb.GetURLResponse, error) {
	if m.getURLFunc != nil {
		return m.getURLFunc()
	}
	return &pb.GetURLResponse{URL: "http://test", Token: "test-token"}, nil
}

func (m *mockClient) GetLogin(ctx context.Context, in *pb.GetLoginRequest, opts ...grpc.CallOption) (*pb.GetLoginResponse, error) {
	if m.getLoginFunc != nil {
		return m.getLoginFunc()
	}
	return &pb.GetLoginResponse{Auth: &pb.GramophileAuth{Token: "final-auth"}}, nil
}

func (m *mockClient) GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc()
	}
	return &pb.GetUserResponse{User: &pb.StoredUser{ExpectedCollectionSize: 100, State: pb.StoredUser_USER_STATE_REFRESHING}}, nil
}

func (m *mockClient) GetState(ctx context.Context, in *pb.GetStateRequest, opts ...grpc.CallOption) (*pb.GetStateResponse, error) {
	if m.getStateFunc != nil {
		return m.getStateFunc()
	}
	return &pb.GetStateResponse{CollectionSize: 50}, nil
}

func (m *mockClient) SetConfig(ctx context.Context, in *pb.SetConfigRequest, opts ...grpc.CallOption) (*pb.SetConfigResponse, error) {
	return &pb.SetConfigResponse{}, nil
}

func (m *mockClient) GetOrg(ctx context.Context, in *pb.GetOrgRequest, opts ...grpc.CallOption) (*pb.GetOrgResponse, error) {
	if m.getOrgFunc != nil {
		return m.getOrgFunc(in)
	}
	return &pb.GetOrgResponse{}, nil
}

func (m *mockClient) GetRecord(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
	if m.getRecordFunc != nil {
		return m.getRecordFunc(in)
	}
	return &pb.GetRecordResponse{}, nil
}

func TestStateTransitions(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})

	if m.state != StateStartupLogo {
		t.Errorf("Expected initial state to be StateStartupLogo, got %v", m.state)
	}

	// Any key press in StateStartupLogo should transition to StateLogin
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}

	if updatedModel.state != StateLogin {
		t.Errorf("Expected state to transition to StateLogin on key press, got %v", updatedModel.state)
	}
	
	// And it should return a command to fetch the URL
	if cmd == nil {
		t.Errorf("Expected a command to be returned to fetch URL")
	}
}

func TestTimerTransition(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	
	// A timeout message should transition to StateLogin
	msg := timeoutMsg{}
	newModel, _ := m.Update(msg)
	
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}

	if updatedModel.state != StateLogin {
		t.Errorf("Expected state to transition to StateLogin on timeout, got %v", updatedModel.state)
	}
}

func TestStateLogin_GetURL(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateLogin

	// Test getting the URL successfully
	msg := urlFetchedMsg{url: "http://test", token: "test-token"}
	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)
	
	if updatedModel.loginURL != "http://test" {
		t.Errorf("Expected loginURL to be http://test, got %v", updatedModel.loginURL)
	}
	
	if cmd == nil {
		t.Errorf("Expected command to poll for login")
	}
}

func TestStateLogin_LoginSuccess(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateLogin
	m.tokenSaver = func(token string) error { return nil } // mock saver
	
	msg := loginSuccessMsg{auth: &pb.GramophileAuth{Token: "test-auth-token"}}
	newModel, _ := m.Update(msg)
	updatedModel := newModel.(Model)
	
	if updatedModel.state != StateLoadingSync {
		t.Errorf("Expected state to transition to StateLoadingSync, got %v", updatedModel.state)
	}
}

func TestStateLoadingSync_Progress(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateLoadingSync

	// Trigger sync poll
	msg := syncPollMsg{}
	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)

	if cmd == nil {
		t.Errorf("Expected command to fetch sync status")
	}

	// Fake the response
	respMsg := syncStatusMsg{
		expectedSize: 100,
		currentSize:  50,
		userState:    pb.StoredUser_USER_STATE_REFRESHING,
	}

	newModel, _ = updatedModel.Update(respMsg)
	updatedModel = newModel.(Model)

	if updatedModel.progress != 0.5 {
		t.Errorf("Expected progress to be 0.5, got %v", updatedModel.progress)
	}

	view := updatedModel.View()
	if view == "" {
		t.Errorf("Expected a progress bar view")
	}
}

func TestStateLoadingSync_Complete(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateLoadingSync

	respMsg := syncStatusMsg{
		expectedSize: 100,
		currentSize:  100,
		userState:    pb.StoredUser_USER_STATE_IN_WAITLIST,
	}

	newModel, _ := m.Update(respMsg)
	updatedModel := newModel.(Model)

	if updatedModel.state != StateWaitlist {
		t.Errorf("Expected state to transition to StateWaitlist on complete, got %v", updatedModel.state)
	}
}

func TestStateWaitlist_Poll(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateWaitlist

	msg := syncPollMsg{}
	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)

	if cmd == nil {
		t.Errorf("Expected command to fetch sync status in waitlist")
	}

	respMsg := syncStatusMsg{
		userState: pb.StoredUser_USER_STATE_IN_WAITLIST,
	}

	newModel, _ = updatedModel.Update(respMsg)
	updatedModel = newModel.(Model)

	if updatedModel.state != StateWaitlist {
		t.Errorf("Expected state to remain StateWaitlist, got %v", updatedModel.state)
	}
}

func TestStateWaitlist_Promoted(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateWaitlist

	respMsg := syncStatusMsg{
		userState: pb.StoredUser_USER_STATE_LIVE,
	}

	newModel, _ := m.Update(respMsg)
	updatedModel := newModel.(Model)

	if updatedModel.state != StateMainApp {
		t.Errorf("Expected state to transition to StateMainApp on promotion, got %v", updatedModel.state)
	}
}

func TestFaultTolerance_ExponentialBackoff(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateLoadingSync

	if m.syncRetryCount != 0 {
		t.Errorf("Expected initial syncRetryCount to be 0")
	}

	respMsg := syncStatusMsg{err: fmt.Errorf("connection refused")}

	newModel, _ := m.Update(respMsg)
	updatedModel := newModel.(Model)

	if updatedModel.syncRetryCount != 1 {
		t.Errorf("Expected syncRetryCount to be 1, got %v", updatedModel.syncRetryCount)
	}
	
	// Test it again to see backoff increase
	newModel, _ = updatedModel.Update(respMsg)
	updatedModel = newModel.(Model)
	
	if updatedModel.syncRetryCount != 2 {
		t.Errorf("Expected syncRetryCount to be 2, got %v", updatedModel.syncRetryCount)
	}
}

func TestOrgCommandParsing(t *testing.T) {
	orgName, slot, hash, debug, err := parseOrgCommand("org --org MyCollection --slot 2 --hash abc1234 --debug")
	if err != nil {
		t.Fatalf("Unexpected error parsing org command: %v", err)
	}
	if orgName != "MyCollection" {
		t.Errorf("Expected orgName to be MyCollection, got %v", orgName)
	}
	if slot != 2 {
		t.Errorf("Expected slot to be 2, got %v", slot)
	}
	if hash != "abc1234" {
		t.Errorf("Expected hash to be abc1234, got %v", hash)
	}
	if !debug {
		t.Errorf("Expected debug to be true, got %v", debug)
	}

	// Test positional org name with orgview
	orgName2, slot2, hash2, debug2, err2 := parseOrgCommand("orgview PositionalOrg")
	if err2 != nil {
		t.Fatalf("Unexpected error parsing orgview command: %v", err2)
	}
	if orgName2 != "PositionalOrg" {
		t.Errorf("Expected orgName to be PositionalOrg, got %v", orgName2)
	}
	if slot2 != 0 || hash2 != "" || debug2 {
		t.Errorf("Expected default values for slot/hash/debug, got slot=%v, hash=%v, debug=%v", slot2, hash2, debug2)
	}

	// Test invalid command prefix
	_, _, _, _, errInvalid := parseOrgCommand("invalidcommand MyOrg")
	if errInvalid == nil {
		t.Errorf("Expected error for non-org command prefix")
	}
}

func TestStateOrgViewTransition(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateMainApp

	newModel, _ := m.handleCommandInput("org --org MyOrg --slot 1")
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}

	if updatedModel.state != StateOrgView {
		t.Errorf("Expected state to transition to StateOrgView, got %v", updatedModel.state)
	}
	if updatedModel.activeOrgName != "MyOrg" {
		t.Errorf("Expected activeOrgName to be MyOrg, got %v", updatedModel.activeOrgName)
	}
	if updatedModel.activeSlot != 1 {
		t.Errorf("Expected activeSlot to be 1, got %v", updatedModel.activeSlot)
	}
}

func TestMockClientGetOrgAndGetRecord(t *testing.T) {
	var client OrgClient = &mockClient{
		getOrgFunc: func(req *pb.GetOrgRequest) (*pb.GetOrgResponse, error) {
			return &pb.GetOrgResponse{
				Snapshot: &pb.OrganisationSnapshot{
					Hash: "test-hash",
				},
			}, nil
		},
		getRecordFunc: func(req *pb.GetRecordRequest) (*pb.GetRecordResponse, error) {
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								Id: 12345,
							},
						},
					},
				},
			}, nil
		},
	}

	orgResp, err := client.GetOrg(context.Background(), &pb.GetOrgRequest{OrgName: "test-org"})
	if err != nil {
		t.Fatalf("GetOrg returned error: %v", err)
	}
	if orgResp.GetSnapshot().GetHash() != "test-hash" {
		t.Errorf("Expected snapshot hash 'test-hash', got '%s'", orgResp.GetSnapshot().GetHash())
	}

	recResp, err := client.GetRecord(context.Background(), &pb.GetRecordRequest{})
	if err != nil {
		t.Fatalf("GetRecord returned error: %v", err)
	}
	if len(recResp.GetRecords()) != 1 || recResp.GetRecords()[0].GetRecord().GetRelease().GetId() != 12345 {
		t.Errorf("Unexpected GetRecord response: %+v", recResp)
	}
}

func TestOrgFetchedAndRecordResolution(t *testing.T) {
	mock := &mockClient{
		getOrgFunc: func(req *pb.GetOrgRequest) (*pb.GetOrgResponse, error) {
			return &pb.GetOrgResponse{
				Snapshot: &pb.OrganisationSnapshot{
					Hash: "hash-123",
					Placements: []*pb.Placement{
						{Iid: 101, Space: "MainShelf", Unit: 1, Index: 1, Width: 12.5},
						{Iid: 102, Space: "MainShelf", Unit: 1, Index: 2, Width: 15.0},
					},
				},
			}, nil
		},
		getRecordFunc: func(req *pb.GetRecordRequest) (*pb.GetRecordResponse, error) {
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								Id:      101,
								Title:   "Blue Train",
								Artists: []*pbd.Artist{{Name: "John Coltrane"}},
							},
							Width: 12.5,
						},
					},
				},
			}, nil
		},
	}

	m := InitialModel(mock, mock)
	m.state = StateMainApp

	newModel, cmd := m.handleCommandInput("org --org TestOrg")
	m = newModel.(Model)

	if m.state != StateOrgView {
		t.Fatalf("Expected state StateOrgView, got %v", m.state)
	}

	if cmd == nil {
		t.Fatalf("Expected fetchOrgCmd to be returned")
	}
	orgMsg := cmd()
	fetchedMsg, ok := orgMsg.(orgFetchedMsg)
	if !ok {
		t.Fatalf("Expected orgFetchedMsg, got %T", orgMsg)
	}

	newModel, batchCmd := m.Update(fetchedMsg)
	m = newModel.(Model)

	if len(m.orgPlacements) != 2 {
		t.Errorf("Expected 2 placements, got %d", len(m.orgPlacements))
	}

	viewContent := m.orgViewport.View()
	if !strings.Contains(viewContent, "Loading...") {
		t.Errorf("Expected viewport to initially contain placeholder Loading..., got:\n%s", viewContent)
	}

	if batchCmd == nil {
		t.Fatalf("Expected batch fetchRecordCmd batch to be returned")
	}

	recMsg1 := recordFetchedMsg{
		iid: 101,
		record: &pb.Record{
			Release: &pbd.Release{
				Id:      101,
				Title:   "Blue Train",
				Artists: []*pbd.Artist{{Name: "John Coltrane"}},
			},
		},
	}
	newModel, _ = m.Update(recMsg1)
	m = newModel.(Model)

	viewContentUpdated := m.orgViewport.View()
	if !strings.Contains(viewContentUpdated, "John Coltrane - Blue Train") {
		t.Errorf("Expected viewport to contain resolved title John Coltrane - Blue Train, got:\n%s", viewContentUpdated)
	}
}

func TestOrgViewNavigationAndExit(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{})
	m.state = StateOrgView
	m.orgViewport = viewport.New(80, 5)
	m.orgViewport.SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10")

	navKeys := []string{"down", "j", "up", "k", "pgdown", "pgup"}
	for _, k := range navKeys {
		var keyMsg tea.KeyMsg
		switch k {
		case "down":
			keyMsg = tea.KeyMsg{Type: tea.KeyDown}
		case "up":
			keyMsg = tea.KeyMsg{Type: tea.KeyUp}
		case "pgdown":
			keyMsg = tea.KeyMsg{Type: tea.KeyPgDown}
		case "pgup":
			keyMsg = tea.KeyMsg{Type: tea.KeyPgUp}
		default:
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
		if m.state != StateOrgView {
			t.Errorf("Key %s changed state unexpectedly to %v", k, m.state)
		}
	}

	exitKeys := []string{"x", "q", "esc"}
	for _, ek := range exitKeys {
		m.state = StateOrgView
		var keyMsg tea.KeyMsg
		if ek == "esc" {
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		} else {
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(ek)}
		}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
		if m.state != StateMainApp {
			t.Errorf("Expected exit key %s to transition to StateMainApp, got %v", ek, m.state)
		}
	}
}


