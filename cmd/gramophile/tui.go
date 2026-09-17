package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/prototext"
)

type appState int

const (
	StateStartupLogo appState = iota
	StateLogin
	StateLoadingSync
	StateWaitlist
	StateMainApp
	StateOrgConfig
	StateOrgView
	StateLocateView
	StateConfigSelect
	StateLocateSearch
	StateLocateVersionSelect
)

type AuthClient interface {
	GetURL(ctx context.Context, in *pb.GetURLRequest, opts ...grpc.CallOption) (*pb.GetURLResponse, error)
	GetLogin(ctx context.Context, in *pb.GetLoginRequest, opts ...grpc.CallOption) (*pb.GetLoginResponse, error)
	GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error)
	GetState(ctx context.Context, in *pb.GetStateRequest, opts ...grpc.CallOption) (*pb.GetStateResponse, error)
	SetConfig(ctx context.Context, in *pb.SetConfigRequest, opts ...grpc.CallOption) (*pb.SetConfigResponse, error)
}

type OrgClient interface {
	GetOrg(ctx context.Context, in *pb.GetOrgRequest, opts ...grpc.CallOption) (*pb.GetOrgResponse, error)
	GetRecord(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error)
}

type LocateClient interface {
	LocateRecord(ctx context.Context, in *pb.LocateRecordRequest, opts ...grpc.CallOption) (*pb.LocateRecordResponse, error)
}

type timeoutMsg struct{}
type urlFetchedMsg struct {
	url   string
	token string
}
type urlFetchErrMsg struct{ err error }
type loginSuccessMsg struct {
	auth *pb.GramophileAuth
}
type loginErrMsg struct{ err error }
type loginPollMsg struct{}
type syncPollMsg struct{}
type syncStatusMsg struct {
	expectedSize int32
	currentSize  int32
	userState    pb.StoredUser_UserState
	user         *pb.StoredUser
	err          error
}

type setConfigMsg struct {
	err error
}

type orgFetchedMsg struct {
	snapshot *pb.OrganisationSnapshot
	err      error
}

type recordFetchedMsg struct {
	iid    int64
	record *pb.Record
	err    error
}

type locateFetchedMsg struct {
	releaseID int64
	response  *pb.LocateRecordResponse
	err       error
}

type checkForUpdateMsg struct{}
type updateAvailableMsg struct{ release *UpdateRelease }
type updateStatusMsg struct{ status string }
type updateErrorMsg struct{ err error }
type updateCompleteMsg struct{}

type collectionFetchedMsg struct {
	records []*pb.Record
	err     error
}

type cacheLoadedMsg struct {
	cache   *pb.CollectionCache
	records []*pb.Record
	err     error
}

type cacheSyncStatusMsg struct {
	status  CacheStatus
	records []*pb.Record
	err     error
}

type releaseEntry struct {
	releaseID int64
	artist    string
	title     string
	searchKey string       // Precomputed lowercase: artist + " " + title
	records   []*pb.Record // Physical copies associated with this release
}

// initialLogoDuration is the time to show the logo before auto-transitioning
const initialLogoDuration = 2 * time.Second

type Model struct {
	state           appState
	client          AuthClient
	orgClient       OrgClient
	locateClient    LocateClient
	loginURL        string
	loginToken      string
	authToken       string
	err             error
	orgErr          string
	tokenLoader     func() (string, error)
	tokenSaver      func(string) error
	progress        float64
	progBar         progress.Model
	syncRetryCount  int
	loginRetryCount int
	user            *pb.StoredUser
	form            *huh.Form
	orgName         string
	spaceName       string
	spaceUnits      string
	spaceWidth      string
	selectedFolders []string
	sortStrategy    string
	configTarget    string

	commandInput    string
	textInput       textinput.Model
	orgViewport     viewport.Model
	orgSpinner      spinner.Model
	activeOrgName   string
	activeSlot      int32
	activeHash      string
	activeDebug     bool
	orgSnapshot     *pb.OrganisationSnapshot
	orgPlacements   []*pb.Placement
	resolvedRecords map[int64]*pb.Record
	totalWidth      int32
	inlineErrMsg    string
	showHelp        bool

	locateViewport viewport.Model
	activeLocateID int64
	locateResponse *pb.LocateRecordResponse

	version      string
	updater      Updater
	updateStatus string
	isUpdating   bool

	locateSearchInput     textinput.Model
	locateSearchCursor    int
	collectionIndex       []*pb.Record
	collectionLoading     bool
	filteredLocateRecords []*pb.Record
	locateSearchErr       string

	indexedReleases      []*releaseEntry // Alphabetically sorted master release index
	filteredReleases     []*releaseEntry // Active filtered subset
	activeReleaseChoice  *releaseEntry  // Selected release for version disambiguation
	versionSelectCursor  int            // Cursor in secondary version view
	locateViewportOffset int            // Window scroll offset for primary list

	cacheManager      *CacheManager
	cacheStatus       CacheStatus
	pendingPriorities map[int64]bool
	priorityFetchCmd  tea.Cmd
}

type priorityRecordResolvedMsg struct {
	record *pb.Record
	err    error
}



func defaultTokenLoader() (string, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(dirname, ".gramophile")
	text, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	auth := &pb.GramophileAuth{}
	err = prototext.Unmarshal(text, auth)
	if err != nil {
		return "", err
	}

	if auth.GetToken() == "" {
		return "", fmt.Errorf("empty auth token")
	}

	return auth.GetToken(), nil
}

func defaultTokenSaver(tokenText string) error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	tmpFile := filepath.Join(dirname, ".gramophile.tmp")
	finalFile := filepath.Join(dirname, ".gramophile")

	f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	auth := &pb.GramophileAuth{Token: tokenText}
	err = proto.MarshalText(f, auth)
	f.Close()

	if err != nil {
		return err
	}

	return os.Rename(tmpFile, finalFile)
}

func newOrgSpinner() spinner.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	return sp
}

func InitialModel(client AuthClient, orgClient OrgClient, locateClient LocateClient) Model {
	ti := textinput.New()
	ti.Placeholder = "locate <release_id> | org [name] | configure | quit"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	lsi := textinput.New()
	lsi.Placeholder = "Search collection by artist or title..."
	lsi.Focus()
	lsi.CharLimit = 256
	lsi.Width = 60

	return Model{
		state:             StateStartupLogo,
		client:            client,
		orgClient:         orgClient,
		locateClient:      locateClient,
		tokenLoader:       defaultTokenLoader,
		tokenSaver:        defaultTokenSaver,
		progBar:           progress.New(progress.WithDefaultGradient()),
		textInput:         ti,
		locateSearchInput: lsi,
		orgSpinner:        newOrgSpinner(),
		version:           "dev",
		cacheManager:      NewCacheManager(""),
		cacheStatus:       CacheStatusUninitialized,
		pendingPriorities: make(map[int64]bool),
	}
}

func checkUpdateCmd(m Model) tea.Cmd {
	return func() tea.Msg {
		if m.updater == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		release, err := m.updater.CheckForUpdate(ctx, m.version)
		if err != nil {
			return updateErrorMsg{err: err}
		}
		if release != nil {
			return updateAvailableMsg{release: release}
		}
		return nil
	}
}

func downloadAndRestartCmd(m Model, release *UpdateRelease) tea.Cmd {
	return func() tea.Msg {
		if m.updater == nil || release == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := m.updater.DownloadAndApply(ctx, release.DownloadURL, ""); err != nil {
			return updateErrorMsg{err: err}
		}
		if err := m.updater.Restart(""); err != nil {
			return updateErrorMsg{err: err}
		}
		return updateCompleteMsg{}
	}
}

func (m Model) buildContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if m.authToken != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "auth-token", m.authToken)
	}
	return context.WithTimeout(ctx, timeout)
}

func (m Model) fetchURL() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		resp, err := m.client.GetURL(ctx, &pb.GetURLRequest{})
		if err != nil {
			return urlFetchErrMsg{err: err}
		}
		return urlFetchedMsg{url: resp.GetURL(), token: resp.GetToken()}
	}
}

func (m Model) pollLogin() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		resp, err := m.client.GetLogin(ctx, &pb.GetLoginRequest{Token: m.loginToken})
		if err != nil {
			return loginErrMsg{err: err}
		}
		return loginSuccessMsg{auth: resp.GetAuth()}
	}
}

