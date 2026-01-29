package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sant0-9/pulp/internal/config"
)

type view int

const (
	viewWelcome view = iota
	viewSetup
	viewDocument
	viewProcessing
	viewResult
	viewSettings
)

type App struct {
	width    int
	height   int
	view     view
	state    *state
	quitting bool
}

func NewApp() *App {
	s := newState()

	// Check if setup needed
	cfg, _ := config.Load()
	if cfg == nil {
		s.needsSetup = true
		s.config = config.DefaultConfig()
	} else {
		s.config = cfg
	}

	return &App{
		view:  viewWelcome,
		state: s,
	}
}

func (a *App) Init() tea.Cmd {
	if a.state.needsSetup {
		a.view = viewSetup
	}
	return tea.Batch(tea.WindowSize(), textinput.Blink)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd := a.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case setupCompleteMsg:
		a.state.needsSetup = false
		a.view = viewWelcome
		a.state.input.Focus()
		return a, textinput.Blink

	case setupErrorMsg:
		// TODO: show error
		return a, nil
	}

	// Update text inputs based on view
	if a.view == viewSetup && a.state.setupStep == 1 {
		var cmd tea.Cmd
		a.state.apiKeyInput, cmd = a.state.apiKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if a.view == viewWelcome || a.view == viewDocument || a.view == viewResult {
		var cmd tea.Cmd
		a.state.input, cmd = a.state.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keys.Quit):
		if a.view == viewSettings {
			a.view = viewWelcome
			return nil
		}
		if a.view == viewSetup && a.state.setupStep == 1 {
			// Go back to provider selection
			a.state.setupStep = 0
			a.state.apiKeyInput.Reset()
			return nil
		}
		a.quitting = true
		return tea.Quit

	case msg.String() == "s":
		if a.view == viewWelcome && !a.state.needsSetup {
			a.view = viewSettings
			return nil
		}
	}

	// View-specific handling
	switch a.view {
	case viewSetup:
		return a.handleSetupKey(msg)
	}

	return nil
}

func (a *App) handleSetupKey(msg tea.KeyMsg) tea.Cmd {
	switch a.state.setupStep {
	case 0: // Provider selection
		switch msg.String() {
		case "up", "k":
			if a.state.selectedProvider > 0 {
				a.state.selectedProvider--
			}
		case "down", "j":
			if a.state.selectedProvider < len(config.Providers)-1 {
				a.state.selectedProvider++
			}
		case "enter":
			provider := config.Providers[a.state.selectedProvider]
			a.state.config.Provider = provider.ID
			a.state.config.Model = provider.DefaultModel

			if provider.NeedsAPIKey {
				a.state.setupStep = 1
				a.state.apiKeyInput.Focus()
				return textinput.Blink
			} else {
				// Skip to save
				return a.finishSetup()
			}
		}

	case 1: // API key entry
		switch msg.String() {
		case "enter":
			a.state.config.APIKey = a.state.apiKeyInput.Value()
			return a.finishSetup()
		}
	}

	return nil
}

func (a *App) finishSetup() tea.Cmd {
	return func() tea.Msg {
		if err := a.state.config.Save(); err != nil {
			return setupErrorMsg{err}
		}
		return setupCompleteMsg{}
	}
}

type setupCompleteMsg struct{}
type setupErrorMsg struct{ error }

func (a *App) View() string {
	if a.quitting {
		return ""
	}

	switch a.view {
	case viewWelcome:
		return a.renderWelcome()
	case viewSetup:
		return a.renderSetup()
	case viewSettings:
		return a.renderSettings()
	default:
		return a.renderWelcome()
	}
}
