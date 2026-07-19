package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type mockOrgClient struct {
	mockClient
	setConfigFunc func(*pb.SetConfigRequest) (*pb.SetConfigResponse, error)
}

func (m *mockOrgClient) SetConfig(ctx context.Context, in *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	if m.setConfigFunc != nil {
		return m.setConfigFunc(in)
	}
	return &pb.SetConfigResponse{}, nil
}

func TestTransitionToOrgConfig(t *testing.T) {
	m := InitialModel(&mockOrgClient{})
	m.state = StateMainApp
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "Inbox", Id: 1},
			{Name: "Uncategorized", Id: 2},
		},
		Config: &pb.GramophileConfig{},
	}

	// Pressing 'o' in StateMainApp should transition to StateOrgConfig
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)

	if updatedModel.state != StateOrgConfig {
		t.Errorf("Expected state to be StateOrgConfig, got %v", updatedModel.state)
	}

	// The form should be initialized
	if updatedModel.form == nil {
		t.Errorf("Expected form to be initialized")
	}

	// Make sure command is nil (no background tasks triggered yet)
	if cmd != nil {
		t.Errorf("Expected cmd to be nil, got %v", cmd)
	}
}

func TestOrgConfigSubmissionSuccess(t *testing.T) {
	var calledConfig *pb.GramophileConfig
	client := &mockOrgClient{
		setConfigFunc: func(req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
			calledConfig = req.GetConfig()
			return &pb.SetConfigResponse{}, nil
		},
	}

	m := InitialModel(client)
	m.state = StateOrgConfig
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "Inbox", Id: 123},
		},
		Config: &pb.GramophileConfig{},
	}

	// Initialize the form
	m.initOrgConfigForm()

	// Fill the variables bound to the form fields
	m.orgName = "My Test Org"
	m.spaceName = "Main Shelf"
	m.spaceUnits = "2"
	m.spaceWidth = "12.5"
	m.selectedFolders = []string{"123"}
	m.sortStrategy = "RELEASE_YEAR"

	// Simulate form completion by manually setting form state to StateCompleted
	m.form.State = huh.StateCompleted

	// Send a dummy update to trigger the StateCompleted check
	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updatedModel := newModel.(Model)

	// Since SetConfig is triggered, a command should be returned
	if cmd == nil {
		t.Fatalf("Expected a command to be returned to call SetConfig")
	}

	// Run the command to get the msg
	msg := cmd()
	newModel, cmd = updatedModel.Update(msg)
	updatedModel = newModel.(Model)

	// Should transition back to StateMainApp on success
	if updatedModel.state != StateMainApp {
		t.Errorf("Expected transition to StateMainApp on success, got %v", updatedModel.state)
	}

	// Check if config was mapped and passed to SetConfig correctly
	if calledConfig == nil {
		t.Fatalf("SetConfig was not called with config")
	}

	orgs := calledConfig.GetOrganisationConfig().GetOrganisations()
	if len(orgs) != 1 {
		t.Fatalf("Expected 1 organisation, got %d", len(orgs))
	}

	org := orgs[0]
	if org.GetName() != "My Test Org" {
		t.Errorf("Expected org name to be 'My Test Org', got %s", org.GetName())
	}

	if len(org.GetSpaces()) != 1 {
		t.Fatalf("Expected 1 space, got %d", len(org.GetSpaces()))
	}
	space := org.GetSpaces()[0]
	if space.GetName() != "Main Shelf" || space.GetUnits() != 2 || space.GetWidth() != 12.5 {
		t.Errorf("Space was mapped incorrectly: %+v", space)
	}

	if len(org.GetFoldersets()) != 1 {
		t.Fatalf("Expected 1 folderset, got %d", len(org.GetFoldersets()))
	}
	fs := org.GetFoldersets()[0]
	if fs.GetName() != "Inbox" || fs.GetFolder() != 123 || fs.GetSort() != pb.Sort_RELEASE_YEAR {
		t.Errorf("Folderset was mapped incorrectly: %+v", fs)
	}
}

func TestOrgConfigSubmissionError(t *testing.T) {
	client := &mockOrgClient{
		setConfigFunc: func(req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
			return nil, fmt.Errorf("duplicate organization name")
		},
	}

	m := InitialModel(client)
	m.state = StateOrgConfig
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "Inbox", Id: 123},
		},
		Config: &pb.GramophileConfig{},
	}

	// Initialize the form and fill details
	m.initOrgConfigForm()
	m.orgName = "Duplicate Org"
	m.spaceName = "Main Shelf"
	m.spaceUnits = "2"
	m.spaceWidth = "12.5"
	m.selectedFolders = []string{"123"}
	m.sortStrategy = "RELEASE_YEAR"

	m.form.State = huh.StateCompleted

	newModel, cmd := m.Update(tea.WindowSizeMsg{})
	updatedModel := newModel.(Model)

	if cmd == nil {
		t.Fatalf("Expected cmd to poll SetConfig")
	}

	// Run the cmd to produce the error
	msg := cmd()
	newModel, cmd = updatedModel.Update(msg)
	updatedModel = newModel.(Model)

	// Since error was returned, we should display it
	if updatedModel.err == nil || updatedModel.err.Error() != "duplicate organization name" {
		t.Errorf("Expected error to be stored in model, got: %v", updatedModel.err)
	}

	// State should remain StateOrgConfig or handle error screen
	if updatedModel.state != StateOrgConfig {
		t.Errorf("Expected state to remain StateOrgConfig, got %v", updatedModel.state)
	}

	// Verify the view shows the error
	view := updatedModel.View()
	if !strings.Contains(view, "duplicate organization name") {
		t.Errorf("Expected view to contain the error message, got:\n%s", view)
	}

	// Pressing any key should clear error and transition back to StateMainApp
	newModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updatedModel = newModel.(Model)

	if updatedModel.state != StateMainApp {
		t.Errorf("Expected transition to StateMainApp on key press, got %v", updatedModel.state)
	}
	if updatedModel.err != nil {
		t.Errorf("Expected error to be cleared, got %v", updatedModel.err)
	}
}
