package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/sant0-9/pulp/internal/config"
	"github.com/sant0-9/pulp/internal/llm"
)

type state struct {
	// Config
	config     *config.Config
	needsSetup bool

	// Setup wizard state
	setupStep        int
	selectedProvider int
	apiKeyInput      textinput.Model

	// Document state
	documentPath string
	documentInfo string

	// Processing
	processing   bool
	currentStage string
	progress     float64

	// Result
	result    string
	streaming bool

	// Input
	input textinput.Model

	// History
	history []message

	// Provider
	provider      llm.Provider
	localProvider llm.Provider
	providerReady bool
	providerError error
}

type message struct {
	role    string
	content string
}

func newState() *state {
	input := textinput.New()
	input.Placeholder = "/help for commands, or drop a file..."
	input.CharLimit = 500
	input.Width = 60

	apiKey := textinput.New()
	apiKey.Placeholder = "Paste your API key here..."
	apiKey.EchoMode = textinput.EchoPassword
	apiKey.CharLimit = 200
	apiKey.Width = 50

	return &state{
		input:       input,
		apiKeyInput: apiKey,
	}
}