func (m Model) pollSync() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		
		userResp, err := m.client.GetUser(ctx, &pb.GetUserRequest{})
		if err != nil {
			return syncStatusMsg{err: err}
		}
		
		stateResp, err := m.client.GetState(ctx, &pb.GetStateRequest{})
		if err != nil {
			return syncStatusMsg{err: err}
		}

		return syncStatusMsg{
			expectedSize: userResp.GetUser().GetExpectedCollectionSize(),
			currentSize:  stateResp.GetCollectionSize(),
			userState:    userResp.GetUser().GetState(),
			user:         userResp.GetUser(),
		}
	}
}

func cachedToRecord(cr *pb.CachedRecord) *pb.Record {
	if cr == nil {
		return nil
	}
	var artists []*pbd.Artist
	if cr.GetArtist() != "" {
		artists = []*pbd.Artist{{Name: cr.GetArtist()}}
	}
	return &pb.Record{
		Release: &pbd.Release{
			Id:         cr.GetReleaseId(),
			InstanceId: cr.GetInstanceId(),
			Title:      cr.GetTitle(),
			Artists:    artists,
		},
		LastUpdateTime: cr.GetLastUpdatedTime(),
	}
}

func (m Model) loadDiskCacheCmd() tea.Cmd {
	return func() tea.Msg {
		if m.cacheManager == nil {
			return cacheLoadedMsg{err: fmt.Errorf("no cache manager initialized")}
		}
		if err := m.cacheManager.LoadFromDisk(); err != nil {
			return cacheLoadedMsg{err: err}
		}
		cachedRecords := m.cacheManager.GetCachedRecords()
		var records []*pb.Record
		for _, cr := range cachedRecords {
			records = append(records, cachedToRecord(cr))
		}
		return cacheLoadedMsg{
			cache:   m.cacheManager.cache,
			records: records,
		}
	}
}

func (m Model) syncCollectionCacheCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(60 * time.Second)
		defer cancel()
		if m.orgClient == nil {
			if m.cacheManager != nil {
				m.cacheManager.SetStatus(CacheStatusStale)
			}
			return cacheSyncStatusMsg{
				status: CacheStatusStale,
				err:    fmt.Errorf("no org client initialized"),
			}
		}

		resp, err := m.orgClient.GetRecord(ctx, &pb.GetRecordRequest{
			Request: &pb.GetRecordRequest_GetAllRecords{
				GetAllRecords: true,
			},
		})
		if err != nil {
			if m.cacheManager != nil {
				m.cacheManager.SetStatus(CacheStatusStale)
			}
			return cacheSyncStatusMsg{
				status: CacheStatusStale,
				err:    err,
			}
		}

		var records []*pb.Record
		if resp != nil {
			for _, r := range resp.GetRecords() {
				if r != nil && r.GetRecord() != nil {
					records = append(records, r.GetRecord())
				}
			}
		}

		if m.cacheManager != nil {
			m.cacheManager.Populate(records, time.Now())
			if err := m.cacheManager.SaveToDisk(); err != nil {
				log.Printf("Warning: failed to save collection cache to disk: %v", err)
			}
			m.cacheManager.SetStatus(CacheStatusReady)
		}

		return cacheSyncStatusMsg{
			status:  CacheStatusReady,
			records: records,
		}
	}
}

func (m Model) fetchPriorityRecordCmd(iid, releaseID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		if m.orgClient == nil {
			return priorityRecordResolvedMsg{
				err: fmt.Errorf("no org client initialized"),
			}
		}

		req := &pb.GetRecordRequest{
			IncludeHistory: false,
			Request: &pb.GetRecordRequest_GetRecordWithId{
				GetRecordWithId: &pb.GetRecordWithId{
					InstanceId: iid,
					ReleaseId:  releaseID,
				},
			},
		}

		resp, err := m.orgClient.GetRecord(ctx, req)
		if err != nil {
			return priorityRecordResolvedMsg{err: err}
		}
		if resp != nil && len(resp.GetRecords()) > 0 && resp.GetRecords()[0].GetRecord() != nil {
			return priorityRecordResolvedMsg{
				record: resp.GetRecords()[0].GetRecord(),
			}
		}
		return priorityRecordResolvedMsg{
			err: fmt.Errorf("record not found"),
		}
	}
}

func (m *Model) getOrFetchPriorityRecord(iid, releaseID int64) (*pb.Record, tea.Cmd) {
	if m.pendingPriorities == nil {
		m.pendingPriorities = make(map[int64]bool)
	}

	// 1. Check CacheManager
	if m.cacheManager != nil {
		if iid != 0 {
			if cr, ok := m.cacheManager.GetByInstanceID(iid); ok && cr != nil && (cr.GetArtist() != "" || cr.GetTitle() != "") {
				return cachedToRecord(cr), nil
			}
		}
		if releaseID != 0 {
			if recs := m.cacheManager.GetByReleaseID(releaseID); len(recs) > 0 && recs[0] != nil && (recs[0].GetArtist() != "" || recs[0].GetTitle() != "") {
				return cachedToRecord(recs[0]), nil
			}
		}
	}

	// 2. Check collectionIndex
	for _, rec := range m.collectionIndex {
		if rec == nil || rec.GetRelease() == nil {
			continue
		}
		if (releaseID != 0 && rec.GetRelease().GetId() == releaseID) || (iid != 0 && rec.GetRelease().GetInstanceId() == iid) {
			targetID := releaseID
			if targetID == 0 {
				targetID = iid
			}
			if !m.pendingPriorities[targetID] && (getRecordArtist(rec) != "" || getRecordTitle(rec) != "") {
				return rec, nil
			}
		}
	}

	// 3. Uncached: track in pendingPriorities, create placeholder, and dispatch fetchPriorityRecordCmd
	targetID := releaseID
	if targetID == 0 {
		targetID = iid
	}

	placeholder := &pb.Record{
		Release: &pbd.Release{
			Id:         releaseID,
			InstanceId: iid,
		},
	}

	if targetID != 0 {
		m.pendingPriorities[targetID] = true
	}
	if releaseID != 0 {
		m.pendingPriorities[releaseID] = true
	}
	if iid != 0 {
		m.pendingPriorities[iid] = true
	}

	cmd := m.fetchPriorityRecordCmd(iid, releaseID)
	return placeholder, cmd
}

func (m *Model) getOrFetchRecord(iid, releaseID int64) (*pb.Record, tea.Cmd) {
	return m.getOrFetchPriorityRecord(iid, releaseID)
}

