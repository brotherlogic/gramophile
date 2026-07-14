package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
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
)

type AuthClient interface {
	GetURL(ctx context.Context, in *pb.GetURLRequest) (*pb.GetURLResponse, error)
	GetLogin(ctx context.Context, in *pb.GetLoginRequest) (*pb.GetLoginResponse, error)
	GetUser(ctx context.Context, in *pb.GetUserRequest) (*pb.GetUserResponse, error)
	GetState(ctx context.Context, in *pb.GetStateRequest) (*pb.GetStateResponse, error)
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
	err          error
}

// initialLogoDuration is the time to show the logo before auto-transitioning
const initialLogoDuration = 2 * time.Second

type Model struct {
	state      appState
	client     AuthClient
	loginURL   string
	loginToken string
	err        error
	tokenSaver func(string) error
	progress   float64
	progBar    progress.Model
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
			m.state = StateLoadingSync
			return m, m.pollSync()
		case loginErrMsg:
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
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
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return syncPollMsg{}
				})
			}
			m.err = nil
			
			if msg.expectedSize > 0 {
				m.progress = float64(msg.currentSize) / float64(msg.expectedSize)
			}
			
			if msg.userState == pb.StoredUser_USER_STATE_IN_WAITLIST {
				m.state = StateWaitlist
				return m, nil
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
			return fmt.Sprintf("Error fetching sync state: %v\n\nRetrying...", m.err)
		}
		return fmt.Sprintf("\nSyncing Collection with Discogs...\n\n%s\n", m.progBar.ViewAs(m.progress))
	}
	return "Gramophile TUI"
}
