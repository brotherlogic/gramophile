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
	"github.com/charmbracelet/huh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockClient struct {
	getURLFunc       func() (*pb.GetURLResponse, error)
	getLoginFunc     func() (*pb.GetLoginResponse, error)
	getUserFunc      func() (*pb.GetUserResponse, error)
	getUserCtxFunc   func(context.Context) (*pb.GetUserResponse, error)
	getStateFunc     func() (*pb.GetStateResponse, error)
	getOrgFunc       func(*pb.GetOrgRequest) (*pb.GetOrgResponse, error)
	getRecordFunc    func(*pb.GetRecordRequest) (*pb.GetRecordResponse, error)
	locateRecordFunc func(*pb.LocateRecordRequest) (*pb.LocateRecordResponse, error)
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
	if m.getUserCtxFunc != nil {
		return m.getUserCtxFunc(ctx)
	}
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

func (m *mockClient) LocateRecord(ctx context.Context, in *pb.LocateRecordRequest, opts ...grpc.CallOption) (*pb.LocateRecordResponse, error) {
	if m.locateRecordFunc != nil {
		return m.locateRecordFunc(in)
	}
	return &pb.LocateRecordResponse{}, nil
}

func TestInitialModel_LocateClient(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	if m.locateClient == nil {
		t.Errorf("Expected locateClient to be initialized")
	}
	expectedPlaceholder := "locate <release_id> | org [name] | configure | quit"
	if m.textInput.Placeholder != expectedPlaceholder {
		t.Errorf("Expected placeholder %q, got %q", expectedPlaceholder, m.textInput.Placeholder)
	}
}


func TestStateTransitions(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})

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

func TestStartupLogoView(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
	view := m.View()

	if !strings.Contains(view, "██████╗") || !strings.Contains(view, "Press any key to continue...") {
		t.Errorf("Expected startup logo view to contain ASCII art logo with '██████╗', got:\n%s", view)
	}
}

func TestLogoPersistsAcrossViews(t *testing.T) {
	mock := &mockClient{}
	states := []appState{
		StateStartupLogo,
		StateLogin,
		StateLoadingSync,
		StateWaitlist,
		StateMainApp,
		StateOrgConfig,
		StateOrgView,
		StateLocateView,
		StateConfigSelect,
	}

	for _, s := range states {
		m := InitialModel(mock, mock, mock)
		m.state = s
		view := m.View()
		if !strings.Contains(view, "██████╗") {
			t.Errorf("Expected state %v view to contain ASCII art logo with '██████╗', got:\n%s", s, view)
		}
	}
}