func (m Model) Init() tea.Cmd {
	logoCmd := tea.Tick(initialLogoDuration, func(t time.Time) tea.Msg {
		return timeoutMsg{}
	})
	if m.version == "dev" || m.version == "" {
		return logoCmd
	}
	return tea.Batch(
		logoCmd,
		m.loadDiskCacheCmd(),
		checkUpdateCmd(m),
		tea.Tick(15*time.Minute, func(t time.Time) tea.Msg {
			return checkForUpdateMsg{}
		}),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if bMsg, ok := msg.(tea.BatchMsg); ok {
		var cmds []tea.Cmd
		for _, c := range bMsg {
			if c != nil {
				subMsg := c()
				newModel, subCmd := m.Update(subMsg)
				m = newModel.(Model)
				if subCmd != nil {
					cmds = append(cmds, subCmd)
				}
			}
		}
		return m, tea.Batch(cmds...)
	}

	if cMsg, ok := msg.(collectionFetchedMsg); ok {
		m.collectionLoading = false
		if cMsg.err != nil {
			m.locateSearchErr = cMsg.err.Error()
		} else {
			m.locateSearchErr = ""
			m.collectionIndex = cMsg.records
			m.buildCollectionIndex(cMsg.records)
			m.filterCollectionIndex()
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case cacheLoadedMsg:
		if msg.err != nil {
			log.Printf("Cache load warning: %v", msg.err)
			return m, nil
		}
		if len(msg.records) > 0 {
			m.collectionIndex = msg.records
			m.buildCollectionIndex(msg.records)
			m.filterCollectionIndex()
		}
		if m.cacheManager != nil {
			if m.cacheManager.IsExpired(7*24*time.Hour) || len(msg.records) == 0 {
				m.cacheStatus = CacheStatusSyncing
				m.cacheManager.SetStatus(CacheStatusSyncing)
				return m, m.syncCollectionCacheCmd()
			}
			m.cacheStatus = CacheStatusReady
			m.cacheManager.SetStatus(CacheStatusReady)
		}
		return m, nil
	case cacheSyncStatusMsg:
		m.cacheStatus = msg.status
		if m.cacheManager != nil {
			m.cacheManager.SetStatus(msg.status)
		}
		if msg.err == nil && msg.records != nil {
			m.collectionIndex = msg.records
			m.buildCollectionIndex(msg.records)
			m.filterCollectionIndex()
		}
		return m, nil
	case priorityRecordResolvedMsg:
		if msg.record != nil && msg.record.GetRelease() != nil {
			relID := msg.record.GetRelease().GetId()
			iid := msg.record.GetRelease().GetInstanceId()
			if m.pendingPriorities != nil {
				delete(m.pendingPriorities, relID)
				if iid != 0 {
					delete(m.pendingPriorities, iid)
				}
			}
			if m.cacheManager != nil {
				m.cacheManager.UpsertRecord(msg.record)
				_ = m.cacheManager.SaveToDisk()
			}
			found := false
			for i, r := range m.collectionIndex {
				if r != nil && r.GetRelease() != nil {
					if (relID != 0 && r.GetRelease().GetId() == relID) || (iid != 0 && r.GetRelease().GetInstanceId() == iid) {
						m.collectionIndex[i] = msg.record
						found = true
						break
					}
				}
			}
			if !found {
				m.collectionIndex = append(m.collectionIndex, msg.record)
			}
			m.buildCollectionIndex(m.collectionIndex)
			m.filterCollectionIndex()
		}
		return m, nil
	case checkForUpdateMsg:
		return m, tea.Batch(
			checkUpdateCmd(m),
			tea.Tick(15*time.Minute, func(t time.Time) tea.Msg {
				return checkForUpdateMsg{}
			}),
		)
	case updateAvailableMsg:
		m.isUpdating = true
		if msg.release != nil {
			m.updateStatus = fmt.Sprintf("Updating to %s...", msg.release.Version)
		} else {
			m.updateStatus = "Updating..."
		}
		return m, downloadAndRestartCmd(m, msg.release)
	case updateStatusMsg:
		m.updateStatus = msg.status
		return m, nil
	case updateErrorMsg:
		m.isUpdating = false
		m.updateStatus = "Update check failed (retrying in 15m)"
		log.Printf("Update warning: %v", msg.err)
		return m, nil
	case updateCompleteMsg:
		m.isUpdating = false
		m.updateStatus = "Update complete"
		return m, nil
	}

	switch m.state {
	case StateStartupLogo:
		switch msg.(type) {
		case tea.KeyMsg, timeoutMsg:
			if m.tokenLoader != nil {
				token, err := m.tokenLoader()
				if err == nil && token != "" {
					m.authToken = token

					if m.cacheManager != nil {
						if m.cacheManager.GetStatus() == CacheStatusUninitialized {
							_ = m.cacheManager.LoadFromDisk()
						}
						if m.collectionIndex == nil {
							cachedRecords := m.cacheManager.GetCachedRecords()
							if len(cachedRecords) > 0 {
								var recs []*pb.Record
								for _, cr := range cachedRecords {
									recs = append(recs, cachedToRecord(cr))
								}
								m.collectionIndex = recs
								m.buildCollectionIndex(recs)
								m.filterCollectionIndex()
							}
						}
					}

					hasValidCache := len(m.collectionIndex) > 0 && m.cacheManager != nil && m.cacheManager.GetStatus() != CacheStatusRebuilding

					if hasValidCache {
						m.state = StateMainApp
						if m.cacheManager.IsExpired(7 * 24 * time.Hour) {
							m.cacheStatus = CacheStatusSyncing
							m.cacheManager.SetStatus(CacheStatusSyncing)
							return m, m.syncCollectionCacheCmd()
						}
						m.cacheStatus = CacheStatusReady
						m.cacheManager.SetStatus(CacheStatusReady)
						return m, nil
					}

					m.cacheStatus = CacheStatusSyncing
					if m.cacheManager != nil {
						m.cacheManager.SetStatus(CacheStatusSyncing)
					}
					m.state = StateLoadingSync
					return m, m.pollSync()
				}
			}
			m.state = StateLogin
			return m, m.fetchURL()
		}
	case StateLogin:
		switch msg := msg.(type) {
		case urlFetchedMsg:
			m.loginURL = msg.url
			m.loginToken = msg.token
			return m, m.pollLogin()
		case urlFetchErrMsg:
			m.err = msg.err
			return m, nil
		case loginSuccessMsg:
			if m.tokenSaver != nil && msg.auth != nil {
				if err := m.tokenSaver(msg.auth.GetToken()); err != nil {
					m.err = err
					return m, nil
				}
			}
			m.loginRetryCount = 0
			if msg.auth != nil {
				m.authToken = msg.auth.GetToken()
			}
			m.state = StateLoadingSync
			return m, m.pollSync()
		case loginErrMsg:
			m.loginRetryCount++
			delay := time.Duration(1<<m.loginRetryCount) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			return m, tea.Tick(delay, func(t time.Time) tea.Msg {
				return loginPollMsg{}
			})
		case loginPollMsg:
			return m, m.pollLogin()
		}
	case StateLoadingSync:
		switch msg := msg.(type) {
		case syncPollMsg:
			return m, m.pollSync()
		case syncStatusMsg:
			if msg.err != nil {
				m.err = msg.err
				m.syncRetryCount++
				delay := time.Duration(1<<m.syncRetryCount) * time.Second
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
				return m, tea.Tick(delay, func(t time.Time) tea.Msg {
					return syncPollMsg{}
				})
			}
			m.err = nil
			m.syncRetryCount = 0
			m.user = msg.user
			
			if msg.expectedSize > 0 {
				m.progress = float64(msg.currentSize) / float64(msg.expectedSize)
			}
			
			if msg.userState == pb.StoredUser_USER_STATE_IN_WAITLIST {
				m.state = StateWaitlist
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return syncPollMsg{}
				})
			} else if msg.userState == pb.StoredUser_USER_STATE_LIVE {
				m.state = StateMainApp
				return m, nil
			}

			return m, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
				return syncPollMsg{}
			})
			
		case tea.WindowSizeMsg:
			m.progBar.Width = msg.Width - 4
			if m.progBar.Width > 80 {
				m.progBar.Width = 80
			}
			return m, nil
		}
	case StateWaitlist:
		switch msg := msg.(type) {
		case syncPollMsg:
			return m, m.pollSync()
		case syncStatusMsg:
			if msg.err != nil {
				m.err = msg.err
				m.syncRetryCount++
				delay := time.Duration(1<<m.syncRetryCount) * time.Second
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
				return m, tea.Tick(delay, func(t time.Time) tea.Msg {
					return syncPollMsg{}
				})
			}
			m.err = nil
			m.syncRetryCount = 0
			if msg.user != nil {
				m.user = msg.user
			}
			
			if msg.userState == pb.StoredUser_USER_STATE_LIVE {
				m.state = StateMainApp
				return m, nil
			}

			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return syncPollMsg{}
			})
		}
	case StateMainApp:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.textInput.Value() == "" && (msg.String() == "h" || msg.String() == "?") {
				m.showHelp = !m.showHelp
				return m, nil
			}

			switch msg.Type {
			case tea.KeyEnter:
				cmdStr := strings.TrimSpace(m.textInput.Value())
				m.textInput.SetValue("")
				if cmdStr == "" {
					return m, nil
				}
				if cmdStr == "q" || cmdStr == "quit" || cmdStr == "exit" {
					return m, tea.Quit
				}
				if cmdStr == "h" || cmdStr == "help" {
					m.showHelp = !m.showHelp
					return m, nil
				}
				if cmdStr == "configure org" || cmdStr == "config org" {
					m.state = StateOrgConfig
					m.orgErr = ""
					m.inlineErrMsg = ""
					m.initOrgConfigForm()
					return m, nil
				}
				if cmdStr == "configure" || cmdStr == "config" {
					m.state = StateConfigSelect
					m.orgErr = ""
					m.inlineErrMsg = ""
					m.initConfigSelectForm()
					return m, nil
				}
				return m.handleCommandInput(cmdStr)
			case tea.KeyEsc:
				if m.showHelp {
					m.showHelp = false
				}
				m.textInput.SetValue("")
				m.inlineErrMsg = ""
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	case StateOrgConfig:
		if msg, ok := msg.(setConfigMsg); ok {
			if msg.err != nil {
				m.orgErr = fmt.Sprintf("gRPC communication failure: %v", msg.err)
				return m, nil
			}
			m.state = StateMainApp
			m.form = nil
			m.orgErr = ""
			return m, nil
		}

		if m.form != nil {
			form, cmd := m.form.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.form = f
			}

			if m.form.State == huh.StateCompleted {
				if strings.TrimSpace(m.orgName) == "" {
					m.orgErr = "Error: Organization name cannot be empty"
					m.form.State = huh.StateNormal
					return m, nil
				}
				if len(m.selectedFolders) == 0 {
					m.orgErr = "Error: At least one folder placement must be selected"
					m.form.State = huh.StateNormal
					return m, nil
				}

				m.orgErr = ""
				unitsVal64, _ := strconv.ParseInt(m.spaceUnits, 10, 32)
				unitsVal := int32(unitsVal64)
				widthVal, _ := strconv.ParseFloat(m.spaceWidth, 64)

				var sortVal pb.Sort
				switch m.sortStrategy {
				case "ARTIST_YEAR":
					sortVal = pb.Sort_ARTIST_YEAR
				case "LABEL_CATNO":
					sortVal = pb.Sort_LABEL_CATNO
				case "RELEASE_YEAR":
					sortVal = pb.Sort_RELEASE_YEAR
				case "EARLIEST_RELEASE_YEAR":
					sortVal = pb.Sort_EARLIEST_RELEASE_YEAR
				case "ADDITION_DATE":
					sortVal = pb.Sort_ADDITION_DATE
				}

				var foldersets []*pb.FolderSet
				for _, folderIdStr := range m.selectedFolders {
					folderId64, _ := strconv.ParseInt(folderIdStr, 10, 32)
					folderId := int32(folderId64)
					var folderName string
					if m.user != nil {
						for _, f := range m.user.GetFolders() {
							if f.GetId() == folderId {
								folderName = f.GetName()
								break
							}
						}
					}
					foldersets = append(foldersets, &pb.FolderSet{
						Name:   folderName,
						Folder: folderId,
						Sort:   sortVal,
					})
				}

				newOrg := &pb.Organisation{
					Name:       m.orgName,
					Foldersets: foldersets,
					Spaces: []*pb.Space{
						{
							Name:  m.spaceName,
							Units: unitsVal,
							Width: float32(widthVal),
						},
					},
				}

				var currentConfig *pb.GramophileConfig
				if m.user != nil && m.user.GetConfig() != nil {
					currentConfig = m.user.GetConfig()
				} else {
					currentConfig = &pb.GramophileConfig{}
				}

				if currentConfig.OrganisationConfig == nil {
					currentConfig.OrganisationConfig = &pb.OrganisationConfig{}
				}

				currentConfig.OrganisationConfig.Organisations = append(
					currentConfig.OrganisationConfig.Organisations,
					newOrg,
				)

				m.orgName = ""
				m.spaceName = ""
				m.spaceUnits = ""
				m.spaceWidth = ""
				m.selectedFolders = nil
				m.sortStrategy = ""

				return m, m.pollSetConfig(currentConfig)
			}

			if m.form.State == huh.StateAborted {
				m.state = StateMainApp
				m.form = nil
				m.orgErr = ""
				return m, nil
			}

			return m, cmd
		}
	case StateConfigSelect:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.state = StateMainApp
				m.form = nil
				m.inlineErrMsg = ""
				return m, nil
			}
		}

		if m.form != nil {
			form, cmd := m.form.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.form = f
			}

			if m.form.State == huh.StateCompleted {
				if m.configTarget == "org" {
					m.state = StateOrgConfig
					m.orgErr = ""
					m.inlineErrMsg = ""
					m.initOrgConfigForm()
					return m, nil
				}
				m.state = StateMainApp
				m.form = nil
				return m, nil
			}

			if m.form.State == huh.StateAborted {
				m.state = StateMainApp
				m.form = nil
				m.inlineErrMsg = ""
				return m, nil
			}

			return m, cmd
		}
	case StateOrgView:
		switch msg := msg.(type) {
		case orgFetchedMsg:
			if msg.err != nil {
				m.inlineErrMsg = msg.err.Error()
				return m, nil
			}
			m.orgSnapshot = msg.snapshot
			if msg.snapshot != nil {
				m.orgPlacements = msg.snapshot.GetPlacements()
			}
			m.resolvedRecords = make(map[int64]*pb.Record)
			m.renderOrgViewport()
			var cmds []tea.Cmd
			for _, p := range m.orgPlacements {
				if p.GetIid() > 0 {
					cmds = append(cmds, m.fetchRecordCmd(p.GetIid()))
				}
			}
			if len(cmds) > 0 {
				cmds = append(cmds, m.orgSpinner.Tick)
			}
			return m, tea.Batch(cmds...)
		case recordFetchedMsg:
			if m.resolvedRecords == nil {
				m.resolvedRecords = make(map[int64]*pb.Record)
			}
			if msg.err == nil && msg.record != nil {
				m.resolvedRecords[msg.iid] = msg.record
			} else {
				m.resolvedRecords[msg.iid] = &pb.Record{}
			}
			m.renderOrgViewport()
			return m, nil
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.orgSpinner, cmd = m.orgSpinner.Update(msg)
			m.renderOrgViewport()
			if !m.hasUnresolvedOrgRecords() {
				return m, nil
			}
			return m, cmd
		case tea.KeyMsg:
			switch msg.String() {
			case "x", "q", "esc":
				m.state = StateMainApp
				return m, nil
			case "up", "k":
				m.orgViewport.LineUp(1)
				return m, nil
			case "down", "j":
				m.orgViewport.LineDown(1)
				return m, nil
			case "pgup":
				m.orgViewport.HalfViewUp()
				return m, nil
			case "pgdown":
				m.orgViewport.HalfViewDown()
				return m, nil
			}
		}
	case StateLocateView:
		switch msg := msg.(type) {
		case locateFetchedMsg:
			if msg.err != nil {
				m.inlineErrMsg = msg.err.Error()
				return m, nil
			}
			m.locateResponse = msg.response
			m.inlineErrMsg = ""
			content := formatLocateOutput(msg.response)
			if m.locateViewport.Width == 0 {
				m.locateViewport = viewport.New(80, 20)
			}
			m.locateViewport.SetContent(content)
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "x", "q", "esc":
				m.state = StateMainApp
				return m, nil
			case "up", "k":
				m.locateViewport.LineUp(1)
				return m, nil
			case "down", "j":
				m.locateViewport.LineDown(1)
				return m, nil
			case "pgup":
				m.locateViewport.HalfViewUp()
				return m, nil
			case "pgdown":
				m.locateViewport.HalfViewDown()
				return m, nil
			}
		}
	case StateLocateSearch:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.locateSearchInput.SetValue("")
				m.locateSearchCursor = 0
				m.locateViewportOffset = 0
				m.filteredReleases = m.indexedReleases
				m.filteredLocateRecords = m.collectionIndex
				m.state = StateMainApp
				return m, nil
			case "up", "k":
				if m.locateSearchCursor > 0 {
					m.locateSearchCursor--
				}
				if m.locateSearchCursor < m.locateViewportOffset {
					m.locateViewportOffset--
				}
				m.clampLocateViewportOffset()
				return m, nil
			case "down", "j":
				maxLen := len(m.filteredReleases)
				if maxLen == 0 {
					maxLen = len(m.filteredLocateRecords)
				}
				if m.locateSearchCursor < maxLen-1 {
					m.locateSearchCursor++
				}
				if m.locateSearchCursor > m.locateViewportOffset+9 {
					m.locateViewportOffset++
				}
				m.clampLocateViewportOffset()
				return m, nil
			case "enter":
				if len(m.filteredReleases) == 0 && len(m.filteredLocateRecords) > 0 {
					if m.locateSearchCursor >= len(m.filteredLocateRecords) {
						m.locateSearchCursor = len(m.filteredLocateRecords) - 1
					}
					if m.locateSearchCursor < 0 {
						m.locateSearchCursor = 0
					}
					rec := m.filteredLocateRecords[m.locateSearchCursor]
					releaseID := rec.GetRelease().GetId()
					m.activeLocateID = releaseID
					m.state = StateLocateView
					m.locateViewport.SetContent("Loading location...")
					m.locateViewport.GotoTop()
					return m, m.fetchLocateCmd(releaseID)
				}
				if len(m.filteredReleases) == 0 {
					return m, nil
				}
				if m.locateSearchCursor >= len(m.filteredReleases) {
					m.locateSearchCursor = len(m.filteredReleases) - 1
				}
				if m.locateSearchCursor < 0 {
					m.locateSearchCursor = 0
				}
				choice := m.filteredReleases[m.locateSearchCursor]
				if len(choice.records) == 1 {
					m.activeLocateID = choice.releaseID
					m.state = StateLocateView
					m.locateViewport.SetContent("Loading location...")
					m.locateViewport.GotoTop()
					return m, m.fetchLocateCmd(choice.releaseID)
				}
				m.activeReleaseChoice = choice
				m.versionSelectCursor = 0
				m.state = StateLocateVersionSelect
				return m, nil
			default:
				oldVal := m.locateSearchInput.Value()
				var cmd tea.Cmd
				m.locateSearchInput, cmd = m.locateSearchInput.Update(msg)
				newVal := m.locateSearchInput.Value()
				if newVal != oldVal {
					m.locateSearchCursor = 0
					m.filterCollectionIndex()
					if m.priorityFetchCmd != nil {
						pCmd := m.priorityFetchCmd
						m.priorityFetchCmd = nil
						if cmd != nil {
							cmd = tea.Batch(cmd, pCmd)
						} else {
							cmd = pCmd
						}
					}
				}
				return m, cmd
			}
		}
	case StateLocateVersionSelect:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.state = StateLocateSearch
				return m, nil
			case "up", "k":
				if m.versionSelectCursor > 0 {
					m.versionSelectCursor--
				}
				return m, nil
			case "down", "j":
				if m.activeReleaseChoice != nil && m.versionSelectCursor < len(m.activeReleaseChoice.records)-1 {
					m.versionSelectCursor++
				}
				return m, nil
			case "enter":
				if m.activeReleaseChoice == nil || len(m.activeReleaseChoice.records) == 0 {
					return m, nil
				}
				if m.versionSelectCursor >= len(m.activeReleaseChoice.records) {
					m.versionSelectCursor = len(m.activeReleaseChoice.records) - 1
				}
				if m.versionSelectCursor < 0 {
					m.versionSelectCursor = 0
				}
				choice := m.activeReleaseChoice
				_ = choice.records[m.versionSelectCursor]
				m.activeLocateID = choice.releaseID
				m.state = StateLocateView
				m.locateViewport.SetContent("Loading location...")
				m.locateViewport.GotoTop()
				return m, m.fetchLocateCmd(choice.releaseID)
			}
		}
	}

	// Handle global quit
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || (m.state != StateOrgView && m.state != StateLocateView && m.state != StateMainApp && m.state != StateOrgConfig && m.state != StateConfigSelect && m.state != StateLocateSearch && m.state != StateLocateVersionSelect && msg.String() == "q") {
			return m, tea.Quit
		}
	}

	return m, nil
}

