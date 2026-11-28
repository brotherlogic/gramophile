package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockOrgClient struct {
	mockClient
	setConfigFunc func(*pb.SetConfigRequest) (*pb.SetConfigResponse, error)
}

func (m *mockOrgClient) SetConfig(ctx context.Context, in *pb.SetConfigRequest, opts ...grpc.CallOption) (*pb.SetConfigResponse, error) {
	if m.setConfigFunc != nil {
		return m.setConfigFunc(in)
	}
	return &pb.SetConfigResponse{}, nil
}

func TestTransitionToOrgConfig(t *testing.T) {
	m := InitialModel(&mockOrgClient{}, &mockOrgClient{}, &mockOrgClient{})
	m.state = StateMainApp
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "Inbox", Id: 1},
			{Name: "Uncategorized", Id: 2},
		},
		Config: &pb.GramophileConfig{},
	}

	// Entering 'configure org' command in StateMainApp should transition to StateOrgConfig
	for _, r := range "configure org" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	m := InitialModel(client, client, client)
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

	m := InitialModel(client, client, client)
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

	// Since error was returned, we should display it inline
	if updatedModel.orgErr == "" && updatedModel.err == nil {
		t.Errorf("Expected error to be stored in model")
	}

	// State should remain StateOrgConfig and form should remain active
	if updatedModel.state != StateOrgConfig {
		t.Errorf("Expected state to remain StateOrgConfig, got %v", updatedModel.state)
	}
	if updatedModel.form == nil {
		t.Errorf("Expected form to remain active for inline error reporting")
	}

	// Verify the view shows the error inline
	view := updatedModel.View()
	if !strings.Contains(view, "duplicate organization name") {
		t.Errorf("Expected view to contain the error message, got:\n%s", view)
	}
}

func TestConfigSelectListsExistingOrganisationsAndPrePopulatesForm(t *testing.T) {
	client := &mockOrgClient{}
	m := InitialModel(client, client, client)
	m.state = StateMainApp
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "12 Inches", Id: 10},
			{Name: "7 Inches", Id: 20},
			{Name: "Box Sets", Id: 30},
		},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "Main Kallax",
						Spaces: []*pb.Space{
							{Name: "Expedit", Units: 8, Width: 13.25},
						},
						Foldersets: []*pb.FolderSet{
							{Name: "12 Inches", Folder: 10, Sort: pb.Sort_ARTIST_YEAR},
							{Name: "Box Sets", Folder: 30, Sort: pb.Sort_ARTIST_YEAR},
						},
					},
					{
						Name: "Singles Box",
						Spaces: []*pb.Space{
							{Name: "Crate", Units: 2, Width: 7.5},
						},
						Foldersets: []*pb.FolderSet{
							{Name: "7 Inches", Folder: 20, Sort: pb.Sort_LABEL_CATNO},
						},
					},
				},
			},
		},
	}

	// Execute "configure" command to enter StateConfigSelect
	for _, r := range "configure" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	if m.state != StateConfigSelect {
		t.Fatalf("Expected StateConfigSelect, got %v", m.state)
	}

	view := m.View()
	if !strings.Contains(view, "Main Kallax") || !strings.Contains(view, "Singles Box") {
		t.Errorf("Expected StateConfigSelect view to list existing organisations 'Main Kallax' and 'Singles Box', got:\n%s", view)
	}

	// Select "Main Kallax" to edit
	m.configTarget = "Main Kallax"
	m.form.State = huh.StateCompleted
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)

	if m.state != StateOrgConfig {
		t.Fatalf("Expected transition to StateOrgConfig when selecting existing org, got %v", m.state)
	}

	if m.orgName != "Main Kallax" {
		t.Errorf("Expected pre-populated orgName 'Main Kallax', got %q", m.orgName)
	}
	if m.spaceName != "Expedit" {
		t.Errorf("Expected pre-populated spaceName 'Expedit', got %q", m.spaceName)
	}
	if m.spaceUnits != "8" {
		t.Errorf("Expected pre-populated spaceUnits '8', got %q", m.spaceUnits)
	}
	if m.spaceWidth != "13.25" {
		t.Errorf("Expected pre-populated spaceWidth '13.25', got %q", m.spaceWidth)
	}
	if len(m.selectedFolders) != 2 || m.selectedFolders[0] != "10" || m.selectedFolders[1] != "30" {
		t.Errorf("Expected pre-populated selectedFolders ['10', '30'], got %v", m.selectedFolders)
	}
	if m.sortStrategy != "ARTIST_YEAR" {
		t.Errorf("Expected pre-populated sortStrategy 'ARTIST_YEAR', got %q", m.sortStrategy)
	}
}

