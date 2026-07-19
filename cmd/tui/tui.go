package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang/protobuf/proto"
)

type appState int

const (
	StateStartupLogo appState = iota
	StateLogin
	StateLoadingSync
	StateWaitlist
	StateMainApp
	StateOrgConfig
)

type AuthClient interface {
	GetURL(ctx context.Context, in *pb.GetURLRequest) (*pb.GetURLResponse, error)
	GetLogin(ctx context.Context, in *pb.GetLoginRequest) (*pb.GetLoginResponse, error)
	GetUser(ctx context.Context, in *pb.GetUserRequest) (*pb.GetUserResponse, error)
	GetState(ctx context.Context, in *pb.GetStateRequest) (*pb.GetStateResponse, error)
	SetConfig(ctx context.Context, in *pb.SetConfigRequest) (*pb.SetConfigResponse, error)
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

// initialLogoDuration is the time to show the logo before auto-transitioning
const initialLogoDuration = 2 * time.Second

type Model struct {
	state           appState
	client          AuthClient
	loginURL        string
	loginToken      string
	err             error
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

func InitialModel(client AuthClient) Model {
	return Model{
		state:      StateStartupLogo,
		client:     client,
		tokenSaver: defaultTokenSaver,
		progBar:    progress.New(progress.WithDefaultGradient()),
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
				m.initOrgConfigForm()
				return m, nil
			}
		}
	case StateOrgConfig:
		if msg, ok := msg.(setConfigMsg); ok {
			if msg.err != nil {
				m.err = msg.err
				m.form = nil
				return m, nil
			}
			m.state = StateMainApp
			m.form = nil
			return m, nil
		}

		if m.err != nil {
			if _, ok := msg.(tea.KeyMsg); ok {
				m.err = nil
				m.state = StateMainApp
				return m, nil
			}
			return m, nil
		}

		if m.form != nil {
			form, cmd := m.form.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.form = f
			}

			if m.form.State == huh.StateCompleted {
				unitsVal, _ := strconv.Atoi(m.spaceUnits)
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
					folderId, _ := strconv.Atoi(folderIdStr)
					var folderName string
					if m.user != nil {
						for _, f := range m.user.GetFolders() {
							if f.GetId() == int32(folderId) {
								folderName = f.GetName()
								break
							}
						}
					}
					foldersets = append(foldersets, &pb.FolderSet{
						Name:   folderName,
						Folder: int32(folderId),
						Sort:   sortVal,
					})
				}

				newOrg := &pb.Organisation{
					Name:       m.orgName,
					Foldersets: foldersets,
					Spaces: []*pb.Space{
						{
							Name:  m.spaceName,
							Units: int32(unitsVal),
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
				return m, nil
			}

			return m, cmd
		}
	}

	// Handle global quit
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
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
		if m.err != nil {
			return fmt.Sprintf("Error saving organization configuration:\n\n  %v\n\nPress any key to return...", m.err)
		}
		if m.form != nil {
			return m.form.View()
		}
		return "Loading wizard..."
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
					val, err := strconv.Atoi(str)
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