const logo = ` ██████╗ ██████╗  █████╗ ███╗   ███╗ ██████╗ ██████╗ ██╗  ██╗██╗██╗     ███████╗
██╔════╝ ██╔══██╗██╔══██╗████╗ ████║██╔═══██╗██╔══██╗██║  ██║██║██║     ██╔════╝
██║  ███╗██████╔╝███████║██╔████╔██║██║   ██║██████╔╝███████║██║██║     █████╗  
██║   ██║██╔══██╗██╔══██║██║╚██╔╝██║██║   ██║██╔═══╝ ██╔══██║██║██║     ██╔══╝  
╚██████╔╝██║  ██║██║  ██║██║ ╚═╝ ██║╚██████╔╝██║     ██║  ██║██║███████╗███████╗
 ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝ ╚═════╝ ╚═╝     ╚═╝  ╚═╝╚═╝╚══════╝╚══════╝`

func (m Model) renderLogo() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Margin(1, 2)

	return style.Render(logo)
}

func (m Model) View() string {
	var body string
	switch m.state {
	case StateStartupLogo:
		body = "Press any key to continue..."
	case StateLogin:
		if m.err != nil {
			body = fmt.Sprintf("Error: %v\nPress q to quit", m.err)
		} else if m.loginURL == "" {
			body = "Fetching authentication URL..."
		} else {
			body = fmt.Sprintf("Please log in by visiting:\n\n  %s\n\nWaiting for authentication...", m.loginURL)
		}
	case StateLoadingSync:
		if m.err != nil {
			body = fmt.Sprintf("Error fetching sync state: %v\n\nReconnecting...", m.err)
		} else {
			body = fmt.Sprintf("Syncing Collection with Discogs...\n\n%s\n", m.progBar.ViewAs(m.progress))
		}
	case StateWaitlist:
		if m.err != nil {
			body = fmt.Sprintf("Error polling waitlist status: %v\n\nReconnecting...", m.err)
		} else {
			style := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFD700")).
				Padding(1, 2)

			waitMsg := fmt.Sprintf("Waiting for Admin Approval (%s)...", m.user.GetState())
			if m.user != nil && m.user.GetUser() != nil && m.user.GetUser().GetUsername() != "" {
				waitMsg = fmt.Sprintf("Waiting for Admin Approval for %s (%s)...", m.user.GetUser().GetUsername(), m.user.GetState())
			}
			body = "Sync Complete!\n\n" + style.Render(waitMsg) + "\n"
		}
	case StateMainApp:
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		promptStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))

		var sb strings.Builder
		if m.showHelp {
			sb.WriteString("Commands:\n")
			sb.WriteString("  locate <release_id>   Locate a record in your organization\n")
			sb.WriteString("  org [name]            View organization layout and placements\n")
			sb.WriteString("  configure             Configure organizations\n")
			sb.WriteString("  quit                  Exit the application\n\n")
		}
		sb.WriteString(promptStyle.Render("Command: ") + m.textInput.View() + "\n")

		if m.inlineErrMsg != "" {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
			sb.WriteString("\n" + errStyle.Render(m.inlineErrMsg) + "\n")
		}

		if m.showHelp {
			sb.WriteString("\n" + helpStyle.Render("press h to hide help"))
		} else {
			sb.WriteString("\n" + helpStyle.Render("press h for help"))
		}
		body = sb.String()
	case StateConfigSelect:
		if m.form != nil {
			body = m.form.View()
		} else {
			body = "Loading configuration options..."
		}
	case StateOrgConfig:
		if m.form != nil {
			body = m.form.View()
		} else {
			body = "Loading wizard..."
		}
		if m.orgErr != "" {
			errStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Bold(true)
			body += "\n\n" + errStyle.Render(m.orgErr)
		} else if m.err != nil {
			errStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Bold(true)
			body += "\n\n" + errStyle.Render(fmt.Sprintf("Error: %v", m.err))
		}
	case StateOrgView:
		if m.inlineErrMsg != "" {
			body = fmt.Sprintf("Error: %s\n\nPress any key to return...", m.inlineErrMsg)
		} else {
			hash := m.activeHash
			if hash == "" && m.orgSnapshot != nil {
				hash = m.orgSnapshot.GetHash()
			}
			if hash != "" {
				body = fmt.Sprintf("%s\n\n%s", hash, m.orgViewport.View())
			} else {
				body = m.orgViewport.View()
			}
		}
	case StateLocateView:
		if m.inlineErrMsg != "" {
			body = fmt.Sprintf("Error: %s\n\nPress any key to return...", m.inlineErrMsg)
		} else {
			body = m.locateViewport.View()
		}
	case StateLocateSearch:
		var sb strings.Builder
		promptStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
		footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
		indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

		header := promptStyle.Render("Search: ") + m.locateSearchInput.View()
		if m.collectionLoading {
			header += " " + m.getOrgSpinnerView()
		}
		sb.WriteString(header + "\n\n")

		if m.locateSearchErr != "" {
			sb.WriteString(fmt.Sprintf("Error loading collection: %s\n\n", m.locateSearchErr))
		}

		if len(m.filteredReleases) > 0 {
			total := len(m.filteredReleases)
			offset := m.locateViewportOffset
			if offset < 0 {
				offset = 0
			}
			maxOffset := total - 10
			if maxOffset < 0 {
				maxOffset = 0
			}
			if offset > maxOffset {
				offset = maxOffset
			}
			end := offset + 10
			if end > total {
				end = total
			}

			for i := offset; i < end; i++ {
				entry := m.filteredReleases[i]
				label := fmt.Sprintf("%s - %s", entry.artist, entry.title)
				if i == m.locateSearchCursor {
					sb.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", label)) + "\n")
				} else {
					sb.WriteString(fmt.Sprintf("  %s\n", label))
				}
			}

			sb.WriteString("\n" + indicatorStyle.Render(fmt.Sprintf("[Showing %d-%d of %s records]", offset+1, end, formatNumberWithCommas(total))) + "\n")
		} else if len(m.filteredLocateRecords) > 0 {
			dupPool := m.filteredLocateRecords
			if len(m.collectionIndex) > len(dupPool) {
				dupPool = m.collectionIndex
			}
			dupCounts := make(map[string]int)
			for _, r := range dupPool {
				k := strings.ToLower(getRecordArtist(r)) + "|" + strings.ToLower(getRecordTitle(r))
				dupCounts[k]++
			}

			for i, rec := range m.filteredLocateRecords {
				k := strings.ToLower(getRecordArtist(rec)) + "|" + strings.ToLower(getRecordTitle(rec))
				isDup := dupCounts[k] > 1
				label := m.formatLocateRecordLabel(rec, isDup)
				if i == m.locateSearchCursor {
					sb.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", label)) + "\n")
				} else {
					sb.WriteString(fmt.Sprintf("  %s\n", label))
				}
			}
		} else {
			sb.WriteString("No matching records found.\n")
		}

		sb.WriteString("\n" + footerStyle.Render("↑/↓: Navigate • Enter: Locate • Esc: Cancel"))
		body = sb.String()
	case StateLocateVersionSelect:
		var sb strings.Builder
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
		footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))

		var artist, title string
		if m.activeReleaseChoice != nil {
			artist = m.activeReleaseChoice.artist
			title = m.activeReleaseChoice.title
		}
		sb.WriteString(headerStyle.Render(fmt.Sprintf("Select copy for: %s - %s", artist, title)) + "\n\n")

		if m.activeReleaseChoice != nil {
			for i, rec := range m.activeReleaseChoice.records {
				rowStr := m.formatVersionRow(rec)
				if i == m.versionSelectCursor {
					sb.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", rowStr)) + "\n")
				} else {
					sb.WriteString(fmt.Sprintf("  %s\n", rowStr))
				}
			}
		}

		sb.WriteString("\n" + footerStyle.Render("↑/↓: Navigate • Enter: Locate Copy • Esc: Back to Search"))
		body = sb.String()
	default:
		body = "Gramophile TUI"
	}

	return m.renderLogo() + "\n\n" + body + "\n\n" + renderFooter(m)
}