func TestOrgConfigEditUpdatesExistingInPlaceWithoutDuplicates(t *testing.T) {
	var calledConfig *pb.GramophileConfig
	client := &mockOrgClient{
		setConfigFunc: func(req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
			calledConfig = req.GetConfig()
			return &pb.SetConfigResponse{}, nil
		},
	}

	m := InitialModel(client, client, client)
	m.state = StateConfigSelect
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "12 Inches", Id: 10},
			{Name: "7 Inches", Id: 20},
		},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "Main Kallax",
						Spaces: []*pb.Space{
							{Name: "Expedit", Units: 4, Width: 12.0},
						},
						Foldersets: []*pb.FolderSet{
							{Name: "12 Inches", Folder: 10, Sort: pb.Sort_RELEASE_YEAR},
						},
					},
					{
						Name: "Singles Box",
						Spaces: []*pb.Space{
							{Name: "Crate", Units: 1, Width: 7.0},
						},
						Foldersets: []*pb.FolderSet{
							{Name: "7 Inches", Folder: 20, Sort: pb.Sort_ADDITION_DATE},
						},
					},
				},
			},
		},
	}

	m.initConfigSelectForm()
	m.configTarget = "Main Kallax"
	m.form.State = huh.StateCompleted

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)

	if m.state != StateOrgConfig {
		t.Fatalf("Expected StateOrgConfig, got %v", m.state)
	}

	// Modify fields of the pre-populated organisation
	m.spaceUnits = "16"
	m.spaceWidth = "13.5"
	m.selectedFolders = []string{"10", "20"}
	m.sortStrategy = "ARTIST_YEAR"
	m.form.State = huh.StateCompleted

	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)
	if cmd == nil {
		t.Fatalf("Expected command to call SetConfig")
	}

	msg := cmd()
	newModel, _ = m.Update(msg)
	m = newModel.(Model)

	if m.state != StateMainApp {
		t.Errorf("Expected transition to StateMainApp, got %v", m.state)
	}

	if calledConfig == nil {
		t.Fatalf("Expected SetConfig to be called")
	}

	orgs := calledConfig.GetOrganisationConfig().GetOrganisations()
	if len(orgs) != 2 {
		t.Fatalf("Expected 2 organisations (updated in place without duplicates), got %d", len(orgs))
	}

	edited := orgs[0]
	if edited.GetName() != "Main Kallax" {
		t.Errorf("Expected first org name 'Main Kallax', got %q", edited.GetName())
	}
	if len(edited.GetSpaces()) != 1 || edited.GetSpaces()[0].GetUnits() != 16 || edited.GetSpaces()[0].GetWidth() != 13.5 {
		t.Errorf("Expected updated space units=16 width=13.5, got %+v", edited.GetSpaces())
	}
	if len(edited.GetFoldersets()) != 2 || edited.GetFoldersets()[0].GetSort() != pb.Sort_ARTIST_YEAR {
		t.Errorf("Expected 2 foldersets with ARTIST_YEAR sort, got %+v", edited.GetFoldersets())
	}

	untouched := orgs[1]
	if untouched.GetName() != "Singles Box" {
		t.Errorf("Expected second org 'Singles Box' to remain untouched, got %q", untouched.GetName())
	}
}

