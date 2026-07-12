package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	state appState
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateStartupLogo:
		switch msg.(type) {
		case tea.KeyMsg:
			// Any key press skips the logo
			m.state = StateLogin
			return m, nil
		case timeoutMsg:
			// Auto transition after timer
			m.state = StateLogin
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
		return "Login Screen"
	}
	return "Gramophile TUI"
}