func renderFooter(m Model) string {
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	if m.version == "dev" || m.version == "vdev" || m.version == "" {
		return footerStyle.Render("vdev (auto-update disabled)")
	}

	v := m.version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	if m.updateStatus != "" {
		if strings.HasPrefix(m.updateStatus, v+" | ") {
			return footerStyle.Render(m.updateStatus)
		}
		return footerStyle.Render(fmt.Sprintf("%s | %s", v, m.updateStatus))
	}

	return footerStyle.Render(fmt.Sprintf("Gramophile %s", v))
}

func (m Model) renderFooter() string {
	return renderFooter(m)
}



func (m *Model) initConfigSelectForm() {
	m.configTarget = "org"
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Configuration Target").
				Options(
					huh.NewOption("org", "org"),
				).
				Value(&m.configTarget),
		),
	)
	m.form.Init()
}

func (m *Model) initOrgConfigForm() {
	var folderOptions []huh.Option[string]
	if m.user != nil {
		for _, f := range m.user.GetFolders() {
			folderOptions = append(folderOptions, huh.NewOption(f.GetName(), fmt.Sprintf("%d", f.GetId())))
		}
	}

	sortOptions := []huh.Option[string]{
		huh.NewOption("Artist, Year", "ARTIST_YEAR"),
		huh.NewOption("Label, Catalog Number", "LABEL_CATNO"),
		huh.NewOption("Release Year", "RELEASE_YEAR"),
		huh.NewOption("Earliest Release Year", "EARLIEST_RELEASE_YEAR"),
		huh.NewOption("Addition Date", "ADDITION_DATE"),
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Organization Name").
				Value(&m.orgName),
			huh.NewInput().
				Title("Space/Shelf Name").
				Value(&m.spaceName),
			huh.NewInput().
				Title("Number of Units").
				Value(&m.spaceUnits).
				Validate(func(str string) error {
					val, err := strconv.ParseInt(str, 10, 32)
					if err != nil || val <= 0 {
						return fmt.Errorf("must be a positive integer")
					}
					return nil
				}),
			huh.NewInput().
				Title("Unit Width").
				Value(&m.spaceWidth).
				Validate(func(str string) error {
					val, err := strconv.ParseFloat(str, 64)
					if err != nil || val <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),
			huh.NewMultiSelect[string]().
				Title("Map Folders").
				Options(folderOptions...).
				Value(&m.selectedFolders),
			huh.NewSelect[string]().
				Title("Sorting Strategy").
				Options(sortOptions...).
				Value(&m.sortStrategy),
		),
	)
	m.form.Init()
}