func TestOrgConfigReconfiguringSameNameUpdatesInPlace(t *testing.T) {
	var calledConfig *pb.GramophileConfig
	client := &mockOrgClient{
		setConfigFunc: func(req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
			calledConfig = req.GetConfig()
			return &pb.SetConfigResponse{}, nil
		},
	}

	m := InitialModel(client, client, client)
	m.state = StateOrgConfig
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "Inbox", Id: 123},
		},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "My Test Org",
						Spaces: []*pb.Space{
							{Name: "Old Shelf", Units: 2, Width: 10.0},
						},
						Foldersets: []*pb.FolderSet{
							{Name: "Inbox", Folder: 123, Sort: pb.Sort_RELEASE_YEAR},
						},
					},
				},
			},
		},
	}

	m.initOrgConfigForm()
	m.orgName = "My Test Org"
	m.spaceName = "Updated Shelf"
	m.spaceUnits = "6"
	m.spaceWidth = "14.0"
	m.selectedFolders = []string{"123"}
	m.sortStrategy = "ADDITION_DATE"
	m.form.State = huh.StateCompleted

	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)
	if cmd == nil {
		t.Fatalf("Expected command to call SetConfig")
	}

	msg := cmd()
	newModel, _ = m.Update(msg)
	m = newModel.(Model)

	orgs := calledConfig.GetOrganisationConfig().GetOrganisations()
	if len(orgs) != 1 {
		t.Fatalf("Expected 1 organisation after reconfiguring existing name, got %d", len(orgs))
	}
	if orgs[0].GetSpaces()[0].GetName() != "Updated Shelf" || orgs[0].GetSpaces()[0].GetUnits() != 6 {
		t.Errorf("Expected existing organisation to be updated in place, got %+v", orgs[0].GetSpaces()[0])
	}
}

func TestReturningUserStartup_LoadsUserProfile(t *testing.T) {
	var seenAuthToken string
	client := &mockOrgClient{
		mockClient: mockClient{
			getUserCtxFunc: func(ctx context.Context) (*pb.GetUserResponse, error) {
				if md, ok := metadata.FromOutgoingContext(ctx); ok {
					vals := md.Get("auth-token")
					if len(vals) > 0 {
						seenAuthToken = vals[0]
					}
				}
				return &pb.GetUserResponse{
					User: &pb.StoredUser{
						User:  &pbd.User{Username: "returninguser"},
						State: pb.StoredUser_USER_STATE_LIVE,
						Folders: []*pbd.Folder{
							{Name: "12 Inches", Id: 10},
							{Name: "7 Inches", Id: 20},
						},
						Config: &pb.GramophileConfig{
							Basis: pb.Basis_GRAMOPHILE,
							OrganisationConfig: &pb.OrganisationConfig{
								Organisations: []*pb.Organisation{
									{Name: "Main Kallax"},
								},
							},
						},
					},
				}, nil
			},
		},
	}

	m := InitialModel(client, client, client)
	m.cacheManager = NewCacheManager(filepath.Join(t.TempDir(), "cache.pb"))
	m.tokenLoader = func() (string, error) {
		return "returning-auth-token", nil
	}

	newModel, cmd := m.Update(m.Init()())
	m = newModel.(Model)
	if m.state != StateMainApp {
		t.Fatalf("Expected StateMainApp on returning-user startup, got %v", m.state)
	}
	if cmd == nil {
		t.Fatalf("Expected startup command to fetch user profile")
	}

	newModel, _ = m.Update(cmd())
	m = newModel.(Model)

	if seenAuthToken != "returning-auth-token" {
		t.Errorf("Expected auth-token metadata 'returning-auth-token', got %q", seenAuthToken)
	}
	if m.user == nil {
		t.Fatalf("Expected m.user to be populated after startup GetUser")
	}
	if m.userLoading {
		t.Errorf("Expected m.userLoading to be false after user fetch completes")
	}

	// 1. configure -> StateConfigSelect should list Main Kallax
	m, _ = sendCommandString(m, "configure")
	if m.state != StateConfigSelect {
		t.Fatalf("Expected StateConfigSelect on 'configure', got %v", m.state)
	}
	if !strings.Contains(m.View(), "Main Kallax") {
		t.Errorf("Expected StateConfigSelect view to list 'Main Kallax', got:\n%s", m.View())
	}
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	// 2. configure org -> StateOrgConfig should display Discogs folders
	m, _ = sendCommandString(m, "configure org")
	if m.state != StateOrgConfig {
		t.Fatalf("Expected StateOrgConfig on 'configure org', got %v", m.state)
	}
	view := m.View()
	if !strings.Contains(view, "12 Inches") || !strings.Contains(view, "7 Inches") {
		t.Errorf("Expected StateOrgConfig view to show Discogs folders '12 Inches' and '7 Inches', got:\n%s", view)
	}
	m.form.State = huh.StateAborted
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)

	// 3. org (no args) -> defaults to Main Kallax
	m, orgCmd := sendCommandString(m, "org")
	if m.state != StateOrgView {
		t.Fatalf("Expected StateOrgView on 'org', got %v (inlineErr=%q)", m.state, m.inlineErrMsg)
	}
	if m.activeOrgName != "Main Kallax" {
		t.Errorf("Expected activeOrgName 'Main Kallax', got %q", m.activeOrgName)
	}
	if orgCmd == nil {
		t.Errorf("Expected fetchOrgCmd to be returned for 'org'")
	}
}