func TestTimerTransition(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
	
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
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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

func TestStateLoadingSync_LiveUserSkipsWaitlist(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
	m.state = StateLoadingSync

	respMsg := syncStatusMsg{
		expectedSize: 100,
		currentSize:  100,
		userState:    pb.StoredUser_USER_STATE_LIVE,
	}

	newModel, _ := m.Update(respMsg)
	updatedModel := newModel.(Model)

	if updatedModel.state != StateMainApp {
		t.Errorf("Expected state to transition to StateMainApp for live user, got %v", updatedModel.state)
	}
}

func TestStateWaitlist_View(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
	m.state = StateWaitlist

	// Default view without user
	view := m.View()
	if !strings.Contains(view, "Waiting for Admin Approval (USER_STATE_UNKNOWN)...") {
		t.Errorf("Expected default waitlist message with enum, got %q", view)
	}

	// View with user and username
	m.user = &pb.StoredUser{
		State: pb.StoredUser_USER_STATE_IN_WAITLIST,
		User: &pbd.User{
			Username: "brotherlogic",
		},
	}
	view = m.View()
	if !strings.Contains(view, "Waiting for Admin Approval for brotherlogic (USER_STATE_IN_WAITLIST)...") {
		t.Errorf("Expected waitlist message with username and enum, got %q", view)
	}

	// View with user but empty username
	m.user = &pb.StoredUser{
		State: pb.StoredUser_USER_STATE_IN_WAITLIST,
		User: &pbd.User{
			Username: "",
		},
	}
	view = m.View()
	if !strings.Contains(view, "Waiting for Admin Approval (USER_STATE_IN_WAITLIST)...") || strings.Contains(view, "Waiting for Admin Approval for") {
		t.Errorf("Expected fallback waitlist message for empty username with enum, got %q", view)
	}
}

func TestStateWaitlist_UpdateUser(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
	m.state = StateWaitlist

	respMsg := syncStatusMsg{
		userState: pb.StoredUser_USER_STATE_IN_WAITLIST,
		user: &pb.StoredUser{
			State: pb.StoredUser_USER_STATE_IN_WAITLIST,
			User: &pbd.User{
				Username: "testuser",
			},
		},
	}

	newModel, _ := m.Update(respMsg)
	updatedModel := newModel.(Model)

	if updatedModel.user == nil || updatedModel.user.GetUser().GetUsername() != "testuser" {
		t.Errorf("Expected user to be updated in StateWaitlist")
	}

	view := updatedModel.View()
	if !strings.Contains(view, "Waiting for Admin Approval for testuser (USER_STATE_IN_WAITLIST)...") {
		t.Errorf("Expected waitlist view to render testuser and state enum, got %q", view)
	}
}

func TestFaultTolerance_ExponentialBackoff(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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

	m := InitialModel(client, client, client)
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
	m2 := InitialModel(client, client, client)
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
	m3 := InitialModel(client, client, client)
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

	// Test multi-word positional org name without flag (Issue #2356: org 12 Inches -> "12 Inches")
	orgName3, _, _, _, err3 := parseOrgCommand("org 12 Inches")
	if err3 != nil {
		t.Fatalf("Unexpected error parsing multi-word org: %v", err3)
	}
	if orgName3 != "12 Inches" {
		t.Errorf("Expected orgName to be '12 Inches', got %q", orgName3)
	}

	// Test multi-word positional org name with flags
	orgName4, slot4, _, _, err4 := parseOrgCommand("org 12 Inches --slot 3")
	if err4 != nil {
		t.Fatalf("Unexpected error parsing multi-word org with flags: %v", err4)
	}
	if orgName4 != "12 Inches" {
		t.Errorf("Expected orgName to be '12 Inches', got %q", orgName4)
	}
	if slot4 != 3 {
		t.Errorf("Expected slot to be 3, got %d", slot4)
	}

	// Test flags before multi-word positional org name
	orgName5, slot5, _, _, err5 := parseOrgCommand("org --slot 4 12 Inches")
	if err5 != nil {
		t.Fatalf("Unexpected error parsing flags before multi-word org: %v", err5)
	}
	if orgName5 != "12 Inches" {
		t.Errorf("Expected orgName to be '12 Inches', got %q", orgName5)
	}
	if slot5 != 4 {
		t.Errorf("Expected slot to be 4, got %d", slot5)
	}

	// Test invalid command prefix
	_, _, _, _, errInvalid := parseOrgCommand("invalidcommand MyOrg")
	if errInvalid == nil {
		t.Errorf("Expected error for non-org command prefix")
	}
}

func TestStateOrgViewTransition(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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

	m := InitialModel(mock, mock, mock)
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
	if !strings.Contains(viewContent, m.orgSpinner.View()) {
		t.Errorf("Expected viewport to initially contain spinner %q, got:\n%s", m.orgSpinner.View(), viewContent)
	}
	if strings.Contains(viewContent, "Loading...") {
		t.Errorf("Expected viewport not to contain placeholder 'Loading...', got:\n%s", viewContent)
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
	expectedLine := "[1] John Coltrane - Blue Train [ MainShelf / 1]"
	if !strings.Contains(viewContentUpdated, expectedLine) {
		t.Errorf("Expected viewport to contain %q, got:\n%s", expectedLine, viewContentUpdated)
	}
	if strings.Contains(viewContentUpdated, "Space:") || strings.Contains(viewContentUpdated, "Width:") || strings.Contains(viewContentUpdated, "Organization:") {
		t.Errorf("Expected viewport not to contain verbose labels, got:\n%s", viewContentUpdated)
	}

	fullView := m.View()
	if strings.Contains(fullView, "Organization View:") || strings.Contains(fullView, "Organization:") {
		t.Errorf("Expected full view not to contain redundant Organization headers, got:\n%s", fullView)
	}
	if !strings.Contains(fullView, "hash-123") {
		t.Errorf("Expected full view header to contain snapshot hash 'hash-123', got:\n%s", fullView)
	}
}

func TestOrgViewDisplaysSpinnerWhileLoadingRecords(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateOrgView
	m.orgPlacements = []*pb.Placement{
		{
			Iid:   301,
			Space: "Shelf A",
			Unit:  1,
			Index: 1,
		},
		{
			Iid:   302,
			Space: "Shelf A",
			Unit:  2,
			Index: 2,
		},
	}
	m.resolvedRecords = make(map[int64]*pb.Record)
	m.renderOrgViewport()

	initialView := m.orgViewport.View()
	if strings.Contains(initialView, "Loading...") {
		t.Errorf("Expected viewport not to contain 'Loading...', got:\n%s", initialView)
	}
	if !strings.Contains(initialView, m.orgSpinner.View()) {
		t.Errorf("Expected viewport to contain initial spinner frame %q, got:\n%s", m.orgSpinner.View(), initialView)
	}

	// Advance spinner by sending TickMsg
	tickMsg := m.orgSpinner.Tick()
	newModel, tickCmd := m.Update(tickMsg)
	m = newModel.(Model)

	tickedView := m.orgViewport.View()
	if !strings.Contains(tickedView, m.orgSpinner.View()) {
		t.Errorf("Expected viewport after tick to contain updated spinner frame %q, got:\n%s", m.orgSpinner.View(), tickedView)
	}
	if tickCmd == nil {
		t.Fatalf("Expected tickCmd to be returned while records are unresolved")
	}

	// Resolve one record
	newModel, _ = m.Update(recordFetchedMsg{
		iid: 301,
		record: &pb.Record{
			Release: &pbd.Release{Title: "Record One", Artists: []*pbd.Artist{{Name: "Artist One"}}},
		},
	})
	m = newModel.(Model)

	partialView := m.orgViewport.View()
	if !strings.Contains(partialView, "Artist One - Record One") {
		t.Errorf("Expected viewport to contain resolved record one, got:\n%s", partialView)
	}
	if !strings.Contains(partialView, m.orgSpinner.View()) {
		t.Errorf("Expected unresolved record two to still display spinner, got:\n%s", partialView)
	}

	// Resolve the second record
	newModel, _ = m.Update(recordFetchedMsg{
		iid: 302,
		record: &pb.Record{
			Release: &pbd.Release{Title: "Record Two", Artists: []*pbd.Artist{{Name: "Artist Two"}}},
		},
	})
	m = newModel.(Model)

	resolvedView := m.orgViewport.View()
	if !strings.Contains(resolvedView, "Artist One - Record One") {
		t.Errorf("Expected viewport to contain Artist One - Record One, got:\n%s", resolvedView)
	}
	if !strings.Contains(resolvedView, "Artist Two - Record Two") {
		t.Errorf("Expected viewport to contain Artist Two - Record Two, got:\n%s", resolvedView)
	}

	// Subsequent tick should stop returning commands since all records are resolved
	tickMsg = m.orgSpinner.Tick()
	newModel, finalCmd := m.Update(tickMsg)
	if finalCmd != nil {
		t.Errorf("Expected nil command after all records resolved, got %v", finalCmd)
	}
}

func TestOrgViewFormattingAndHashHeader(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
	m.state = StateOrgView
	m.activeHash = "my-test-org-hash-999"
	m.orgPlacements = []*pb.Placement{
		{
			Iid:   200,
			Space: "Main Shelves",
			Unit:  1,
			Index: 1,
		},
	}
	m.resolvedRecords = map[int64]*pb.Record{
		200: {
			Release: &pbd.Release{
				Id:      200,
				Title:   "Are You Serious",
				Artists: []*pbd.Artist{{Name: "Andrew Bird"}},
			},
		},
	}
	m.renderOrgViewport()

	expectedPlacement := "[1] Andrew Bird - Are You Serious [ Main Shelves / 1]"
	vpContent := m.orgViewport.View()
	if !strings.Contains(vpContent, expectedPlacement) {
		t.Errorf("Expected viewport content to contain %q, got:\n%s", expectedPlacement, vpContent)
	}
	if strings.Contains(vpContent, "Space:") || strings.Contains(vpContent, "Organization:") {
		t.Errorf("Expected viewport content not to contain 'Space:' or 'Organization:', got:\n%s", vpContent)
	}

	view := m.View()
	if !strings.Contains(view, "my-test-org-hash-999") {
		t.Errorf("Expected View() to contain the org hash in header, got:\n%s", view)
	}
	if strings.Contains(view, "Organization View:") || strings.Contains(view, "Organization:") {
		t.Errorf("Expected View() not to contain 'Organization View:' or 'Organization:', got:\n%s", view)
	}
}

func TestOrgViewNavigationAndExit(t *testing.T) {
	m := InitialModel(&mockClient{}, &mockClient{}, &mockClient{})
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

func TestStateLocateView_FieldsAndMsg(t *testing.T) {
	m := Model{
		state:            StateLocateView,
		locateViewport:   viewport.New(80, 20),
		activeLocateID:   12345,
		locateResponse:   &pb.LocateRecordResponse{},
	}

	if m.state != StateLocateView {
		t.Errorf("Expected state to be StateLocateView, got %v", m.state)
	}
	if m.activeLocateID != 12345 {
		t.Errorf("Expected activeLocateID to be 12345, got %d", m.activeLocateID)
	}

	msg := locateFetchedMsg{
		releaseID: 12345,
		response:  &pb.LocateRecordResponse{},
		err:       nil,
	}
	if msg.releaseID != 12345 {
		t.Errorf("Expected releaseID 12345, got %d", msg.releaseID)
	}
}

func TestParseLocateCommand_ZeroArgs(t *testing.T) {
	id, isSearch, err := parseLocateCommand("locate")
	if err != nil {
		t.Fatalf("Unexpected error for 'locate': %v", err)
	}
	if !isSearch {
		t.Errorf("Expected isSearch=true for 'locate', got false")
	}
	if id != 0 {
		t.Errorf("Expected id=0 for zero-argument locate, got %d", id)
	}

	// Whitespace variations
	id, isSearch, err = parseLocateCommand("   locate   ")
	if err != nil {
		t.Fatalf("Unexpected error for '   locate   ': %v", err)
	}
	if !isSearch || id != 0 {
		t.Errorf("Expected isSearch=true and id=0, got isSearch=%v, id=%d", isSearch, id)
	}
}

func TestParseLocateCommand_DirectID(t *testing.T) {
	tests := []struct {
		input      string
		expectedID int64
	}{
		{"locate 12345", 12345},
		{"locate --id 67890", 67890},
		{"locate --id=54321", 54321},
	}

	for _, tt := range tests {
		id, isSearch, err := parseLocateCommand(tt.input)
		if err != nil {
			t.Fatalf("Unexpected error for %q: %v", tt.input, err)
		}
		if isSearch {
			t.Errorf("Expected isSearch=false for %q, got true", tt.input)
		}
		if id != tt.expectedID {
			t.Errorf("Expected id=%d for %q, got %d", tt.expectedID, tt.input, id)
		}
	}
}

func TestParseLocateCommand_InvalidArgs(t *testing.T) {
	tests := []string{
		"locate invalid",
		"locate --id invalid",
		"locate --invalidflag",
		"locate 0",
		"locate -10",
		"locate --id 0",
		"locate --id -5",
		"locate 12345 67890",
		"",
		"othercommand",
	}

	for _, input := range tests {
		_, _, err := parseLocateCommand(input)
		if err == nil {
			t.Errorf("Expected error for input %q, got nil", input)
		}
	}
}

func TestFetchCollectionIndexCmd(t *testing.T) {
	var requestedWithGetAllRecords bool
	mock := &mockClient{
		getRecordFunc: func(req *pb.GetRecordRequest) (*pb.GetRecordResponse, error) {
			requestedWithGetAllRecords = req.GetGetAllRecords()
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{Record: &pb.Record{Release: &pbd.Release{Id: 101, Title: "Test Record 1"}}},
					{Record: &pb.Record{Release: &pbd.Release{Id: 102, Title: "Test Record 2"}}},
				},
			}, nil
		},
	}
	m := InitialModel(mock, mock, mock)
	cmd := m.fetchCollectionIndexCmd()
	if cmd == nil {
		t.Fatalf("Expected fetchCollectionIndexCmd to return a tea.Cmd")
	}

	msg := cmd()
	fetchedMsg, ok := msg.(collectionFetchedMsg)
	if !ok {
		t.Fatalf("Expected collectionFetchedMsg, got %T", msg)
	}
	if fetchedMsg.err != nil {
		t.Fatalf("Unexpected error in collectionFetchedMsg: %v", fetchedMsg.err)
	}
	if !requestedWithGetAllRecords {
		t.Errorf("Expected GetRecord to be called with GetAllRecords=true")
	}
	if len(fetchedMsg.records) != 2 {
		t.Errorf("Expected 2 records in collectionFetchedMsg, got %d", len(fetchedMsg.records))
	}

	// Test error branch
	mockErr := &mockClient{
		getRecordFunc: func(req *pb.GetRecordRequest) (*pb.GetRecordResponse, error) {
			return nil, fmt.Errorf("injected fetch error")
		},
	}
	mErr := InitialModel(mockErr, mockErr, mockErr)
	cmdErr := mErr.fetchCollectionIndexCmd()
	msgErr := cmdErr()
	fetchedMsgErr, ok := msgErr.(collectionFetchedMsg)
	if !ok {
		t.Fatalf("Expected collectionFetchedMsg, got %T", msgErr)
	}
	if fetchedMsgErr.err == nil || !strings.Contains(fetchedMsgErr.err.Error(), "injected fetch error") {
		t.Errorf("Expected injected fetch error, got %v", fetchedMsgErr.err)
	}
}

func TestHandleCommandInput_LocateSearch(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Zero-arg locate should transition to StateLocateSearch and dispatch fetchCollectionIndexCmd if collectionIndex is nil
	newModel, cmd := m.handleCommandInput("locate")
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}
	if updatedModel.state != StateLocateSearch {
		t.Errorf("Expected state to transition to StateLocateSearch, got %v", updatedModel.state)
	}
	if !updatedModel.collectionLoading {
		t.Errorf("Expected collectionLoading=true when collectionIndex is nil")
	}
	if cmd == nil {
		t.Errorf("Expected fetchCollectionIndexCmd to be returned")
	}
	if updatedModel.locateSearchCursor != 0 {
		t.Errorf("Expected locateSearchCursor to be reset to 0, got %d", updatedModel.locateSearchCursor)
	}

	// When collectionIndex is already cached, does not dispatch cmd and populates filteredLocateRecords
	cachedRecords := []*pb.Record{
		{Release: &pbd.Release{Id: 999, Title: "Cached Record"}},
	}
	updatedModel.collectionIndex = cachedRecords
	updatedModel.collectionLoading = false
	updatedModel.state = StateMainApp

	newModelCached, cmdCached := updatedModel.handleCommandInput("locate")
	cachedModel := newModelCached.(Model)
	if cachedModel.state != StateLocateSearch {
		t.Errorf("Expected state to be StateLocateSearch, got %v", cachedModel.state)
	}
	if cmdCached != nil {
		t.Errorf("Expected cmd to be nil when collectionIndex is already cached")
	}
	if len(cachedModel.filteredLocateRecords) != 1 {
		t.Errorf("Expected filteredLocateRecords to be populated from cache, got %d", len(cachedModel.filteredLocateRecords))
	}
}