func (m Model) pollSetConfig(config *pb.GramophileConfig) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		_, err := m.client.SetConfig(ctx, &pb.SetConfigRequest{Config: config})
		return setConfigMsg{err: err}
	}
}

// parseOrgCommand parses command string input for org / orgview commands, supporting positional organization name (e.g. org 12 Inches) and optional flags (--org, --slot, --hash, --debug).
func parseOrgCommand(input string) (string, int32, string, bool, error) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return "", 0, "", false, fmt.Errorf("empty command")
	}
	cmd := fields[0]
	if cmd != "org" && cmd != "orgview" {
		return "", 0, "", false, fmt.Errorf("unknown command: %s", cmd)
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	var orgName string
	var slot int
	var hash string
	var debug bool

	fs.StringVar(&orgName, "org", "", "organization name")
	fs.IntVar(&slot, "slot", 0, "slot number")
	fs.StringVar(&hash, "hash", "", "snapshot hash")
	fs.BoolVar(&debug, "debug", false, "debug mode")

	var flagArgs []string
	var posArgs []string
	args := fields[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && arg != "--debug" && arg != "-debug" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	err := fs.Parse(flagArgs)
	if err != nil {
		return "", 0, "", false, err
	}

	if orgName == "" && len(posArgs) > 0 {
		orgName = strings.Join(posArgs, " ")
	} else if orgName != "" && len(posArgs) > 0 {
		orgName = orgName + " " + strings.Join(posArgs, " ")
	}
	orgName = strings.Trim(orgName, "\"'")

	return orgName, int32(slot), hash, debug, nil
}

func (m Model) fetchOrgCmd(orgName, hash string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		if m.orgClient == nil {
			return orgFetchedMsg{err: fmt.Errorf("no org client initialized")}
		}
		resp, err := m.orgClient.GetOrg(ctx, &pb.GetOrgRequest{
			OrgName: orgName,
		})
		if err != nil {
			return orgFetchedMsg{err: err}
		}
		return orgFetchedMsg{snapshot: resp.GetSnapshot()}
	}
}

func (m Model) fetchRecordCmd(iid int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		if m.orgClient == nil {
			return recordFetchedMsg{iid: iid, err: fmt.Errorf("no org client initialized")}
		}
		resp, err := m.orgClient.GetRecord(ctx, &pb.GetRecordRequest{
			IncludeHistory: false,
			Request: &pb.GetRecordRequest_GetRecordWithId{
				GetRecordWithId: &pb.GetRecordWithId{
					InstanceId: iid,
				},
			},
		})
		if err != nil {
			return recordFetchedMsg{iid: iid, err: err}
		}
		if len(resp.GetRecords()) > 0 {
			return recordFetchedMsg{iid: iid, record: resp.GetRecords()[0].GetRecord()}
		}
		return recordFetchedMsg{iid: iid, err: fmt.Errorf("record not found")}
	}
}

func (m *Model) getOrgSpinnerView() string {
	if len(m.orgSpinner.Spinner.Frames) == 0 {
		m.orgSpinner = newOrgSpinner()
	}
	return m.orgSpinner.View()
}