func TestConfigGuard_DeferredUntilUserLoaded(t *testing.T) {
	loadedUser := &pb.StoredUser{
		Folders: []*pbd.Folder{
			{Name: "12 Inches", Id: 10},
		},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{Name: "Main Kallax"},
				},
			},
		},
	}

	t.Run("in-flight startup fetch", func(t *testing.T) {
		getUserCalls := 0
		client := &mockOrgClient{
			mockClient: mockClient{
				getUserFunc: func() (*pb.GetUserResponse, error) {
					getUserCalls++
					return &pb.GetUserResponse{User: loadedUser}, nil
				},
			},
		}
		m := InitialModel(client, client, client)
		m.state = StateMainApp
		m.user = nil
		m.userLoading = true

		m, cmd := sendCommandString(m, "configure")
		if m.state != StateConfigSelect {
			t.Fatalf("Expected StateConfigSelect, got %v", m.state)
		}
		if m.form != nil {
			t.Errorf("Expected m.form to be nil while user is still loading")
		}
		if cmd != nil {
			t.Errorf("Expected nil cmd to avoid duplicate GetUser RPC while fetch is in flight")
		}
		if getUserCalls != 0 {
			t.Errorf("Expected 0 duplicate GetUser calls, got %d", getUserCalls)
		}
		if !strings.Contains(m.View(), "Loading configuration options...") {
			t.Errorf("Expected loading message in View, got:\n%s", m.View())
		}

		newModel, _ := m.Update(userFetchedMsg{user: loadedUser})
		m = newModel.(Model)
		if m.user == nil {
			t.Fatalf("Expected m.user to be populated")
		}
		if m.form == nil {
			t.Fatalf("Expected m.form to be initialized after userFetchedMsg")
		}
		if !strings.Contains(m.View(), "Main Kallax") {
			t.Errorf("Expected StateConfigSelect view to list 'Main Kallax' after load, got:\n%s", m.View())
		}
	})

	t.Run("not yet loading triggers fresh GetUser fetch", func(t *testing.T) {
		client := &mockOrgClient{
			mockClient: mockClient{
				getUserFunc: func() (*pb.GetUserResponse, error) {
					return &pb.GetUserResponse{User: loadedUser}, nil
				},
			},
		}
		m := InitialModel(client, client, client)
		m.state = StateMainApp
		m.user = nil
		m.userLoading = false

		m, cmd := sendCommandString(m, "configure org")
		if m.state != StateOrgConfig {
			t.Fatalf("Expected StateOrgConfig, got %v", m.state)
		}
		if m.form != nil {
			t.Errorf("Expected m.form to be nil before user is loaded")
		}
		if cmd == nil {
			t.Fatalf("Expected fetchUserCmd when user is nil and not yet loading")
		}

		newModel, _ := m.Update(cmd())
		m = newModel.(Model)
		if m.user == nil {
			t.Fatalf("Expected m.user to be populated")
		}
		if m.form == nil {
			t.Fatalf("Expected m.form to be initialized once user loads")
		}
		if !strings.Contains(m.View(), "12 Inches") {
			t.Errorf("Expected Discogs folder '12 Inches' in StateOrgConfig view, got:\n%s", m.View())
		}
	})

	t.Run("GetUser failure surfaces error and allows Esc", func(t *testing.T) {
		client := &mockOrgClient{}
		m := InitialModel(client, client, client)
		m.state = StateMainApp
		m.user = nil
		m.userLoading = true

		m, _ = sendCommandString(m, "configure")
		newModel, _ := m.Update(userFetchedMsg{err: fmt.Errorf("unavailable")})
		m = newModel.(Model)

		if m.form != nil {
			t.Errorf("Expected m.form to remain nil on user fetch error")
		}
		if m.orgErr == "" {
			t.Errorf("Expected m.orgErr to be populated on user fetch error")
		}
		if !strings.Contains(m.View(), "unavailable") {
			t.Errorf("Expected view to display error 'unavailable', got:\n%s", m.View())
		}

		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = newModel.(Model)
		if m.state != StateMainApp {
			t.Errorf("Expected Esc to return cleanly to StateMainApp, got %v", m.state)
		}

		// Also verify Esc when m.form == nil in StateOrgConfig
		m.user = nil
		m.userLoading = true
		m, _ = sendCommandString(m, "configure org")
		newModel, _ = m.Update(userFetchedMsg{err: fmt.Errorf("unavailable")})
		m = newModel.(Model)
		if m.form != nil || !strings.Contains(m.View(), "unavailable") {
			t.Errorf("Expected error in StateOrgConfig with nil form, got orgErr=%q view=%s", m.orgErr, m.View())
		}
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = newModel.(Model)
		if m.state != StateMainApp {
			t.Errorf("Expected Esc in StateOrgConfig with nil form to return to StateMainApp, got %v", m.state)
		}
	})
}

