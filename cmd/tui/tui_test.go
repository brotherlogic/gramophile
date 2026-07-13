package tui

import (
	"context"
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

func TestLoginInitiatedTransition(t *testing.T) {
	m := InitialModel()
	m.state = StateLogin

	msg := loginInitiatedMsg{url: "http://example.com", token: "testtoken"}
	newModel, cmd := m.Update(msg)
	
	updatedModel, _ := newModel.(Model)

	if updatedModel.loginURL != "http://example.com" {
		t.Errorf("Expected loginURL to be set")
	}
	if updatedModel.loginToken != "testtoken" {
		t.Errorf("Expected loginToken to be set")
	}
	if cmd == nil {
		t.Errorf("Expected a command to be returned to poll login")
	}
}

func TestLoginPollTransition(t *testing.T) {
	m := InitialModel()
	m.state = StateLogin
	m.loginToken = "testtoken"

	// Mock an auth response
	msg := loginPollMsg{auth: nil, err: nil} // still waiting
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Errorf("Expected tick command when auth is nil")
	}

	// Wait, we need the auth proto, which we can't easily mock here without import. 
	// But let's just use an empty struct if we import it, or just test error state.
	msgErr := loginPollMsg{err: context.DeadlineExceeded}
	newModelErr, _ := m.Update(msgErr)
	updatedModelErr, _ := newModelErr.(Model)
	if updatedModelErr.loginErr != context.DeadlineExceeded {
		t.Errorf("Expected login error to be set")
	}
}
