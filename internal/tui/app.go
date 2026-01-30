package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sant0-9/pulp/internal/config"
	"github.com/sant0-9/pulp/internal/converter"
	"github.com/sant0-9/pulp/internal/intent"
	"github.com/sant0-9/pulp/internal/llm"
	"github.com/sant0-9/pulp/internal/pipeline"
	"github.com/sant0-9/pulp/internal/writer"
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
	viewSkills
)

type App struct {
	width    int
	height   int
	view     view
	state    *state
	quitting bool
	program  *tea.Program
}

// SetProgram sets the tea.Program reference for async messaging
func (a *App) SetProgram(p *tea.Program) {
	a.program = p
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

	case documentLoadedMsg:
		a.state.loadingDoc = false
		a.state.document = msg.doc
		a.state.docError = nil
		a.view = viewDocument
		a.state.input.Reset()
		a.state.input.Placeholder = "What do you want to do with this document?"
		a.state.input.Focus()
		return a, textinput.Blink

	case documentErrorMsg:
		a.state.loadingDoc = false
		a.state.docError = msg.error
		return a, nil

	case intentParsedMsg:
		a.state.parsingIntent = false
		a.state.currentIntent = msg.intent

		if a.state.isFollowUp {
			// Skip pipeline, go straight to writer (reuse cached extraction)
			a.state.streaming = true
			a.state.result = ""
			a.view = viewResult
			return a, a.startWriter()
		}

		// First time: run full pipeline
		a.view = viewProcessing
		return a, a.runPipeline()

	case pipelineProgressMsg:
		a.state.pipelineProgress = &msg.progress
		return a, nil

	case pipelineDoneMsg:
		a.state.pipelineResult = msg.result
		a.state.streaming = true
		a.state.result = ""
		a.view = viewResult
		return a, a.startWriter()

	case streamChunkMsg:
		a.state.result += msg.chunk
		return a, nil

	case streamDoneMsg:
		a.state.streaming = false
		a.state.history = append(a.state.history, message{
			role:    "assistant",
			content: a.state.result,
		})
		a.state.input.Focus() // Focus input for follow-up
		return a, textinput.Blink

	case streamErrorMsg:
		a.state.streaming = false
		a.state.processingError = msg.error
		return a, nil

	case clipboardMsg:
		// Could show notification
		return a, nil

	case saveMsg:
		// Could show notification with path
		return a, nil

	case pipelineErrorMsg:
		a.state.processingError = msg.error
		a.view = viewDocument
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
		if a.view == viewSettings || a.view == viewHelp || a.view == viewSkills {
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
		if a.view == viewDocument && a.state.providerReady {
			instruction := strings.TrimSpace(a.state.input.Value())
			if instruction != "" {
				a.state.parsingIntent = true
				a.state.input.Reset()
				return a.parseIntent(instruction)
			}
		}
		// Handle result view follow-up
		if a.view == viewResult && !a.state.streaming {
			instruction := strings.TrimSpace(a.state.input.Value())
			if instruction != "" {
				// Add user message to history
				a.state.history = append(a.state.history, message{
					role:    "user",
					content: instruction,
				})
				a.state.isFollowUp = true
				a.state.input.Reset()
				return a.parseIntent(instruction)
			}
		}
	}

	// Handle 'n' for new document
	if msg.String() == "n" {
		if a.view == viewDocument || a.view == viewResult {
			a.state.document = nil
			a.state.documentPath = ""
			a.state.docError = nil
			a.state.currentIntent = nil
			a.state.pipelineResult = nil
			a.state.result = ""
			a.state.history = nil      // Clear history
			a.state.isFollowUp = false // Reset flag
			a.state.input.Reset()
			a.state.input.Placeholder = "/help for commands, or drop a file..."
			a.view = viewWelcome
			return nil
		}
	}

	// Handle result view keys
	if a.view == viewResult && !a.state.streaming {
		switch msg.String() {
		case "c":
			return copyToClipboard(a.state.result)
		case "s":
			return saveToFile(a.state.result, a.state.document.Metadata.Title)
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
		case cmd == "/skills":
			a.view = viewSkills
			a.state.input.Reset()
			return nil
		case cmd == "/quit" || cmd == "/q":
			a.quitting = true
			return tea.Quit
		}
	}

	// Handle file path input
	a.state.loadingDoc = true
	a.state.documentPath = input
	a.state.docError = nil
	a.state.input.Reset()
	return a.loadDocument(input)
}

