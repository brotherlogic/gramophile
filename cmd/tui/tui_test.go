package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStateTransitions(t *testing.T) {
	m := InitialModel()

	if m.state != StateStartupLogo {
		t.Errorf("Expected initial state to be StateStartupLogo, got %v", m.state)
	}

	// Any key press in StateStartupLogo should transition to StateLogin
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}

	if updatedModel.state != StateLogin {
		t.Errorf("Expected state to transition to StateLogin on key press, got %v", updatedModel.state)
	}
}

func TestTimerTransition(t *testing.T) {
	m := InitialModel()
	
	// A timeout message should transition to StateLogin
	msg := timeoutMsg{}
	newModel, _ := m.Update(msg)
	
	updatedModel, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Expected model to be of type Model")
	}

	if updatedModel.state != StateLogin {
		t.Errorf("Expected state to transition to StateLogin on timeout, got %v", updatedModel.state)
	}
}

type mockGetURLMsg struct {
	url   string
	token string
}

func TestLoginURLTransition(t *testing.T) {
	m := InitialModel()
	m.state = StateLogin

	// When receiving a loginURLMsg, it should update the URL and token
	msg := loginURLMsg{url: "http://test", token: "tok123"}
	newModel, _ := m.Update(msg)
	
	updatedModel, _ := newModel.(Model)
	if updatedModel.url != "http://test" {
		t.Errorf("Expected url to be http://test, got %v", updatedModel.url)
	}
	if updatedModel.loginToken != "tok123" {
		t.Errorf("Expected token to be tok123, got %v", updatedModel.loginToken)
	}
}

func TestLoginSuccessTransition(t *testing.T) {
	m := InitialModel()
	m.state = StateLogin
	m.loginToken = "tok123"

	// When receiving a loginSuccessMsg, it should transition to StateLoadingSync
	msg := loginSuccessMsg{auth: nil}
	newModel, _ := m.Update(msg)
	
	updatedModel, _ := newModel.(Model)
	if updatedModel.state != StateLoadingSync {
		t.Errorf("Expected state to transition to StateLoadingSync on success, got %v", updatedModel.state)
	}
}
