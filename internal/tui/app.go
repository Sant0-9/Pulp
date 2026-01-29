package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sant0-9/pulp/internal/config"
	"github.com/sant0-9/pulp/internal/llm"
)

type view int

const (
	viewWelcome view = iota
	viewSetup
	viewDocument
	viewProcessing
	viewResult
	viewSettings
	viewHelp
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
		return tea.Batch(tea.WindowSize(), textinput.Blink)
	}

	// Test provider connection
	return tea.Batch(
		tea.WindowSize(),
		textinput.Blink,
		a.testProvider(),
	)
}

func (a *App) testProvider() tea.Cmd {
	return func() tea.Msg {
		provider, err := llm.NewProvider(a.state.config)
		if err != nil {
			return providerErrorMsg{err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := provider.Ping(ctx); err != nil {
			return providerErrorMsg{err}
		}

		return providerReadyMsg{}
	}
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
		return a, a.testProvider()

	case setupErrorMsg:
		// TODO: show error
		return a, nil

	case providerReadyMsg:
		a.state.providerReady = true
		provider, _ := llm.NewProvider(a.state.config)
		a.state.provider = provider
		a.state.input.Focus()
		return a, textinput.Blink

	case providerErrorMsg:
		a.state.providerError = msg.error
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
		if a.view == viewSettings || a.view == viewHelp {
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

	case key.Matches(msg, keys.Enter):
		if a.view == viewWelcome && a.state.providerReady {
			return a.handleInput()
		}
	}

	// View-specific handling
	switch a.view {
	case viewSetup:
		return a.handleSetupKey(msg)
	}

	return nil
}

func (a *App) handleInput() tea.Cmd {
	input := strings.TrimSpace(a.state.input.Value())
	if input == "" {
		return nil
	}

	// Handle slash commands
	if strings.HasPrefix(input, "/") {
		cmd := strings.ToLower(input)
		switch {
		case cmd == "/help" || cmd == "/h":
			a.view = viewHelp
			a.state.input.Reset()
			return nil
		case cmd == "/settings" || cmd == "/s":
			a.view = viewSettings
			a.state.input.Reset()
			return nil
		case cmd == "/quit" || cmd == "/q":
			a.quitting = true
			return tea.Quit
		}
	}

	// TODO: handle file paths and other input
	a.state.input.Reset()
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
type providerReadyMsg struct{}
type providerErrorMsg struct{ error }

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
	case viewHelp:
		return a.renderHelp()
	default:
		return a.renderWelcome()
	}
}