func TestUpdate_CollectionFetchedMsg(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateLocateSearch
	m.collectionLoading = true

	records := []*pb.Record{
		{Release: &pbd.Release{Id: 201, Title: "Record A"}},
		{Release: &pbd.Release{Id: 202, Title: "Record B"}},
	}

	newModel, _ := m.Update(collectionFetchedMsg{records: records})
	updatedModel := newModel.(Model)

	if updatedModel.collectionLoading {
		t.Errorf("Expected collectionLoading=false after collectionFetchedMsg")
	}
	if len(updatedModel.collectionIndex) != 2 {
		t.Errorf("Expected collectionIndex to have 2 records, got %d", len(updatedModel.collectionIndex))
	}
	if len(updatedModel.filteredLocateRecords) != 2 {
		t.Errorf("Expected filteredLocateRecords to have 2 records, got %d", len(updatedModel.filteredLocateRecords))
	}
	if updatedModel.locateSearchErr != "" {
		t.Errorf("Expected locateSearchErr to be empty, got %s", updatedModel.locateSearchErr)
	}

	// Test error handling
	newModelErr, _ := m.Update(collectionFetchedMsg{err: fmt.Errorf("failed to fetch")})
	updatedModelErr := newModelErr.(Model)
	if updatedModelErr.collectionLoading {
		t.Errorf("Expected collectionLoading=false after error")
	}
	if updatedModelErr.locateSearchErr != "failed to fetch" {
		t.Errorf("Expected locateSearchErr='failed to fetch', got %s", updatedModelErr.locateSearchErr)
	}
}