func (a *App) loadDocument(path string) tea.Cmd {
	return func() tea.Msg {
		converter, err := converter.NewConverter()
		if err != nil {
			return documentErrorMsg{err}
		}

		ctx := context.Background()
		doc, err := converter.Convert(ctx, path)
		if err != nil {
			return documentErrorMsg{err}
		}

		return documentLoadedMsg{doc}
	}
}

func (a *App) parseIntent(instruction string) tea.Cmd {
	return func() tea.Msg {
		parser := intent.NewParser(a.state.provider, a.state.config.Model, a.state.skillIndex)
		ctx := context.Background()

		parsed, err := parser.Parse(ctx, instruction)
		if err != nil {
			// Use simple intent on error
			parsed = intent.New(instruction)
		}

		return intentParsedMsg{parsed}
	}
}

func (a *App) runPipeline() tea.Cmd {
	return func() tea.Msg {
		pipe := pipeline.NewPipeline(a.state.provider, a.state.config.Model)

		ctx := context.Background()
		result, err := pipe.Process(ctx, a.state.document, a.state.currentIntent)
		if err != nil {
			return pipelineErrorMsg{err}
		}

		return pipelineDoneMsg{result}
	}
}

func (a *App) startWriter() tea.Cmd {
	return func() tea.Msg {
		w := writer.NewWriter(a.state.provider, a.state.config.Model)

		// Convert history to writer format
		var history []writer.Message
		for _, m := range a.state.history {
			history = append(history, writer.Message{
				Role:    m.role,
				Content: m.content,
			})
		}

		// Get previous result for follow-ups
		var previousResult string
		if a.state.isFollowUp && len(a.state.history) > 0 {
			// Find last assistant message
			for i := len(a.state.history) - 1; i >= 0; i-- {
				if a.state.history[i].role == "assistant" {
					previousResult = a.state.history[i].content
					break
				}
			}
		}

		req := &writer.WriteRequest{
			Aggregated:     a.state.pipelineResult.Aggregated,
			Intent:         a.state.currentIntent,
			DocTitle:       a.state.document.Metadata.Title,
			History:        history,
			IsFollowUp:     a.state.isFollowUp,
			PreviousResult: previousResult,
		}

		ctx := context.Background()
		stream, err := w.Stream(ctx, req)
		if err != nil {
			return streamErrorMsg{err}
		}

		// Stream chunks via program.Send for real-time updates
		go func() {
			for event := range stream {
				if event.Error != nil {
					a.program.Send(streamErrorMsg{event.Error})
					return
				}
				if event.Done {
					a.program.Send(streamDoneMsg{})
					return
				}
				a.program.Send(streamChunkMsg{event.Chunk})
			}
			a.program.Send(streamDoneMsg{})
		}()

		return nil
	}
}

func copyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		// For simplicity, just return success
		// A real implementation would use OS-specific clipboard commands
		return clipboardMsg{success: true}
	}
}

func saveToFile(content, title string) tea.Cmd {
	return func() tea.Msg {
		// Generate filename
		filename := strings.ReplaceAll(title, " ", "_") + "_summary.md"
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, "Documents", filename)

		// Ensure directory exists
		os.MkdirAll(filepath.Dir(path), 0755)

		// Write file
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return saveMsg{err: err}
		}

		return saveMsg{path: path}
	}
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
type documentLoadedMsg struct {
	doc *converter.Document
}
type documentErrorMsg struct{ error }
type intentParsedMsg struct {
	intent *intent.Intent
}
type pipelineProgressMsg struct {
	progress pipeline.Progress
}
type pipelineDoneMsg struct {
	result *pipeline.Result
}
type pipelineErrorMsg struct {
	error
}
type streamChunkMsg struct {
	chunk string
}
type streamDoneMsg struct{}
type streamErrorMsg struct {
	error
}
type clipboardMsg struct {
	success bool
	err     error
}
type saveMsg struct {
	path string
	err  error
}

func (a *App) View() string {
	if a.quitting {
		return ""
	}

	switch a.view {
	case viewWelcome:
		return a.renderWelcome()
	case viewSetup:
		return a.renderSetup()
	case viewDocument:
		return a.renderDocument()
	case viewProcessing:
		return a.renderProcessing()
	case viewResult:
		return a.renderResult()
	case viewSettings:
		return a.renderSettings()
	case viewHelp:
		return a.renderHelp()
	case viewSkills:
		return a.renderSkills()
	default:
		return a.renderWelcome()
	}
}
