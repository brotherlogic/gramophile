package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"google.golang.org/grpc"
)

type mockClient struct {
	getURLFunc   func() (*pb.GetURLResponse, error)
	getLoginFunc func() (*pb.GetLoginResponse, error)
	getUserFunc  func() (*pb.GetUserResponse, error)
	getStateFunc func() (*pb.GetStateResponse, error)
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

func TestStateTransitions(t *testing.T) {
	m := InitialModel(&mockClient{})

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
	m := InitialModel(&mockClient{})
	
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
	m := InitialModel(&mockClient{})
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
	m := InitialModel(&mockClient{})
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
	m := InitialModel(&mockClient{})
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
	m := InitialModel(&mockClient{})
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
	m := InitialModel(&mockClient{})
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
	m := InitialModel(&mockClient{})
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
	m := InitialModel(&mockClient{})
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

func TestOrgErrorInlineReporting(t *testing.T) {
	client := &mockOrgClient{
		setConfigFunc: func(req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
			return nil, fmt.Errorf("gRPC communication failure")
		},
	}

	m := InitialModel(client)
	m.state = StateOrgConfig
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "Inbox", Id: 123}},
		Config:  &pb.GramophileConfig{},
	}
	m.initOrgConfigForm()
	m.orgName = "Valid Org"
	m.spaceName = "Main Shelf"
	m.spaceUnits = "2"
	m.spaceWidth = "12.5"
	m.selectedFolders = []string{"123"}
	m.sortStrategy = "RELEASE_YEAR"
	m.form.State = huh.StateCompleted

	newModel, cmd := m.Update(tea.WindowSizeMsg{})
	updatedModel := newModel.(Model)
	if cmd == nil {
		t.Fatalf("Expected cmd to call SetConfig")
	}

	msg := cmd()
	newModel, _ = updatedModel.Update(msg)
	updatedModel = newModel.(Model)

	if updatedModel.state != StateOrgConfig {
		t.Errorf("Expected state to remain StateOrgConfig on gRPC error, got %v", updatedModel.state)
	}
	if updatedModel.form == nil {
		t.Errorf("Expected form to not be nil on gRPC error")
	}
	view := updatedModel.View()
	if !strings.Contains(view, "gRPC communication failure") {
		t.Errorf("Expected view to contain inline error message 'gRPC communication failure', got:\n%s", view)
	}

	// Test case 2: Empty organization name validation
	m2 := InitialModel(client)
	m2.state = StateOrgConfig
	m2.user = &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "Inbox", Id: 123}},
		Config:  &pb.GramophileConfig{},
	}
	m2.initOrgConfigForm()
	m2.orgName = ""
	m2.spaceName = "Main Shelf"
	m2.spaceUnits = "2"
	m2.spaceWidth = "12.5"
	m2.selectedFolders = []string{"123"}
	m2.sortStrategy = "RELEASE_YEAR"
	m2.form.State = huh.StateCompleted

	newModel2, _ := m2.Update(tea.WindowSizeMsg{})
	updatedModel2 := newModel2.(Model)
	if updatedModel2.state != StateOrgConfig {
		t.Errorf("Expected state to remain StateOrgConfig on invalid org name, got %v", updatedModel2.state)
	}
	view2 := updatedModel2.View()
	if !strings.Contains(view2, "Organization name cannot be empty") && !strings.Contains(view2, "invalid organization name") && !strings.Contains(view2, "cannot be empty") {
		t.Errorf("Expected view to contain inline error message for empty org name, got:\n%s", view2)
	}

	// Test case 3: Empty placement list (selectedFolders) validation
	m3 := InitialModel(client)
	m3.state = StateOrgConfig
	m3.user = &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "Inbox", Id: 123}},
		Config:  &pb.GramophileConfig{},
	}
	m3.initOrgConfigForm()
	m3.orgName = "Valid Org"
	m3.spaceName = "Main Shelf"
	m3.spaceUnits = "2"
	m3.spaceWidth = "12.5"
	m3.selectedFolders = nil
	m3.sortStrategy = "RELEASE_YEAR"
	m3.form.State = huh.StateCompleted

	newModel3, _ := m3.Update(tea.WindowSizeMsg{})
	updatedModel3 := newModel3.(Model)
	if updatedModel3.state != StateOrgConfig {
		t.Errorf("Expected state to remain StateOrgConfig on empty placement list, got %v", updatedModel3.state)
	}
	view3 := updatedModel3.View()
	if !strings.Contains(view3, "placement") && !strings.Contains(view3, "folder") {
		t.Errorf("Expected view to contain inline error message for empty placement list, got:\n%s", view3)
	}
}
