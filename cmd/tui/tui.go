package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
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

// initialLogoDuration is the time to show the logo before auto-transitioning
const initialLogoDuration = 2 * time.Second

type Model struct {
	state           appState
	client          AuthClient
	orgClient       OrgClient
	locateClient    LocateClient
	loginURL        string
	loginToken      string
	err             error
	orgErr          string
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

	commandInput    string
	orgViewport     viewport.Model
	activeOrgName   string
	activeSlot      int32
	activeHash      string
	activeDebug     bool
	orgSnapshot     *pb.OrganisationSnapshot
	orgPlacements   []*pb.Placement
	resolvedRecords map[int64]*pb.Record
	totalWidth      int32
	inlineErrMsg    string
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

func InitialModel(client AuthClient, orgClient OrgClient, locateClient LocateClient) Model {
	return Model{
		state:        StateStartupLogo,
		client:       client,
		orgClient:    orgClient,
		locateClient: locateClient,
		tokenSaver:   defaultTokenSaver,
		progBar:      progress.New(progress.WithDefaultGradient()),
	}
}

func (m Model) fetchURL() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

func (m Model) Init() tea.Cmd {
	return tea.Tick(initialLogoDuration, func(t time.Time) tea.Msg {
		return timeoutMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateStartupLogo:
		switch msg.(type) {
		case tea.KeyMsg:
			m.state = StateLogin
			return m, m.fetchURL()
		case timeoutMsg:
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
			if msg.String() == "o" {
				m.state = StateOrgConfig
				m.orgErr = ""
				m.initOrgConfigForm()
				return m, nil
			}
		}
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
			return m, tea.Batch(cmds...)
		case recordFetchedMsg:
			if msg.err == nil && msg.record != nil {
				if m.resolvedRecords == nil {
					m.resolvedRecords = make(map[int64]*pb.Record)
				}
				m.resolvedRecords[msg.iid] = msg.record
				m.renderOrgViewport()
			}
			return m, nil
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
	}

	// Handle global quit
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || (m.state != StateOrgView && msg.String() == "q") {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case StateStartupLogo:
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(2, 4).
			Margin(1, 1)

		return style.Render("GRAMOPHILE") + "\n\nPress any key to continue..."
	case StateLogin:
		if m.err != nil {
			return fmt.Sprintf("Error: %v\nPress q to quit", m.err)
		}
		if m.loginURL == "" {
			return "Fetching authentication URL..."
		}
		return fmt.Sprintf("Please log in by visiting:\n\n  %s\n\nWaiting for authentication...", m.loginURL)
	case StateLoadingSync:
		if m.err != nil {
			return fmt.Sprintf("Error fetching sync state: %v\n\nReconnecting...", m.err)
		}
		return fmt.Sprintf("\nSyncing Collection with Discogs...\n\n%s\n", m.progBar.ViewAs(m.progress))
	case StateWaitlist:
		if m.err != nil {
			return fmt.Sprintf("Error polling waitlist status: %v\n\nReconnecting...", m.err)
		}
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			Padding(1, 2)
		return "\nSync Complete!\n\n" + style.Render("Waiting for Admin Approval...") + "\n"
	case StateMainApp:
		return "\nHandoff to main application complete.\n"
	case StateOrgConfig:
		var view string
		if m.form != nil {
			view = m.form.View()
		} else {
			view = "Loading wizard..."
		}
		if m.orgErr != "" {
			errStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Bold(true)
			view += "\n\n" + errStyle.Render(m.orgErr)
		} else if m.err != nil {
			errStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Bold(true)
			view += "\n\n" + errStyle.Render(fmt.Sprintf("Error: %v", m.err))
		}
		return view
	case StateOrgView:
		if m.inlineErrMsg != "" {
			return fmt.Sprintf("Error: %s\n\nPress any key to return...", m.inlineErrMsg)
		}
		return fmt.Sprintf("Organization View: %s (Slot: %d, Hash: %s, Debug: %t)\n%s",
			m.activeOrgName, m.activeSlot, m.activeHash, m.activeDebug, m.orgViewport.View())
	}
	return "Gramophile TUI"
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := m.client.SetConfig(ctx, &pb.SetConfigRequest{Config: config})
		return setConfigMsg{err: err}
	}
}

// parseOrgCommand parses command string input for org / orgview commands and flags (--org, --slot, --hash, --debug).
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
			if !strings.Contains(arg, "=") && arg != "--debug" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
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
		orgName = posArgs[0]
	}

	return orgName, int32(slot), hash, debug, nil
}

func (m Model) fetchOrgCmd(orgName, hash string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

func (m *Model) renderOrgViewport() {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Organization: %s", m.activeOrgName))
	if m.activeSlot > 0 {
		sb.WriteString(fmt.Sprintf(" | Slot: %d", m.activeSlot))
	}
	if m.activeHash != "" {
		sb.WriteString(fmt.Sprintf(" | Hash: %s", m.activeHash))
	} else if m.orgSnapshot != nil && m.orgSnapshot.GetHash() != "" {
		sb.WriteString(fmt.Sprintf(" | Hash: %s", m.orgSnapshot.GetHash()))
	}
	if m.activeDebug {
		sb.WriteString(" [DEBUG]")
	}
	sb.WriteString("\n\n")

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
					titleStr = "Loading..."
				}
			} else {
				titleStr = "Loading..."
			}

			sb.WriteString(fmt.Sprintf("[%d] Space: %s | Unit: %d | Index: %d | Title: %s | Width: %.1f\n",
				i+1, p.GetSpace(), p.GetUnit(), p.GetIndex(), titleStr, p.GetWidth()))
		}
	}

	if m.orgViewport.Width == 0 {
		m.orgViewport = viewport.New(80, 20)
	}
	m.orgViewport.SetContent(sb.String())
}

// handleCommandInput parses the command string and transitions state to StateOrgView.
func (m Model) handleCommandInput(cmdStr string) (tea.Model, tea.Cmd) {
	orgName, slot, hash, debug, err := parseOrgCommand(cmdStr)
	if err != nil {
		m.inlineErrMsg = err.Error()
		return m, nil
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
	return m, m.fetchOrgCmd(orgName, hash)
}
