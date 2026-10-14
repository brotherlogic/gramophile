package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	pstore_client "github.com/brotherlogic/pstore/client"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type tuiIntegrationHarness struct {
	ctx    context.Context
	d      db.Database
	di     *discogs.TestDiscogsClient
	qc     *queuelogic.Queue
	s      *server.Server
	client pb.GramophileEServiceClient
}

func setupTUIIntegrationHarness(t *testing.T) *tuiIntegrationHarness {
	t.Helper()
	ctx := context.Background()

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{
			{Id: 10, Name: "Goal Folder"},
			{Id: 11, Name: "Width"},
		},
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterGramophileEServiceServer(grpcServer, s)
	go func() {
		_ = grpcServer.Serve(lis)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := pb.NewGramophileEServiceClient(conn)
	return &tuiIntegrationHarness{
		ctx:    ctx,
		d:      d,
		di:     di,
		qc:     qc,
		s:      s,
		client: client,
	}
}

func newIntegrationModel(t *testing.T, h *tuiIntegrationHarness) (Model, *string) {
	t.Helper()
	var savedToken string
	m := InitialModel(h.client, h.client, h.client)
	m.cacheManager = NewCacheManager(filepath.Join(t.TempDir(), "collection_cache.pb"))
	m.tokenLoader = func() (string, error) {
		if savedToken == "" {
			return "", os.ErrNotExist
		}
		return savedToken, nil
	}
	m.tokenSaver = func(tok string) error {
		savedToken = tok
		return nil
	}
	return m, &savedToken
}

func sendCommandString(m Model, command string) (Model, tea.Cmd) {
	for _, r := range command {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(Model), cmd
}

func TestIntegration_LoginAndInitialSyncJourney(t *testing.T) {
	h := setupTUIIntegrationHarness(t)
	m, savedToken := newIntegrationModel(t, h)

	// 1. Startup logo transitions to StateLogin when no token is saved
	next, cmd := m.Update(timeoutMsg{})
	m = next.(Model)
	if m.state != StateLogin {
		t.Fatalf("Expected StateLogin, got %v", m.state)
	}
	if cmd == nil {
		t.Fatalf("Expected fetchURL command on entering StateLogin")
	}

	// 2. Execute GetURL against the in-process server via bufconn
	msg := cmd()
	next, cmd = m.Update(msg)
	m = next.(Model)
	if m.err != nil {
		t.Fatalf("Unexpected error fetching login URL: %v", m.err)
	}
	if m.loginToken != "madeuptoken" {
		t.Fatalf("Expected loginToken 'madeuptoken', got %q", m.loginToken)
	}
	if cmd == nil {
		t.Fatalf("Expected pollLogin command after receiving URL")
	}

	// 3. Polling before Discogs callback completes should schedule a retry without transitioning
	preCallbackMsg := cmd()
	next, retryCmd := m.Update(preCallbackMsg)
	m = next.(Model)
	if m.state != StateLogin {
		t.Fatalf("Expected to remain in StateLogin before callback, got %v", m.state)
	}
	if retryCmd == nil {
		t.Fatalf("Expected retry tick command when login is not yet completed")
	}

	// 4. Simulate Discogs OAuth callback populating user credentials in database
	logins, err := h.d.LoadLogins(h.ctx)
	if err != nil {
		t.Fatalf("Failed to load logins from db: %v", err)
	}
	foundAttempt := false
	for _, attempt := range logins.GetAttempts() {
		if attempt.GetRequestToken() == m.loginToken {
			attempt.UserSecret = "discogs-secret-123"
			attempt.UserToken = "discogs-token-123"
			foundAttempt = true
		}
	}
	if !foundAttempt {
		t.Fatalf("Could not find login attempt for request token %q", m.loginToken)
	}
	if err := h.d.SaveLogins(h.ctx, logins); err != nil {
		t.Fatalf("Failed to save updated logins: %v", err)
	}

	// 5. Trigger loginPollMsg -> GetLogin RPC succeeds, saves token, transitions to StateLoadingSync
	next, cmd = m.Update(loginPollMsg{})
	m = next.(Model)
	if cmd == nil {
		t.Fatalf("Expected pollLogin command on loginPollMsg")
	}
	loginMsg := cmd()
	next, cmd = m.Update(loginMsg)
	m = next.(Model)

	if m.state != StateLoadingSync {
		t.Fatalf("Expected transition to StateLoadingSync after login success, got %v (err=%v)", m.state, m.err)
	}
	if *savedToken == "" {
		t.Fatalf("Expected auth token to be persisted via tokenSaver")
	}
	if m.authToken != *savedToken {
		t.Fatalf("Expected m.authToken=%q to match savedToken=%q", m.authToken, *savedToken)
	}
	if cmd == nil {
		t.Fatalf("Expected pollSync command upon entering StateLoadingSync")
	}

	// 6. Initial pollSync against server: user is in USER_STATE_REFRESHING
	syncMsg := cmd()
	next, _ = m.Update(syncMsg)
	m = next.(Model)
	if m.err != nil {
		t.Fatalf("Unexpected sync error: %v", m.err)
	}
	if m.state != StateLoadingSync {
		t.Fatalf("Expected to remain in StateLoadingSync while user is REFRESHING, got %v", m.state)
	}
	if m.user == nil || m.user.GetUser().GetUsername() != "madeupuser" {
		t.Fatalf("Expected synced user 'madeupuser', got %+v", m.user)
	}

	// 7. Update user state on server to IN_WAITLIST and seed collection progress
	storedUser, err := h.d.GetUser(h.ctx, m.authToken)
	if err != nil {
		t.Fatalf("Failed to fetch stored user from db: %v", err)
	}
	storedUser.ExpectedCollectionSize = 2
	storedUser.State = pb.StoredUser_USER_STATE_IN_WAITLIST
	if err := h.d.SaveUser(h.ctx, storedUser); err != nil {
		t.Fatalf("Failed to save waitlist user state: %v", err)
	}
	if err := h.d.SaveRecord(h.ctx, storedUser.GetUser().GetDiscogsUserId(), &pb.Record{
		Release: &pbd.Release{Id: 1001, InstanceId: 2001, Title: "First Record"},
	}, &db.SaveOptions{}); err != nil {
		t.Fatalf("Failed to seed record: %v", err)
	}

	next, cmd = m.Update(syncPollMsg{})
	m = next.(Model)
	waitlistMsg := cmd()
	next, _ = m.Update(waitlistMsg)
	m = next.(Model)

	if m.state != StateWaitlist {
		t.Fatalf("Expected transition to StateWaitlist, got %v", m.state)
	}
	if m.progress != 0.5 {
		t.Errorf("Expected sync progress 0.5 (1/2 records), got %v", m.progress)
	}
	if !strings.Contains(m.View(), "Waiting for Admin Approval") {
		t.Errorf("Expected waitlist message in view, got:\n%s", m.View())
	}

	// 8. Upgrade user to USER_STATE_LIVE via server UpgradeUser RPC and poll sync
	if _, err := h.s.UpgradeUser(h.ctx, &pb.UpgradeUserRequest{
		Username: "madeupuser",
		NewState: pb.StoredUser_USER_STATE_LIVE,
	}); err != nil {
		t.Fatalf("UpgradeUser failed: %v", err)
	}

	next, cmd = m.Update(syncPollMsg{})
	m = next.(Model)
	liveMsg := cmd()
	next, _ = m.Update(liveMsg)
	m = next.(Model)

	if m.state != StateMainApp {
		t.Fatalf("Expected transition to StateMainApp once user is LIVE, got %v", m.state)
	}
	if m.user.GetState() != pb.StoredUser_USER_STATE_LIVE {
		t.Errorf("Expected user state LIVE, got %v", m.user.GetState())
	}
}

func TestIntegration_ManualTokenLoginJourney(t *testing.T) {
	h := setupTUIIntegrationHarness(t)

	// Seed a live user in the database
	const manualToken = "manual-live-auth-token"
	err := h.d.SaveUser(h.ctx, &pb.StoredUser{
		User:  &pbd.User{DiscogsUserId: 123, Username: "manualuser", UserToken: "tok", UserSecret: "sec"},
		Auth:  &pb.GramophileAuth{Token: manualToken},
		State: pb.StoredUser_USER_STATE_LIVE,
	})
	if err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}

	m, savedToken := newIntegrationModel(t, h)
	m.state = StateLogin
	m.loginInput.Focus()

	// Type the manual token and press Enter
	m, cmd := sendCommandString(m, manualToken)
	if m.state != StateLoadingSync {
		t.Fatalf("Expected StateLoadingSync after submitting manual token, got %v", m.state)
	}
	if *savedToken != manualToken {
		t.Errorf("Expected savedToken=%q, got %q", manualToken, *savedToken)
	}
	if cmd == nil {
		t.Fatalf("Expected pollSync command after manual token submission")
	}

	// Execute pollSync over bufconn gRPC client
	syncMsg := cmd()
	next, _ := m.Update(syncMsg)
	m = next.(Model)

	if m.state != StateMainApp {
		t.Fatalf("Expected transition to StateMainApp for live user, got %v (err=%v)", m.state, m.err)
	}
	if m.user == nil || m.user.GetUser().GetUsername() != "manualuser" {
		t.Errorf("Expected resolved user 'manualuser', got %+v", m.user)
	}
}

func TestIntegration_OrganisationCreateViewAndUpdateJourney(t *testing.T) {
	h := setupTUIIntegrationHarness(t)

	const authToken = "org-journey-token"
	seededUser := &pb.StoredUser{
		User: &pbd.User{
			DiscogsUserId: 123,
			Username:      "orguser",
			UserToken:     "tok",
			UserSecret:    "sec",
		},
		Auth:  &pb.GramophileAuth{Token: authToken},
		State: pb.StoredUser_USER_STATE_LIVE,
		Folders: []*pbd.Folder{
			{Id: 10, Name: "12 Inches"},
			{Id: 20, Name: "7 Inches"},
		},
		Config: &pb.GramophileConfig{},
	}
	if err := h.d.SaveUser(h.ctx, seededUser); err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}

	// Seed two records across the two folders
	if err := h.d.SaveRecord(h.ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:          501,
			InstanceId:  5001,
			FolderId:    10,
			Title:       "Blue Monday",
			ReleaseDate: 1983,
			Artists:     []*pbd.Artist{{Name: "New Order"}},
		},
	}, &db.SaveOptions{}); err != nil {
		t.Fatalf("Failed to save record 5001: %v", err)
	}
	if err := h.d.SaveRecord(h.ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:          502,
			InstanceId:  5002,
			FolderId:    20,
			Title:       "Transmission",
			ReleaseDate: 1979,
			Artists:     []*pbd.Artist{{Name: "Joy Division"}},
		},
	}, &db.SaveOptions{}); err != nil {
		t.Fatalf("Failed to save record 5002: %v", err)
	}

	m, _ := newIntegrationModel(t, h)
	m.authToken = authToken
	m.state = StateMainApp
	m.user = seededUser

	// 1. Enter "configure org" in StateMainApp to create a new organisation
	m, cmd := sendCommandString(m, "configure org")
	if m.state != StateOrgConfig {
		t.Fatalf("Expected StateOrgConfig, got %v", m.state)
	}
	if cmd != nil {
		t.Fatalf("Expected nil cmd on entering StateOrgConfig, got %v", cmd)
	}

	// Fill out the form fields for creating "Main Shelf Org"
	m.orgName = "Main Shelf Org"
	m.spaceName = "Kallax Top"
	m.spaceUnits = "2"
	m.spaceWidth = "15"
	m.selectedFolders = []string{"10"}
	m.sortStrategy = "RELEASE_YEAR"
	m.form.State = huh.StateCompleted

	next, setConfigCmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	if setConfigCmd == nil {
		t.Fatalf("Expected pollSetConfig command when form completes")
	}

	// Execute SetConfig RPC against the in-process server
	setConfigRes := setConfigCmd()
	next, _ = m.Update(setConfigRes)
	m = next.(Model)

	if m.state != StateMainApp {
		t.Fatalf("Expected transition back to StateMainApp after SetConfig, got %v (orgErr=%q)", m.state, m.orgErr)
	}

	if err := h.qc.FlushQueue(h.ctx); err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	// Verify organisation was persisted in the server database
	authCtx := metadata.AppendToOutgoingContext(h.ctx, "auth-token", authToken)
	userResp, err := h.client.GetUser(authCtx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	orgs := userResp.GetUser().GetConfig().GetOrganisationConfig().GetOrganisations()
	if len(orgs) != 1 {
		t.Fatalf("Expected 1 persisted organisation on server, got %d", len(orgs))
	}
	if orgs[0].GetName() != "Main Shelf Org" || len(orgs[0].GetSpaces()) != 1 || orgs[0].GetSpaces()[0].GetUnits() != 2 {
		t.Errorf("Unexpected persisted organisation: %+v", orgs[0])
	}

	// 2. View the organisation layout via "org Main Shelf Org" command in TUI
	m, orgCmd := sendCommandString(m, "org Main Shelf Org")
	if m.state != StateOrgView {
		t.Fatalf("Expected StateOrgView after 'org Main Shelf Org', got %v", m.state)
	}
	if orgCmd == nil {
		t.Fatalf("Expected fetchOrgCmd when entering StateOrgView")
	}

	// Execute GetOrg RPC -> builds and saves snapshot on server, then dispatches record resolution
	orgMsg := orgCmd()
	next, recBatchCmd := m.Update(orgMsg)
	m = next.(Model)
	if m.inlineErrMsg != "" {
		t.Fatalf("Unexpected error in StateOrgView: %s", m.inlineErrMsg)
	}
	if len(m.orgPlacements) != 1 {
		t.Fatalf("Expected 1 placement in snapshot for folder 10, got %d", len(m.orgPlacements))
	}
	if recBatchCmd == nil {
		t.Fatalf("Expected batch command to resolve placement records")
	}

	// Execute record resolution batch command against GetRecord RPC
	batchMsg := recBatchCmd()
	next, _ = m.Update(batchMsg)
	m = next.(Model)

	orgViewOutput := m.View()
	if !strings.Contains(orgViewOutput, "New Order - Blue Monday") {
		t.Errorf("Expected resolved record 'New Order - Blue Monday' in StateOrgView, got:\n%s", orgViewOutput)
	}
	if !strings.Contains(orgViewOutput, "Kallax Top") {
		t.Errorf("Expected space 'Kallax Top' in StateOrgView, got:\n%s", orgViewOutput)
	}

	// Exit StateOrgView back to StateMainApp
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.state != StateMainApp {
		t.Fatalf("Expected StateMainApp after Esc from StateOrgView, got %v", m.state)
	}

	// 3. Re-open configuration via "configure" -> StateConfigSelect -> select existing "Main Shelf Org"
	m, _ = sendCommandString(m, "configure")
	if m.state != StateConfigSelect {
		t.Fatalf("Expected StateConfigSelect after 'configure', got %v", m.state)
	}
	m.configTarget = "Main Shelf Org"
	m.form.State = huh.StateCompleted
	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)

	if m.state != StateOrgConfig {
		t.Fatalf("Expected transition to StateOrgConfig when editing existing org, got %v", m.state)
	}
	if m.editingOrgName != "Main Shelf Org" {
		t.Errorf("Expected editingOrgName='Main Shelf Org', got %q", m.editingOrgName)
	}
	// Verify form pre-population from existing org
	if m.orgName != "Main Shelf Org" || m.spaceName != "Kallax Top" || m.spaceUnits != "2" || m.spaceWidth != "15" {
		t.Errorf("Expected pre-populated form values, got name=%q space=%q units=%q width=%q",
			m.orgName, m.spaceName, m.spaceUnits, m.spaceWidth)
	}

	// Modify properties (update units to 4, width to 25, include both folders 10 and 20)
	m.spaceUnits = "4"
	m.spaceWidth = "25"
	m.selectedFolders = []string{"10", "20"}
	m.form.State = huh.StateCompleted

	next, updateConfigCmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	if updateConfigCmd == nil {
		t.Fatalf("Expected pollSetConfig command on updating existing org")
	}
	updateRes := updateConfigCmd()
	next, _ = m.Update(updateRes)
	m = next.(Model)

	if m.state != StateMainApp {
		t.Fatalf("Expected StateMainApp after updating org, got %v (orgErr=%q)", m.state, m.orgErr)
	}

	// Verify in-place update on server without duplicate organisations
	userRespAfter, err := h.client.GetUser(authCtx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("GetUser after update failed: %v", err)
	}
	updatedOrgs := userRespAfter.GetUser().GetConfig().GetOrganisationConfig().GetOrganisations()
	if len(updatedOrgs) != 1 {
		t.Fatalf("Expected exactly 1 organisation after in-place edit (no duplicates), got %d", len(updatedOrgs))
	}
	if updatedOrgs[0].GetSpaces()[0].GetUnits() != 4 || updatedOrgs[0].GetSpaces()[0].GetWidth() != 25 {
		t.Errorf("Expected updated space units=4 width=25, got %+v", updatedOrgs[0].GetSpaces()[0])
	}
	if len(updatedOrgs[0].GetFoldersets()) != 2 {
		t.Errorf("Expected 2 foldersets after update, got %d", len(updatedOrgs[0].GetFoldersets()))
	}
}

