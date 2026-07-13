package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

// initialLogoDuration is the time to show the logo before auto-transitioning
const initialLogoDuration = 2 * time.Second

type Model struct {
	state      appState
	url        string
	loginToken string
	err        error
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

type loginURLMsg struct {
	url   string
	token string
}

type loginSuccessMsg struct {
	auth *pb.GramophileAuth
}

type errMsg struct {
	err error
}

type retryPollLoginMsg struct {
	token string
}

func fetchLoginURLCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return errMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewGramophileEServiceClient(conn)
		resp, err := client.GetURL(ctx, &pb.GetURLRequest{})
		if err != nil {
			return errMsg{err: err}
		}

		return loginURLMsg{url: resp.GetURL(), token: resp.GetToken()}
	}
}

func pollLoginCmd(token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return errMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewGramophileEServiceClient(conn)
		resp, err := client.GetLogin(ctx, &pb.GetLoginRequest{Token: token})
		if err != nil {
			return errMsg{err: err}
		}

		return loginSuccessMsg{auth: resp.GetAuth()}
	}
}

func tickRetryPollLoginCmd(token string) tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return retryPollLoginMsg{token: token}
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateStartupLogo:
		switch msg.(type) {
		case tea.KeyMsg:
			// Any key press skips the logo
			m.state = StateLogin
			return m, fetchLoginURLCmd()
		case timeoutMsg:
			// Auto transition after timer
			m.state = StateLogin
			return m, fetchLoginURLCmd()
		}
	case StateLogin:
		switch msg := msg.(type) {
		case loginURLMsg:
			m.url = msg.url
			m.loginToken = msg.token
			return m, pollLoginCmd(m.loginToken)
		case loginSuccessMsg:
			err := saveToken(msg.auth)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.state = StateLoadingSync
			return m, nil
		case errMsg:
			if status.Code(msg.err) == codes.DataLoss {
				return m, tickRetryPollLoginCmd(m.loginToken)
			}
			m.err = msg.err
			return m, nil
		case retryPollLoginMsg:
			return m, pollLoginCmd(msg.token)
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
		if m.err != nil && status.Code(m.err) != codes.DataLoss {
			return fmt.Sprintf("Error logging in: %v\nPress q to quit", m.err)
		}
		if m.url == "" {
			return "Fetching login URL..."
		}
		return fmt.Sprintf("Please open this URL in your browser to authenticate:\n\n%s\n\nWaiting for authorization...", m.url)
	}
	return "Gramophile TUI"
}