func (m Model) hasUnresolvedOrgRecords() bool {
	if m.state != StateOrgView || len(m.orgPlacements) == 0 {
		return false
	}
	for _, p := range m.orgPlacements {
		if p.GetIid() > 0 {
			if m.resolvedRecords == nil || m.resolvedRecords[p.GetIid()] == nil {
				return true
			}
		}
	}
	return false
}

func (m *Model) renderOrgViewport() {
	var sb strings.Builder

	var sumWidth float32
	for _, p := range m.orgPlacements {
		sumWidth += p.GetWidth()
	}
	m.totalWidth = int32(sumWidth)

	if len(m.orgPlacements) == 0 {
		sb.WriteString("No placements found in snapshot.\n")
	} else {
		for i, p := range m.orgPlacements {
			iid := p.GetIid()
			var titleStr string
			if m.resolvedRecords != nil {
				if rec, ok := m.resolvedRecords[iid]; ok && rec != nil {
					artist := ""
					title := ""
					if rec.GetRelease() != nil {
						if len(rec.GetRelease().GetArtists()) > 0 {
							artist = rec.GetRelease().GetArtists()[0].GetName()
						}
						title = rec.GetRelease().GetTitle()
					}
					if artist != "" && title != "" {
						titleStr = fmt.Sprintf("%s - %s", artist, title)
					} else if title != "" {
						titleStr = title
					} else if artist != "" {
						titleStr = artist
					} else {
						titleStr = fmt.Sprintf("Release #%d", iid)
					}
				} else {
					titleStr = m.getOrgSpinnerView()
				}
			} else {
				titleStr = m.getOrgSpinnerView()
			}

			idx := p.GetIndex()
			if idx == 0 {
				idx = int32(i + 1)
			}
			sb.WriteString(fmt.Sprintf("[%d] %s [ %s / %d]\n",
				idx, titleStr, p.GetSpace(), p.GetUnit()))
		}
	}

	if m.orgViewport.Width == 0 {
		m.orgViewport = viewport.New(80, 20)
	}
	m.orgViewport.SetContent(sb.String())
}

// parseLocateCommand parses command string input for locate commands supporting locate, locate <release_id> and locate --id <release_id>.
func parseLocateCommand(cmdStr string) (int64, bool, error) {
	fields := strings.Fields(strings.TrimSpace(cmdStr))
	usageErr := fmt.Errorf("Invalid release ID format. Usage: locate <release_id> or locate --id <release_id>")
	if len(fields) == 0 {
		return 0, false, usageErr
	}
	if fields[0] != "locate" {
		return 0, false, fmt.Errorf("unknown command: %s", fields[0])
	}
	if len(fields) == 1 {
		return 0, true, nil
	}

	fs := flag.NewFlagSet("locate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var releaseID int64
	fs.Int64Var(&releaseID, "id", 0, "release id")

	var flagArgs []string
	var posArgs []string
	args := fields[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	err := fs.Parse(flagArgs)
	if err != nil {
		return 0, false, usageErr
	}

	if len(posArgs) > 1 {
		return 0, false, usageErr
	}

	if releaseID <= 0 && len(posArgs) == 1 {
		parsed, parseErr := strconv.ParseInt(posArgs[0], 10, 64)
		if parseErr == nil && parsed > 0 {
			releaseID = parsed
		}
	}

	if releaseID <= 0 {
		return 0, false, usageErr
	}

	return releaseID, false, nil
}

func (m Model) fetchLocateCmd(releaseID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(10 * time.Second)
		defer cancel()
		if m.locateClient == nil {
			return locateFetchedMsg{releaseID: releaseID, err: fmt.Errorf("no locate client initialized")}
		}
		resp, err := m.locateClient.LocateRecord(ctx, &pb.LocateRecordRequest{
			ReleaseId: releaseID,
		})
		if err != nil {
			return locateFetchedMsg{releaseID: releaseID, err: err}
		}
		return locateFetchedMsg{releaseID: releaseID, response: resp}
	}
}

func (m Model) fetchCollectionIndexCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.buildContext(30 * time.Second)
		defer cancel()
		if m.orgClient == nil {
			return collectionFetchedMsg{err: fmt.Errorf("no org client initialized")}
		}
		resp, err := m.orgClient.GetRecord(ctx, &pb.GetRecordRequest{
			Request: &pb.GetRecordRequest_GetAllRecords{
				GetAllRecords: true,
			},
		})
		if err != nil {
			return collectionFetchedMsg{err: err}
		}
		var records []*pb.Record
		if resp != nil {
			for _, r := range resp.GetRecords() {
				if r != nil && r.GetRecord() != nil {
					records = append(records, r.GetRecord())
				}
			}
		}
		return collectionFetchedMsg{records: records}
	}
}

// handleCommandInput parses the command string and transitions state to StateOrgView or StateLocateView.
func (m Model) handleCommandInput(cmdStr string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(strings.TrimSpace(cmdStr))
	if len(fields) > 0 && fields[0] == "locate" {
		releaseID, isSearch, err := parseLocateCommand(cmdStr)
		if err != nil {
			m.inlineErrMsg = err.Error()
			return m, nil
		}
		if isSearch {
			m.commandInput = cmdStr
			m.state = StateLocateSearch
			m.inlineErrMsg = ""
			m.locateSearchInput.SetValue("")
			m.locateSearchInput.Focus()
			m.locateSearchInput.Placeholder = "Search collection by artist or title..."
			m.locateSearchCursor = 0
			m.locateViewportOffset = 0
			if m.collectionIndex == nil && !m.collectionLoading {
				m.collectionLoading = true
				return m, m.fetchCollectionIndexCmd()
			}
			m.filteredLocateRecords = m.collectionIndex
			m.filteredReleases = m.indexedReleases
			return m, nil
		}
		m.commandInput = cmdStr
		m.activeLocateID = releaseID
		m.inlineErrMsg = ""
		m.state = StateLocateView
		m.locateResponse = nil
		m.locateViewport = viewport.New(80, 20)
		return m, m.fetchLocateCmd(releaseID)
	}

	if len(fields) > 0 && (fields[0] == "configure" || fields[0] == "config") {
		m.inlineErrMsg = fmt.Sprintf("Unknown configuration target: %s. Usage: configure [org]", strings.Join(fields[1:], " "))
		return m, nil
	}

	orgName, slot, hash, debug, err := parseOrgCommand(cmdStr)
	if err != nil {
		m.inlineErrMsg = err.Error()
		return m, nil
	}

	if orgName == "" {
		if m.user != nil && m.user.GetConfig() != nil && m.user.GetConfig().GetOrganisationConfig() != nil && len(m.user.GetConfig().GetOrganisationConfig().GetOrganisations()) > 0 {
			orgName = m.user.GetConfig().GetOrganisationConfig().GetOrganisations()[0].GetName()
		} else {
			m.inlineErrMsg = "No organization specified. Usage: org [name]"
			return m, nil
		}
	}

	m.commandInput = cmdStr
	m.activeOrgName = orgName
	m.activeSlot = slot
	m.activeHash = hash
	m.activeDebug = debug
	m.inlineErrMsg = ""
	m.state = StateOrgView
	m.orgSnapshot = nil
	m.orgPlacements = nil
	m.resolvedRecords = nil
	m.orgViewport = viewport.New(80, 20)
	m.orgSpinner = newOrgSpinner()
	return m, m.fetchOrgCmd(orgName, hash)
}

func calculatePercentage(beforeCount, afterCount int) float64 {
	total := beforeCount + afterCount + 1
	targetIndex := beforeCount + 1
	return float64(targetIndex) / float64(total) * 100.0
}

