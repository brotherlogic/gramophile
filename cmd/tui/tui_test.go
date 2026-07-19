package tui

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
	tea "github.com/charmbracelet/bubbletea"
)

type mockClient struct {
	getURLFunc   func() (*pb.GetURLResponse, error)
	getLoginFunc func() (*pb.GetLoginResponse, error)
	getUserFunc  func() (*pb.GetUserResponse, error)
	getStateFunc func() (*pb.GetStateResponse, error)
}

func (m *mockClient) GetURL(ctx context.Context, in *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	if m.getURLFunc != nil {
		return m.getURLFunc()
	}
	return &pb.GetURLResponse{URL: "http://test", Token: "test-token"}, nil
}

func (m *mockClient) GetLogin(ctx context.Context, in *pb.GetLoginRequest) (*pb.GetLoginResponse, error) {
	if m.getLoginFunc != nil {
		return m.getLoginFunc()
	}
	return &pb.GetLoginResponse{Auth: &pb.GramophileAuth{Token: "final-auth"}}, nil
}

func (m *mockClient) GetUser(ctx context.Context, in *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc()
	}
	return &pb.GetUserResponse{User: &pb.StoredUser{ExpectedCollectionSize: 100, State: pb.StoredUser_USER_STATE_REFRESHING}}, nil
}

func (m *mockClient) GetState(ctx context.Context, in *pb.GetStateRequest) (*pb.GetStateResponse, error) {
	if m.getStateFunc != nil {
		return m.getStateFunc()
	}
	return &pb.GetStateResponse{CollectionSize: 50}, nil
}

func (m *mockClient) SetConfig(ctx context.Context, in *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
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