func TestFetchLocateCmd(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	cmd := m.fetchLocateCmd(12345)
	if cmd == nil {
		t.Fatalf("Expected fetchLocateCmd to return a tea.Cmd")
	}

	msg := cmd()
	fetchedMsg, ok := msg.(locateFetchedMsg)
	if !ok {
		t.Fatalf("Expected locateFetchedMsg, got %T", msg)
	}
	if fetchedMsg.releaseID != 12345 {
		t.Errorf("Expected releaseID 12345, got %d", fetchedMsg.releaseID)
	}
}

func TestHandleCommandInput_Locate(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Valid locate command
	newModel, cmd := m.handleCommandInput("locate 12345")
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}

	if updatedModel.state != StateLocateView {
		t.Errorf("Expected state to transition to StateLocateView, got %v", updatedModel.state)
	}
	if updatedModel.activeLocateID != 12345 {
		t.Errorf("Expected activeLocateID to be 12345, got %d", updatedModel.activeLocateID)
	}
	if cmd == nil {
		t.Errorf("Expected fetchLocateCmd to be returned")
	}

	// Invalid locate command
	m2 := InitialModel(mock, mock, mock)
	m2.state = StateMainApp

	newModel2, cmd2 := m2.handleCommandInput("locate invalid")
	updatedModel2 := newModel2.(Model)

	if updatedModel2.inlineErrMsg != "Invalid release ID format. Usage: locate <release_id> or locate --id <release_id>" {
		t.Errorf("Expected inlineErrMsg to be set for invalid locate syntax, got %q", updatedModel2.inlineErrMsg)
	}
	if cmd2 != nil {
		t.Errorf("Expected cmd to be nil for invalid locate syntax")
	}
}

func TestFormatLocateOutput(t *testing.T) {
	resp := &pb.LocateRecordResponse{
		Locations: []*pb.Location{
			{
				LocationName: "Shelf A",
				Slot:         3,
				Record:       "Target Album",
				Before: []*pb.Context{
					{Record: "Before Album 1"},
					{Record: "Before Album 2"},
				},
				After: []*pb.Context{
					{Record: "After Album 1"},
				},
			},
		},
	}

	out := formatLocateOutput(resp)
	if !strings.Contains(out, "Target Album is in Shelf A, Slot 3 (75%):") && !strings.Contains(out, "Target Album is in Shelf A, Slot 3 (75 %):") {
		t.Errorf("Expected location header, got:\n%s", out)
	}
	if !strings.Contains(out, "Before Album 1") || !strings.Contains(out, "Before Album 2") {
		t.Errorf("Expected preceding records in output, got:\n%s", out)
	}
	if !strings.Contains(out, "After Album 1") {
		t.Errorf("Expected following records in output, got:\n%s", out)
	}

	// Empty locations edge case
	emptyResp := &pb.LocateRecordResponse{}
	emptyOut := formatLocateOutput(emptyResp)
	if !strings.Contains(emptyOut, "No location found for Release ID") && !strings.Contains(emptyOut, "No location found") {
		t.Errorf("Expected empty location error message, got %q", emptyOut)
	}
}

func TestStateLocateView_UpdateAndNavigation(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateLocateView
	m.activeLocateID = 999

	// Process locateFetchedMsg
	fetchedMsg := locateFetchedMsg{
		releaseID: 999,
		response: &pb.LocateRecordResponse{
			Locations: []*pb.Location{
				{
					LocationName: "Shelf 1",
					Slot:         1,
					Record:       "Record 999",
				},
			},
		},
	}
	newModel, _ := m.Update(fetchedMsg)
	mUpdated := newModel.(Model)

	if !strings.Contains(mUpdated.locateViewport.View(), "Record 999 is in Shelf 1") {
		t.Errorf("Expected viewport to contain formatted location output, got %q", mUpdated.locateViewport.View())
	}

	// Key bindings: esc or q to return to StateMainApp
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	mEsc, _ := mUpdated.Update(escMsg)
	if mEsc.(Model).state != StateMainApp {
		t.Errorf("Expected esc to transition to StateMainApp, got %v", mEsc.(Model).state)
	}

	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	mQ, _ := mUpdated.Update(qMsg)
	if mQ.(Model).state != StateMainApp {
		t.Errorf("Expected 'q' to transition to StateMainApp, got %v", mQ.(Model).state)
	}

	// Key bindings: viewport scrolling
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	mDown, _ := mUpdated.Update(downMsg)
	if mDown.(Model).state != StateLocateView {
		t.Errorf("Expected state to stay StateLocateView on scroll, got %v", mDown.(Model).state)
	}

	// Error handling: locateFetchedMsg with error
	errMsg := locateFetchedMsg{
		releaseID: 999,
		err:       fmt.Errorf("gRPC failure"),
	}
	mErrModel, _ := m.Update(errMsg)
	mErr := mErrModel.(Model)
	if mErr.inlineErrMsg != "gRPC failure" {
		t.Errorf("Expected inlineErrMsg to be 'gRPC failure', got %q", mErr.inlineErrMsg)
	}
	viewErr := mErr.View()
	if !strings.Contains(viewErr, "gRPC failure") {
		t.Errorf("Expected View() to contain error message, got %q", viewErr)
	}
}

func TestLocateCommand_Success(t *testing.T) {
	mock := &mockClient{
		locateRecordFunc: func(req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
			return &pb.LocateRecordResponse{
				Locations: []*pb.Location{
					{
						LocationName: "Box A",
						Slot:         12,
						Record:       "Kind of Blue",
					},
				},
			}, nil
		},
	}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	newModel, cmd := m.handleCommandInput("locate 12345")
	if cmd == nil {
		t.Fatalf("Expected non-nil cmd for valid locate command")
	}

	msg := cmd()
	fetchedMsg, ok := msg.(locateFetchedMsg)
	if !ok {
		t.Fatalf("Expected locateFetchedMsg, got %T", msg)
	}
	if fetchedMsg.err != nil {
		t.Fatalf("Unexpected error in locateFetchedMsg: %v", fetchedMsg.err)
	}

	m2, _ := newModel.(Model).Update(fetchedMsg)
	mUpdated := m2.(Model)
	if mUpdated.state != StateLocateView {
		t.Errorf("Expected state StateLocateView, got %v", mUpdated.state)
	}
	if !strings.Contains(mUpdated.locateViewport.View(), "Kind of Blue is in Box A") {
		t.Errorf("Expected viewport to contain 'Kind of Blue is in Box A', got:\n%s", mUpdated.locateViewport.View())
	}
}

