package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/sant0-9/pulp/internal/config"
	"github.com/sant0-9/pulp/internal/converter"
	"github.com/sant0-9/pulp/internal/intent"
	"github.com/sant0-9/pulp/internal/llm"
	"github.com/sant0-9/pulp/internal/pipeline"
	"github.com/sant0-9/pulp/internal/skill"
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
	document     *converter.Document
	documentPath string
	loadingDoc   bool
	docError     error

	// Processing
	processing   bool
	currentStage string
	progress     float64

	// Result
	result    string
	streaming bool

	// Input
	input textinput.Model

	// History and follow-up tracking
	history    []message
	isFollowUp bool

	// Provider
	provider      llm.Provider
	localProvider llm.Provider
	providerReady bool
	providerError error

	// Intent
	currentIntent *intent.Intent
	parsingIntent bool

	// Pipeline
	pipelineProgress *pipeline.Progress
	pipelineResult   *pipeline.Result
	processingError  error

	// Skills
	skillIndex       *skill.SkillIndex
	generatingSkill  bool
	newSkillError    error
	lastCreatedSkill string
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

	// Load skill index (errors are ignored - skills are optional)
	skillIdx, _ := skill.NewSkillIndex()

	return &state{
		input:       input,
		apiKeyInput: apiKey,
		skillIndex:  skillIdx,
	}
}