func formatLocateOutput(res *pb.LocateRecordResponse) string {
	if res == nil || len(res.GetLocations()) == 0 {
		return "No location found for Release ID."
	}

	var sb strings.Builder
	for i, location := range res.GetLocations() {
		if location == nil {
			continue
		}
		if i > 0 {
			sb.WriteString("\n")
		}
		percentage := calculatePercentage(len(location.GetBefore()), len(location.GetAfter()))
		sb.WriteString(fmt.Sprintf("%v is in %v, Slot %v (%.0f%%):\n\n", location.GetRecord(), location.GetLocationName(), location.GetSlot(), percentage))

		for j := len(location.GetBefore()) - 1; j >= 0; j-- {
			b := location.GetBefore()[j]
			sb.WriteString(fmt.Sprintf("%v\n", b.GetRecord()))
		}

		sb.WriteString(lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%v", location.GetRecord())) + "\n")

		for _, a := range location.GetAfter() {
			sb.WriteString(fmt.Sprintf("%v\n", a.GetRecord()))
		}
	}
	return sb.String()
}

func getRecordArtist(rec *pb.Record) string {
	if rec == nil || rec.GetRelease() == nil {
		return ""
	}
	if len(rec.GetRelease().GetArtists()) > 0 {
		var names []string
		for _, a := range rec.GetRelease().GetArtists() {
			if a.GetName() != "" {
				names = append(names, a.GetName())
			}
		}
		if len(names) > 0 {
			return strings.Join(names, ", ")
		}
	}
	return ""
}

func getRecordTitle(rec *pb.Record) string {
	if rec == nil || rec.GetRelease() == nil {
		return ""
	}
	return rec.GetRelease().GetTitle()
}

func (m Model) resolveRecordFolder(rec *pb.Record) string {
	if rec == nil {
		return ""
	}
	folderID := rec.GetRelease().GetFolderId()
	if folderID != 0 && m.user != nil {
		for _, f := range m.user.GetFolders() {
			if f.GetId() == folderID && f.GetName() != "" {
				return f.GetName()
			}
		}
	}
	return rec.GetGoalFolder()
}

func (m Model) formatLocateRecordLabel(rec *pb.Record, isDuplicate bool) string {
	if rec != nil && rec.GetRelease() != nil {
		relID := rec.GetRelease().GetId()
		iid := rec.GetRelease().GetInstanceId()
		if (relID != 0 && m.pendingPriorities[relID]) || (iid != 0 && m.pendingPriorities[iid]) {
			id := relID
			if id == 0 {
				id = iid
			}
			return fmt.Sprintf("Loading Release #%d... [Fetching]", id)
		}
	}
	artist := getRecordArtist(rec)
	title := getRecordTitle(rec)
	var base string
	if artist != "" && title != "" {
		base = fmt.Sprintf("%s - %s", artist, title)
	} else if title != "" {
		base = title
	} else if artist != "" {
		base = artist
	} else {
		base = fmt.Sprintf("%d", rec.GetRelease().GetId())
	}
	if isDuplicate {
		folderName := m.resolveRecordFolder(rec)
		return fmt.Sprintf("%s [ID: %d | Location: %s]", base, rec.GetRelease().GetId(), folderName)
	}
	return base
}

func formatRecordFormat(rec *pb.Record) string {
	if rec == nil || rec.GetRelease() == nil || len(rec.GetRelease().GetFormats()) == 0 {
		return "Unknown Format"
	}
	f := rec.GetRelease().GetFormats()[0]
	name := strings.TrimSpace(f.GetName())
	if name == "" {
		return "Unknown Format"
	}
	if len(f.GetDescriptions()) > 0 && strings.TrimSpace(f.GetDescriptions()[0]) != "" {
		return fmt.Sprintf("%s, %s", name, strings.TrimSpace(f.GetDescriptions()[0]))
	}
	return name
}

func (m Model) formatRecordShelfLocation(rec *pb.Record) string {
	folder := m.resolveRecordFolder(rec)
	if folder == "" {
		return "Unassigned Shelf"
	}
	return folder
}

func (m Model) formatVersionRow(rec *pb.Record) string {
	var iid int64
	if rec != nil && rec.GetRelease() != nil {
		iid = rec.GetRelease().GetInstanceId()
	}
	format := formatRecordFormat(rec)
	shelf := m.formatRecordShelfLocation(rec)
	return fmt.Sprintf("%d - %s - %s", iid, format, shelf)
}

func (m *Model) buildCollectionIndex(records []*pb.Record) {
	releaseMap := make(map[int64]*releaseEntry)
	var indexed []*releaseEntry

	for _, rec := range records {
		if rec == nil || rec.GetRelease() == nil {
			continue
		}
		relID := rec.GetRelease().GetId()
		if entry, ok := releaseMap[relID]; ok {
			entry.records = append(entry.records, rec)
		} else {
			artist := getRecordArtist(rec)
			title := getRecordTitle(rec)
			searchKey := strings.ToLower(fmt.Sprintf("%s %s", artist, title))
			entry := &releaseEntry{
				releaseID: relID,
				artist:    artist,
				title:     title,
				searchKey: searchKey,
				records:   []*pb.Record{rec},
			}
			releaseMap[relID] = entry
			indexed = append(indexed, entry)
		}
	}

	sort.SliceStable(indexed, func(i, j int) bool {
		artistI := strings.ToLower(indexed[i].artist)
		artistJ := strings.ToLower(indexed[j].artist)
		if artistI != artistJ {
			return artistI < artistJ
		}
		titleI := strings.ToLower(indexed[i].title)
		titleJ := strings.ToLower(indexed[j].title)
		return titleI < titleJ
	})

	m.indexedReleases = indexed
	m.filteredReleases = indexed
	m.locateViewportOffset = 0
}

func (m *Model) filterCollectionIndex() {
	if len(m.indexedReleases) == 0 && len(m.collectionIndex) > 0 {
		m.buildCollectionIndex(m.collectionIndex)
	}

	m.locateSearchCursor = 0
	m.locateViewportOffset = 0

	rawQuery := strings.TrimSpace(strings.ToLower(m.locateSearchInput.Value()))
	if rawQuery == "" {
		m.filteredReleases = m.indexedReleases
		m.filteredLocateRecords = m.collectionIndex
		return
	}

	tokens := strings.Fields(rawQuery)
	var matched []*releaseEntry
	for _, entry := range m.indexedReleases {
		match := true
		for _, token := range tokens {
			if !strings.Contains(entry.searchKey, token) {
				match = false
				break
			}
		}
		if match {
			matched = append(matched, entry)
		}
	}
	m.filteredReleases = matched

	var filtered []*pb.Record
	for _, rec := range m.collectionIndex {
		artist := strings.ToLower(getRecordArtist(rec))
		title := strings.ToLower(getRecordTitle(rec))
		relIDStr := ""
		iidStr := ""
		if rec.GetRelease() != nil {
			relIDStr = fmt.Sprintf("%d", rec.GetRelease().GetId())
			iidStr = fmt.Sprintf("%d", rec.GetRelease().GetInstanceId())
		}
		searchStr := artist + " " + title + " " + relIDStr + " " + iidStr
		match := true
		for _, token := range tokens {
			if !strings.Contains(searchStr, token) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, rec)
		}
	}

	cleanQuery := strings.TrimPrefix(rawQuery, "#")
	cleanQuery = strings.TrimPrefix(cleanQuery, "id:")
	cleanQuery = strings.TrimPrefix(cleanQuery, "release:")
	cleanQuery = strings.TrimSpace(cleanQuery)
	parsedID, parseErr := strconv.ParseInt(cleanQuery, 10, 64)

	if parseErr == nil && parsedID > 0 {
		found := false
		for _, r := range filtered {
			if r != nil && r.GetRelease() != nil && (r.GetRelease().GetId() == parsedID || r.GetRelease().GetInstanceId() == parsedID) {
				found = true
				break
			}
		}
		if !found {
			placeholder, cmd := m.getOrFetchPriorityRecord(0, parsedID)
			if placeholder != nil {
				filtered = append(filtered, placeholder)
			}
			m.priorityFetchCmd = cmd
		}
	}

	m.filteredLocateRecords = filtered
}

func (m *Model) clampLocateViewportOffset() {
	if m.locateSearchCursor > m.locateViewportOffset+9 {
		m.locateViewportOffset = m.locateSearchCursor - 9
	}
	if m.locateSearchCursor < m.locateViewportOffset {
		m.locateViewportOffset = m.locateSearchCursor
	}
	total := len(m.filteredReleases)
	if total == 0 {
		total = len(m.filteredLocateRecords)
	}
	maxOffset := total - 10
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.locateViewportOffset > maxOffset {
		m.locateViewportOffset = maxOffset
	}
	if m.locateViewportOffset < 0 {
		m.locateViewportOffset = 0
	}
}

func formatNumberWithCommas(n int) string {
	in := strconv.Itoa(n)
	if len(in) <= 3 {
		return in
	}
	var out []byte
	rem := len(in) % 3
	if rem > 0 {
		out = append(out, in[:rem]...)
		if len(in) > rem {
			out = append(out, ',')
		}
	}
	for i := rem; i < len(in); i += 3 {
		out = append(out, in[i:i+3]...)
		if i+3 < len(in) {
			out = append(out, ',')
		}
	}
	return string(out)
}