func TestLocateCommand_NotFound(t *testing.T) {
	mock := &mockClient{
		locateRecordFunc: func(req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
			return &pb.LocateRecordResponse{
				Locations: []*pb.Location{},
			}, nil
		},
	}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	newModel, cmd := m.handleCommandInput("locate 99999")
	if cmd == nil {
		t.Fatalf("Expected non-nil cmd for valid locate command")
	}

	msg := cmd()
	fetchedMsg, ok := msg.(locateFetchedMsg)
	if !ok {
		t.Fatalf("Expected locateFetchedMsg, got %T", msg)
	}

	m2, _ := newModel.(Model).Update(fetchedMsg)
	mUpdated := m2.(Model)
	if !strings.Contains(mUpdated.locateViewport.View(), "No location found") {
		t.Errorf("Expected viewport to state no location found, got:\n%s", mUpdated.locateViewport.View())
	}
}

func TestLocateCommand_RPCError(t *testing.T) {
	mock := &mockClient{
		locateRecordFunc: func(req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
			return nil, fmt.Errorf("rpc error: code = Unavailable desc = transport is closing")
		},
	}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	newModel, cmd := m.handleCommandInput("locate 12345")
	if cmd == nil {
		t.Fatalf("Expected non-nil cmd for valid locate command")
	}

	msg := cmd()
	fetchedMsg, ok := msg.(locateFetchedMsg)
	if !ok {
		t.Fatalf("Expected locateFetchedMsg, got %T", msg)
	}
	if fetchedMsg.err == nil {
		t.Fatalf("Expected error in locateFetchedMsg, got nil")
	}

	m2, _ := newModel.(Model).Update(fetchedMsg)
	mUpdated := m2.(Model)
	if mUpdated.inlineErrMsg != "rpc error: code = Unavailable desc = transport is closing" {
		t.Errorf("Expected inlineErrMsg to be set on RPC error, got %q", mUpdated.inlineErrMsg)
	}
}

func TestStartup_ExistingToken(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.tokenLoader = func() (string, error) {
		return "test-stored-token", nil
	}

	// 1. Key press transition
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(keyMsg)
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}
	if updatedModel.state != StateLoadingSync {
		t.Errorf("Expected state to transition directly to StateLoadingSync, got %v", updatedModel.state)
	}
	if updatedModel.authToken != "test-stored-token" {
		t.Errorf("Expected authToken to be 'test-stored-token', got %v", updatedModel.authToken)
	}
	if cmd == nil {
		t.Errorf("Expected pollSync cmd to be returned")
	}

	// 2. Timeout transition
	timeoutMsgVal := timeoutMsg{}
	newModel2, cmd2 := m.Update(timeoutMsgVal)
	updatedModel2, ok := newModel2.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}
	if updatedModel2.state != StateLoadingSync {
		t.Errorf("Expected state to transition directly to StateLoadingSync on timeout, got %v", updatedModel2.state)
	}
	if updatedModel2.authToken != "test-stored-token" {
		t.Errorf("Expected authToken to be 'test-stored-token', got %v", updatedModel2.authToken)
	}
	if cmd2 == nil {
		t.Errorf("Expected pollSync cmd to be returned")
	}
}

func TestStartup_NoToken(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.tokenLoader = func() (string, error) {
		return "", fmt.Errorf("file not found")
	}

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(keyMsg)
	updatedModel := newModel.(Model)

	if updatedModel.state != StateLogin {
		t.Errorf("Expected state to transition to StateLogin when no token exists, got %v", updatedModel.state)
	}
	if updatedModel.authToken != "" {
		t.Errorf("Expected authToken to be empty, got %v", updatedModel.authToken)
	}
	if cmd == nil {
		t.Errorf("Expected fetchURL cmd to be returned")
	}
}

func TestDefaultTokenLoader_SuccessAndFailure(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Failure when file does not exist
	_, err := defaultTokenLoader()
	if err == nil {
		t.Errorf("Expected error when ~/.gramophile does not exist, got nil")
	}

	// Failure when token is empty
	err = defaultTokenSaver("")
	if err != nil {
		t.Fatalf("Unexpected error saving empty token: %v", err)
	}
	_, err = defaultTokenLoader()
	if err == nil {
		t.Errorf("Expected error when token is empty, got nil")
	}

	// Success when valid token is saved
	err = defaultTokenSaver("valid-auth-token-12345")
	if err != nil {
		t.Fatalf("Unexpected error saving token: %v", err)
	}

	token, err := defaultTokenLoader()
	if err != nil {
		t.Fatalf("Unexpected error loading token: %v", err)
	}
	if token != "valid-auth-token-12345" {
		t.Errorf("Expected token 'valid-auth-token-12345', got %q", token)
	}
}

func TestContextAuthTokenPropagation(t *testing.T) {
	var capturedUserMD metadata.MD

	mock := &mockClient{
		getUserCtxFunc: func(ctx context.Context) (*pb.GetUserResponse, error) {
			capturedUserMD, _ = metadata.FromOutgoingContext(ctx)
			return &pb.GetUserResponse{User: &pb.StoredUser{}}, nil
		},
	}

	m := InitialModel(mock, mock, mock)
	m.authToken = "secret-token-xyz"

	// pollSync with custom client to inspect metadata
	pollSyncCmd := m.pollSync()
	pollSyncCmd()
	if vals := capturedUserMD.Get("auth-token"); len(vals) == 0 || vals[0] != "secret-token-xyz" {
		t.Errorf("Expected auth-token metadata in pollSync, got %v", vals)
	}

	// buildContext directly
	ctx, cancel := m.buildContext(10)
	defer cancel()
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok || len(md.Get("auth-token")) == 0 || md.Get("auth-token")[0] != "secret-token-xyz" {
		t.Errorf("Expected buildContext to attach auth-token metadata, got %v", md)
	}
}

