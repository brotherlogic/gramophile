package tui

import (
	"context"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/browser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

type appState int

const (
	StateStartupLogo appState = iota
	StateLogin
	StateLoadingSync
	StateWaitlist
	StateMainApp
)

type timeoutMsg struct{}
type tickPollMsg struct{}

type loginInitiatedMsg struct {
	url   string
	token string
	err   error
}

type loginPollMsg struct {
	auth *pb.GramophileAuth
	err  error
}

// initialLogoDuration is the time to show the logo before auto-transitioning
const initialLogoDuration = 2 * time.Second

type Model struct {
	state       appState
	loginURL    string
	loginToken  string
	loginErr    error
	loginStatus string
}

func InitialModel() Model {
	return Model{
		state: StateStartupLogo,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(initialLogoDuration, func(t time.Time) tea.Msg {
		return timeoutMsg{}
	})
}

func saveToken(auth *pb.GramophileAuth) error {
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

	err = proto.MarshalText(f, auth)
	f.Close()

	if err != nil {
		return err
	}

	return os.Rename(tmpFile, finalFile)
}

func (m Model) initLogin() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return loginInitiatedMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewGramophileEServiceClient(conn)
		resp, err := client.GetURL(ctx, &pb.GetURLRequest{})
		if err != nil {
			return loginInitiatedMsg{err: err}
		}

		return loginInitiatedMsg{url: resp.GetURL(), token: resp.GetToken()}
	}
}

func (m Model) pollLoginTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickPollMsg{}
	})
}

func (m Model) checkLogin(token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return loginPollMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewGramophileEServiceClient(conn)
		resp, err := client.GetLogin(ctx, &pb.GetLoginRequest{Token: token})
		if err != nil {
			if status.Code(err) != codes.DataLoss {
				return loginPollMsg{err: err}
			}
			return loginPollMsg{auth: nil, err: nil} // keep polling
		}

		return loginPollMsg{auth: resp.GetAuth(), err: nil}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateStartupLogo:
		switch msg.(type) {
		case tea.KeyMsg:
			m.state = StateLogin
			return m, m.initLogin()
		case timeoutMsg:
			m.state = StateLogin
			return m, m.initLogin()
		}

	case StateLogin:
		switch msg := msg.(type) {
		case loginInitiatedMsg:
			if msg.err != nil {
				m.loginErr = msg.err
				m.loginStatus = "Failed to get login URL: " + msg.err.Error()
				return m, nil
			}
			m.loginURL = msg.url
			m.loginToken = msg.token
			m.loginStatus = "Please open this URL in your browser:\n\n" + msg.url
			
			// Try to open the browser automatically
			_ = browser.OpenURL(msg.url)
			
			return m, m.pollLoginTick()

		case tickPollMsg:
			return m, m.checkLogin(m.loginToken)

		case loginPollMsg:
			if msg.err != nil {
				m.loginErr = msg.err
				m.loginStatus = "Error checking login: " + msg.err.Error()
				return m, nil
			}
			if msg.auth == nil {
				// Keep polling
				return m, m.pollLoginTick()
			}
			
			// Success
			err := saveToken(msg.auth)
			if err != nil {
				m.loginErr = err
				m.loginStatus = "Error saving token: " + err.Error()
				return m, nil
			}
			
			m.loginStatus = "Login successful!"
			m.state = StateLoadingSync
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
		if m.loginErr != nil {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(m.loginStatus)
		}
		if m.loginURL == "" {
			return "Fetching login URL..."
		}
		
		style := lipgloss.NewStyle().Padding(1, 2)
		return style.Render(m.loginStatus + "\n\nWaiting for authentication...")
	}
	return "Gramophile TUI"
}