func TestOrgConfigSubmissionBlockedWhenUserNil(t *testing.T) {
	setConfigCalled := false
	client := &mockOrgClient{
		setConfigFunc: func(req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
			setConfigCalled = true
			return &pb.SetConfigResponse{}, nil
		},
	}

	m := InitialModel(client, client, client)
	m.state = StateOrgConfig
	m.user = &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "Inbox", Id: 123}},
	}
	m.initOrgConfigForm()
	m.user = nil
	m.orgName = "My Test Org"
	m.spaceName = "Main Shelf"
	m.spaceUnits = "2"
	m.spaceWidth = "12.5"
	m.selectedFolders = []string{"123"}
	m.sortStrategy = "RELEASE_YEAR"
	m.form.State = huh.StateCompleted

	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)

	if setConfigCalled || cmd != nil {
		t.Errorf("Expected SetConfig NOT to be called when m.user is nil (setConfigCalled=%v, cmd=%v)", setConfigCalled, cmd)
	}
	if m.state != StateOrgConfig {
		t.Errorf("Expected state to remain StateOrgConfig, got %v", m.state)
	}
	if m.orgErr == "" {
		t.Errorf("Expected m.orgErr to be non-empty")
	}
	if !strings.Contains(m.View(), m.orgErr) {
		t.Errorf("Expected view to contain error %q, got:\n%s", m.orgErr, m.View())
	}
}