func TestStateMainApp_View_ContainsInputBarAndCommands(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Initially, commands list is hidden and footer shows "press h for help"
	view := m.View()
	if strings.Contains(view, "Handoff to main application complete") {
		t.Errorf("Expected view not to contain handoff message, got:\n%s", view)
	}
	if strings.Contains(view, "Commands:\n") || strings.Contains(view, "Locate a record in your organization") {
		t.Errorf("Expected commands list to be hidden initially, got:\n%s", view)
	}
	if !strings.Contains(view, "press h for help") {
		t.Errorf("Expected view to prompt 'press h for help', got:\n%s", view)
	}
	if !strings.Contains(view, "Command: ") {
		t.Errorf("Expected view to contain 'Command: ' input prompt, got:\n%s", view)
	}

	// Press 'h' to toggle commands visible
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(Model)

	viewHelp := m.View()
	if !strings.Contains(viewHelp, "Commands:\n") || !strings.Contains(viewHelp, "Locate a record in your organization") {
		t.Errorf("Expected view to list supported commands after pressing 'h', got:\n%s", viewHelp)
	}
	if !strings.Contains(viewHelp, "locate <release_id>") || !strings.Contains(viewHelp, "org [name]") || !strings.Contains(viewHelp, "configure") {
		t.Errorf("Expected view to list supported commands (locate, org, configure), got:\n%s", viewHelp)
	}
	if strings.Contains(viewHelp, "  o   ") {
		t.Errorf("Expected view to not list 'o' as command, got:\n%s", viewHelp)
	}
	if !strings.Contains(viewHelp, "press h to hide help") {
		t.Errorf("Expected view to prompt 'press h to hide help', got:\n%s", viewHelp)
	}

	// Press 'h' again to toggle commands hidden
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(Model)

	viewHidden := m.View()
	if strings.Contains(viewHidden, "Commands:\n") || strings.Contains(viewHidden, "Locate a record in your organization") {
		t.Errorf("Expected commands list to be hidden after pressing 'h' again, got:\n%s", viewHidden)
	}
	if !strings.Contains(viewHidden, "press h for help") {
		t.Errorf("Expected view to prompt 'press h for help', got:\n%s", viewHidden)
	}
}

func TestStateMainApp_ExecuteLocate(t *testing.T) {
	mock := &mockClient{
		locateRecordFunc: func(req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
			return &pb.LocateRecordResponse{}, nil
		},
	}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Type "locate 12345"
	for _, r := range "locate 12345" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	if m.textInput.Value() != "locate 12345" {
		t.Fatalf("Expected textInput value 'locate 12345', got %q", m.textInput.Value())
	}

	// Press Enter
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateLocateView {
		t.Errorf("Expected state to transition to StateLocateView, got %v", m.state)
	}
	if m.activeLocateID != 12345 {
		t.Errorf("Expected activeLocateID 12345, got %d", m.activeLocateID)
	}
	if cmd == nil {
		t.Errorf("Expected cmd to fetch locate record")
	}
}

func TestStateMainApp_ExecuteOrg(t *testing.T) {
	mock := &mockClient{
		getOrgFunc: func(req *pb.GetOrgRequest) (*pb.GetOrgResponse, error) {
			return &pb.GetOrgResponse{}, nil
		},
	}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Type "org --org MyShelf --slot 2"
	for _, r := range "org --org MyShelf --slot 2" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	// Press Enter
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateOrgView {
		t.Errorf("Expected state to transition to StateOrgView, got %v", m.state)
	}
	if m.activeOrgName != "MyShelf" {
		t.Errorf("Expected activeOrgName 'MyShelf', got %q", m.activeOrgName)
	}
	if m.activeSlot != 2 {
		t.Errorf("Expected activeSlot 2, got %d", m.activeSlot)
	}
	if cmd == nil {
		t.Errorf("Expected cmd to fetch org")
	}

	// Test multi-word org command: "org 12 Inches"
	m2 := InitialModel(mock, mock, mock)
	m2.state = StateMainApp
	for _, r := range "org 12 Inches" {
		newModel, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m2 = newModel.(Model)
	}
	newModel2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 = newModel2.(Model)
	if m2.state != StateOrgView {
		t.Errorf("Expected state to transition to StateOrgView, got %v", m2.state)
	}
	if m2.activeOrgName != "12 Inches" {
		t.Errorf("Expected activeOrgName '12 Inches', got %q", m2.activeOrgName)
	}
	if cmd2 == nil {
		t.Errorf("Expected cmd to fetch org")
	}

	// Test "org" with configured organization defaults to first org
	m3 := InitialModel(mock, mock, mock)
	m3.state = StateMainApp
	m3.user = &pb.StoredUser{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{Name: "Default Shelf"},
				},
			},
		},
	}
	for _, r := range "org" {
		newModel, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m3 = newModel.(Model)
	}
	newModel3, cmd3 := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 = newModel3.(Model)
	if m3.state != StateOrgView {
		t.Errorf("Expected state to transition to StateOrgView, got %v", m3.state)
	}
	if m3.activeOrgName != "Default Shelf" {
		t.Errorf("Expected activeOrgName 'Default Shelf', got %q", m3.activeOrgName)
	}
	if cmd3 == nil {
		t.Errorf("Expected cmd to fetch org")
	}

	// Test "org" without configured organization shows inline error
	m4 := InitialModel(mock, mock, mock)
	m4.state = StateMainApp
	for _, r := range "org" {
		newModel, _ := m4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m4 = newModel.(Model)
	}
	newModel4, cmd4 := m4.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 = newModel4.(Model)
	if m4.state != StateMainApp {
		t.Errorf("Expected state to remain StateMainApp when no org configured, got %v", m4.state)
	}
	if m4.inlineErrMsg != "No organization specified. Usage: org [name]" {
		t.Errorf("Expected inline error message for missing org, got %q", m4.inlineErrMsg)
	}
	if cmd4 != nil {
		t.Errorf("Expected cmd to be nil when org validation fails")
	}
}

func TestStateMainApp_ExecuteConfigure_TransitionsToConfigSelect(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Type "configure" and press enter
	for _, r := range "configure" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateConfigSelect {
		t.Fatalf("Expected state to transition to StateConfigSelect on 'configure' command, got %v", m.state)
	}
	if m.form == nil {
		t.Fatalf("Expected config selection form to be initialized")
	}

	view := m.View()
	if !strings.Contains(view, "Configuration Target") || !strings.Contains(view, "org") {
		t.Errorf("Expected StateConfigSelect view to show configuration selection, got:\n%s", view)
	}

	// Completing selection with "org" transitions to StateOrgConfig
	m.configTarget = "org"
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)

	if m.state != StateOrgConfig {
		t.Fatalf("Expected state to transition to StateOrgConfig after selecting 'org', got %v", m.state)
	}
	if m.form == nil {
		t.Errorf("Expected org config form to be initialized")
	}
}

func TestStateMainApp_ExecuteConfigure_AbortReturnsToMainApp(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	for _, r := range "configure" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateConfigSelect {
		t.Fatalf("Expected StateConfigSelect, got %v", m.state)
	}

	// Pressing Esc aborts back to StateMainApp
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	if m.state != StateMainApp {
		t.Errorf("Expected state to return to StateMainApp on Esc, got %v", m.state)
	}
}