func TestIntegration_LocateRecordJourney(t *testing.T) {
	h := setupTUIIntegrationHarness(t)

	const authToken = "locate-journey-token"
	user := &pb.StoredUser{
		User: &pbd.User{
			DiscogsUserId: 123,
			Username:      "locateuser",
			UserToken:     "tok",
			UserSecret:    "sec",
		},
		Auth:  &pb.GramophileAuth{Token: authToken},
		State: pb.StoredUser_USER_STATE_LIVE,
		Folders: []*pbd.Folder{
			{Id: 10, Name: "LP Shelf"},
		},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "Living Room Kallax",
						Spaces: []*pb.Space{
							{Name: "Cube A", Units: 2, Width: 50},
						},
						Foldersets: []*pb.FolderSet{
							{Name: "LP Shelf", Folder: 10, Sort: pb.Sort_RELEASE_YEAR},
						},
					},
				},
			},
		},
	}
	if err := h.d.SaveUser(h.ctx, user); err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Seed 3 records in folder 10 so LocateRecord has before and after context
	records := []*pb.Record{
		{
			Release: &pbd.Release{
				Id:          701,
				InstanceId:  7001,
				FolderId:    10,
				Title:       "Unknown Pleasures",
				ReleaseDate: 1979,
				Artists:     []*pbd.Artist{{Name: "Joy Division"}},
			},
		},
		{
			Release: &pbd.Release{
				Id:          702,
				InstanceId:  7002,
				FolderId:    10,
				Title:       "Closer",
				ReleaseDate: 1980,
				Artists:     []*pbd.Artist{{Name: "Joy Division"}},
			},
		},
		{
			Release: &pbd.Release{
				Id:          703,
				InstanceId:  7003,
				FolderId:    10,
				Title:       "Movement",
				ReleaseDate: 1981,
				Artists:     []*pbd.Artist{{Name: "New Order"}},
			},
		},
	}
	for _, r := range records {
		if err := h.d.SaveRecord(h.ctx, 123, r, &db.SaveOptions{}); err != nil {
			t.Fatalf("Failed to seed record %d: %v", r.GetRelease().GetId(), err)
		}
	}

	// Build and persist the organisation snapshot via GetOrg on the server
	authCtx := metadata.AppendToOutgoingContext(h.ctx, "auth-token", authToken)
	if _, err := h.client.GetOrg(authCtx, &pb.GetOrgRequest{OrgName: "Living Room Kallax"}); err != nil {
		t.Fatalf("GetOrg failed to build initial snapshot: %v", err)
	}

	m, _ := newIntegrationModel(t, h)
	m.authToken = authToken
	m.state = StateMainApp
	m.user = user

	// 1. Direct "locate 702" command from StateMainApp
	m, locateCmd := sendCommandString(m, "locate 702")
	if m.state != StateLocateView {
		t.Fatalf("Expected StateLocateView after 'locate 702', got %v", m.state)
	}
	if locateCmd == nil {
		t.Fatalf("Expected fetchLocateCmd for release 702")
	}

	locateMsg := locateCmd()
	next, _ := m.Update(locateMsg)
	m = next.(Model)

	if m.inlineErrMsg != "" {
		t.Fatalf("Unexpected locate error: %s", m.inlineErrMsg)
	}
	view := m.View()
	if !strings.Contains(view, "Joy Division - Closer is in Living Room Kallax, Slot 1") {
		t.Errorf("Expected location header in StateLocateView, got:\n%s", view)
	}
	if !strings.Contains(view, "Joy Division - Unknown Pleasures") {
		t.Errorf("Expected before-context record 'Joy Division - Unknown Pleasures' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "New Order - Movement") {
		t.Errorf("Expected after-context record 'New Order - Movement' in view, got:\n%s", view)
	}

	// Return to StateMainApp
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.state != StateMainApp {
		t.Fatalf("Expected StateMainApp after Esc from StateLocateView, got %v", m.state)
	}

	// 2. Interactive search via "locate" -> StateLocateSearch -> GetRecord(GetAllRecords) -> filter -> Enter -> StateLocateView
	m, fetchColCmd := sendCommandString(m, "locate")
	if m.state != StateLocateSearch {
		t.Fatalf("Expected StateLocateSearch after 'locate', got %v", m.state)
	}
	if fetchColCmd == nil {
		t.Fatalf("Expected fetchCollectionIndexCmd when entering StateLocateSearch with empty index")
	}

	colMsg := fetchColCmd()
	next, _ = m.Update(colMsg)
	m = next.(Model)

	if len(m.indexedReleases) != 3 {
		t.Fatalf("Expected 3 indexed releases loaded from server, got %d", len(m.indexedReleases))
	}

	// Filter search input by typing "Movement"
	for _, r := range "Movement" {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}
	if len(m.filteredReleases) != 1 || m.filteredReleases[0].releaseID != 703 {
		t.Fatalf("Expected filtered release 703 ('Movement'), got %+v", m.filteredReleases)
	}

	// Press Enter to locate the selected release
	next, locateFromSearchCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.state != StateLocateView {
		t.Fatalf("Expected transition to StateLocateView on Enter in StateLocateSearch, got %v", m.state)
	}
	if locateFromSearchCmd == nil {
		t.Fatalf("Expected fetchLocateCmd after selecting release in StateLocateSearch")
	}

	locateSearchRes := locateFromSearchCmd()
	next, _ = m.Update(locateSearchRes)
	m = next.(Model)

	searchLocateView := m.View()
	if !strings.Contains(searchLocateView, "New Order - Movement is in Living Room Kallax, Slot 1") {
		t.Errorf("Expected 'New Order - Movement' location in view, got:\n%s", searchLocateView)
	}
}