func TestStateMainApp_ExecuteConfigureOrg_DirectTransition(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Type "configure org" and press enter
	for _, r := range "configure org" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateOrgConfig {
		t.Errorf("Expected state to transition directly to StateOrgConfig on 'configure org', got %v", m.state)
	}
	if m.form == nil {
		t.Errorf("Expected org config form to be initialized")
	}

	// Test "config org" command
	m2 := InitialModel(mock, mock, mock)
	m2.state = StateMainApp
	for _, r := range "config org" {
		newModel, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m2 = newModel.(Model)
	}
	newModel2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 = newModel2.(Model)
	if m2.state != StateOrgConfig {
		t.Errorf("Expected state to transition to StateOrgConfig on 'config org', got %v", m2.state)
	}

	// Test "config" command transitions to StateConfigSelect
	m3 := InitialModel(mock, mock, mock)
	m3.state = StateMainApp
	for _, r := range "config" {
		newModel, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m3 = newModel.(Model)
	}
	newModel3, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 = newModel3.(Model)
	if m3.state != StateConfigSelect {
		t.Errorf("Expected state to transition to StateConfigSelect on 'config', got %v", m3.state)
	}
}

func TestStateMainApp_ExecuteConfigure_InvalidTarget(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	for _, r := range "configure unknown" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateMainApp {
		t.Errorf("Expected state to remain StateMainApp on invalid configure target, got %v", m.state)
	}
	if !strings.Contains(m.inlineErrMsg, "Unknown configuration target: unknown") {
		t.Errorf("Expected inline error for unknown configuration target, got %q", m.inlineErrMsg)
	}
}

func TestStateMainApp_ExecuteO_NoLongerConfigures(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Typing "o" should no longer configure org
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = newModel.(Model)

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state == StateOrgConfig || m.state == StateConfigSelect {
		t.Errorf("Expected state NOT to transition to OrgConfig/ConfigSelect on 'o', got %v", m.state)
	}
	if m.inlineErrMsg != "unknown command: o" {
		t.Errorf("Expected inlineErrMsg 'unknown command: o', got %q", m.inlineErrMsg)
	}
}

func TestStateMainApp_ExecuteQuit(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	for _, r := range "quit" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected quit command to return tea.Quit cmd")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Expected tea.QuitMsg, got %T", msg)
	}
}

func TestStateMainApp_InvalidCommand_ShowsError(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	for _, r := range "unknown_command 123" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateMainApp {
		t.Errorf("Expected state to remain StateMainApp on invalid command, got %v", m.state)
	}
	if cmd != nil {
		t.Errorf("Expected no command to be returned on invalid input")
	}
	if m.inlineErrMsg == "" {
		t.Errorf("Expected inlineErrMsg to be set on invalid command")
	}
	view := m.View()
	if !strings.Contains(view, m.inlineErrMsg) {
		t.Errorf("Expected view to render inline error message %q, got:\n%s", m.inlineErrMsg, view)
	}

	// Test Esc clears inline error and text input
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.inlineErrMsg != "" {
		t.Errorf("Expected Esc to clear inlineErrMsg, got %q", m.inlineErrMsg)
	}
	if m.textInput.Value() != "" {
		t.Errorf("Expected Esc to clear textInput value, got %q", m.textInput.Value())
	}
}

func TestStateMainApp_TypingQ_DoesNotQuit(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Type 'q'
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(Model)

	if cmd != nil {
		msg := cmd()
		if _, isQuit := msg.(tea.QuitMsg); isQuit {
			t.Fatalf("Typing 'q' in StateMainApp should not immediately quit the application")
		}
	}
	if m.textInput.Value() != "q" {
		t.Errorf("Expected textInput value 'q', got %q", m.textInput.Value())
	}
}

func TestStateMainApp_TypingHWithText_DoesNotToggleHelp(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Type "locate 12"
	for _, r := range "locate 12" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	// Type 'h'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(Model)

	if m.textInput.Value() != "locate 12h" {
		t.Errorf("Expected textInput value 'locate 12h', got %q", m.textInput.Value())
	}
	if m.showHelp {
		t.Errorf("Expected showHelp to remain false when typing 'h' in non-empty input")
	}
}

func TestStateMainApp_EscDismissesHelp(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Toggle help on with 'h'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(Model)
	if !m.showHelp {
		t.Fatalf("Expected showHelp to be true after pressing 'h'")
	}

	// Press Esc
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)
	if m.showHelp {
		t.Errorf("Expected Esc to dismiss help (showHelp=false)")
	}
}

func TestStateMainApp_HelpCommand_TogglesHelp(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateMainApp

	// Type "help" and press Enter
	for _, r := range "help" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if !m.showHelp {
		t.Errorf("Expected 'help' command to enable showHelp")
	}

	// Type "h" and press Enter to toggle off
	for _, r := range "h" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.showHelp {
		t.Errorf("Expected 'h' command to toggle showHelp off")
	}
}

func TestLocateSearch_Filtering(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateLocateSearch
	m.collectionIndex = []*pb.Record{
		{Release: &pbd.Release{Id: 101, Title: "The Wall", Artists: []*pbd.Artist{{Name: "Pink Floyd"}}}},
		{Release: &pbd.Release{Id: 102, Title: "Kind of Blue", Artists: []*pbd.Artist{{Name: "Miles Davis"}}}},
		{Release: &pbd.Release{Id: 103, Title: "Blue Train", Artists: []*pbd.Artist{{Name: "John Coltrane"}}}},
	}
	m.filteredLocateRecords = m.collectionIndex

	// Type "mIlEs" (case-insensitive substring match on artist)
	for _, r := range "mIlEs" {
		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newM.(Model)
	}
	if len(m.filteredLocateRecords) != 1 {
		t.Fatalf("Expected 1 match for 'mIlEs', got %d", len(m.filteredLocateRecords))
	}
	if m.filteredLocateRecords[0].GetRelease().GetId() != 102 {
		t.Errorf("Expected release ID 102, got %d", m.filteredLocateRecords[0].GetRelease().GetId())
	}

	// Change search cursor to non-zero, then change query to verify cursor resets
	m.locateSearchCursor = 1

	// Clear and search for "blue" (case-insensitive substring match on title)
	m.locateSearchInput.SetValue("")
	for _, r := range "blue" {
		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newM.(Model)
	}
	if len(m.filteredLocateRecords) != 2 {
		t.Fatalf("Expected 2 matches for 'blue', got %d", len(m.filteredLocateRecords))
	}
	if m.locateSearchCursor != 0 {
		t.Errorf("Expected locateSearchCursor to be reset to 0, got %d", m.locateSearchCursor)
	}

	// Non-matching query
	for _, r := range "xyz123" {
		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newM.(Model)
	}
	if len(m.filteredLocateRecords) != 0 {
		t.Errorf("Expected 0 matches for non-matching query, got %d", len(m.filteredLocateRecords))
	}
}

func TestLocateSearch_Disambiguation(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateLocateSearch
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Id: 12, Name: "Listening Pile"},
		},
	}
	// Two records with identical artist and title (duplicate pressings) + one unique record
	rec1 := &pb.Record{
		Release: &pbd.Release{
			Id:       102,
			Title:    "Kind of Blue",
			Artists:  []*pbd.Artist{{Name: "Miles Davis"}},
			FolderId: 12,
		},
	}
	rec2 := &pb.Record{
		Release: &pbd.Release{
			Id:       104,
			Title:    "Kind of Blue",
			Artists:  []*pbd.Artist{{Name: "Miles Davis"}},
			FolderId: 99, // not in user.Folders -> fallback to GoalFolder
		},
		GoalFolder: "Archive Box",
	}
	rec3 := &pb.Record{
		Release: &pbd.Release{
			Id:       103,
			Title:    "Blue Train",
			Artists:  []*pbd.Artist{{Name: "John Coltrane"}},
			FolderId: 12,
		},
	}
	m.collectionIndex = []*pb.Record{rec1, rec2, rec3}
	m.filteredLocateRecords = m.collectionIndex

	view := m.View()

	// rec1 has folder resolved from user.Folders
	expectedRec1 := "Miles Davis - Kind of Blue [ID: 102 | Location: Listening Pile]"
	if !strings.Contains(view, expectedRec1) {
		t.Errorf("Expected view to contain disambiguated record 1 %q, got:\n%s", expectedRec1, view)
	}

	// rec2 has folder resolved from GoalFolder
	expectedRec2 := "Miles Davis - Kind of Blue [ID: 104 | Location: Archive Box]"
	if !strings.Contains(view, expectedRec2) {
		t.Errorf("Expected view to contain disambiguated record 2 %q, got:\n%s", expectedRec2, view)
	}

	// rec3 is unique, so should render standard <Artist> - <Title> without ID or Location
	if !strings.Contains(view, "John Coltrane - Blue Train") {
		t.Errorf("Expected view to contain standard format for unique record %q, got:\n%s", "John Coltrane - Blue Train", view)
	}
	if strings.Contains(view, "John Coltrane - Blue Train [ID:") {
		t.Errorf("Expected unique record not to contain disambiguation metadata, got:\n%s", view)
	}
}

func TestLocateSearch_NavigationAndSelection(t *testing.T) {
	mock := &mockClient{
		locateRecordFunc: func(req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
			return &pb.LocateRecordResponse{}, nil
		},
	}
	m := InitialModel(mock, mock, mock)
	m.state = StateLocateSearch
	m.filteredLocateRecords = []*pb.Record{
		{Release: &pbd.Release{Id: 101, Title: "Record 1"}},
		{Release: &pbd.Release{Id: 102, Title: "Record 2"}},
		{Release: &pbd.Release{Id: 103, Title: "Record 3"}},
	}
	m.locateSearchCursor = 0

	// Test arrow down navigation
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)
	if m.locateSearchCursor != 1 {
		t.Errorf("Expected cursor 1 after down arrow, got %d", m.locateSearchCursor)
	}

	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)
	if m.locateSearchCursor != 2 {
		t.Errorf("Expected cursor 2 after down arrow, got %d", m.locateSearchCursor)
	}

	// Down arrow clamped at end
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)
	if m.locateSearchCursor != 2 {
		t.Errorf("Expected cursor clamped to 2, got %d", m.locateSearchCursor)
	}

	// Test up arrow navigation
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newM.(Model)
	if m.locateSearchCursor != 1 {
		t.Errorf("Expected cursor 1 after up arrow, got %d", m.locateSearchCursor)
	}

	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newM.(Model)
	if m.locateSearchCursor != 0 {
		t.Errorf("Expected cursor 0 after up arrow, got %d", m.locateSearchCursor)
	}

	// Up arrow clamped at 0
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newM.(Model)
	if m.locateSearchCursor != 0 {
		t.Errorf("Expected cursor clamped to 0, got %d", m.locateSearchCursor)
	}

	// Also test j / k navigation
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newM.(Model)
	if m.locateSearchCursor != 1 {
		t.Errorf("Expected cursor 1 after 'j', got %d", m.locateSearchCursor)
	}

	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newM.(Model)
	if m.locateSearchCursor != 0 {
		t.Errorf("Expected cursor 0 after 'k', got %d", m.locateSearchCursor)
	}

	// Move cursor to 1 (Record 2, ID 102) and press Enter to select
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)
	if m.locateSearchCursor != 1 {
		t.Fatalf("Expected cursor at 1 before Enter, got %d", m.locateSearchCursor)
	}

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.state != StateLocateView {
		t.Errorf("Expected state StateLocateView after Enter, got %v", m.state)
	}
	if m.activeLocateID != 102 {
		t.Errorf("Expected activeLocateID 102, got %d", m.activeLocateID)
	}
	if cmd == nil {
		t.Fatalf("Expected cmd to fetch locate record on Enter")
	}

	msg := cmd()
	fetchedMsg, ok := msg.(locateFetchedMsg)
	if !ok {
		t.Fatalf("Expected locateFetchedMsg, got %T", msg)
	}
	if fetchedMsg.releaseID != 102 {
		t.Errorf("Expected fetchedMsg.releaseID 102, got %d", fetchedMsg.releaseID)
	}
}

func TestLocateSearch_EmptySelectionDisabled(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateLocateSearch
	m.filteredLocateRecords = []*pb.Record{}

	// When filtered records are empty, pressing Enter does nothing (returns m, nil)
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.state != StateLocateSearch {
		t.Errorf("Expected state to remain StateLocateSearch, got %v", m.state)
	}
	if cmd != nil {
		t.Errorf("Expected nil cmd when selecting from empty results")
	}

	// Verify View() displays "No matching records found."
	view := m.View()
	if !strings.Contains(view, "No matching records found.") {
		t.Errorf("Expected view to contain 'No matching records found.', got:\n%s", view)
	}
	// Verify footer
	if !strings.Contains(view, "↑/↓: Navigate • Enter: Locate • Esc: Cancel") {
		t.Errorf("Expected view to contain footer '↑/↓: Navigate • Enter: Locate • Esc: Cancel', got:\n%s", view)
	}
}

func TestLocateSearch_Cancellation(t *testing.T) {
	mock := &mockClient{}
	m := InitialModel(mock, mock, mock)
	m.state = StateLocateSearch
	m.locateSearchInput.SetValue("some query")
	m.locateSearchCursor = 3

	// Press Esc
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newM.(Model)

	if m.state != StateMainApp {
		t.Errorf("Expected state to transition to StateMainApp on Esc, got %v", m.state)
	}
	if m.locateSearchInput.Value() != "" {
		t.Errorf("Expected search input to be cleared, got %q", m.locateSearchInput.Value())
	}
	if m.locateSearchCursor != 0 {
		t.Errorf("Expected locateSearchCursor to be reset to 0, got %d", m.locateSearchCursor)
	}
	if cmd != nil {
		t.Errorf("Expected nil cmd on cancel, got %v", cmd)
	}
}



