package tui

import (
	"context"
	"fmt"
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
	"github.com/sant0-9/pulp/internal/prompts"
	"github.com/sant0-9/pulp/internal/skill"
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
	viewNewSkill
	viewChat
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

	case skillGeneratedMsg:
		a.state.generatingSkill = false
		a.state.lastCreatedSkill = msg.skillName
		// Reload skill index
		a.state.skillIndex, _ = skill.NewSkillIndex()
		a.view = viewSkills
		return a, nil

	case skillGenerationErrorMsg:
		a.state.generatingSkill = false
		a.state.newSkillError = msg.error
		return a, nil

	case chatChunkMsg:
		// First chunk = streaming started
		if a.state.streamPhase == "connecting" {
			a.state.streamPhase = "streaming"
		}
		a.state.chatResult += msg.chunk
		a.state.streamTokens += estimateTokens(msg.chunk)
		a.state.contextUsed += estimateTokens(msg.chunk)
		return a, tickCmd() // Keep ticking for animation

	case chatDoneMsg:
		a.state.chatStreaming = false
		a.state.streamPhase = "complete"
		a.state.chatHistory = append(a.state.chatHistory, message{
			role:    "assistant",
			content: a.state.chatResult,
		})
		a.state.input.Focus()
		return a, textinput.Blink

	case chatErrorMsg:
		a.state.chatStreaming = false
		a.state.docError = msg.error
		return a, nil

	case tickMsg:
		// Animate spinner during streaming
		if a.state.chatStreaming || a.state.streaming {
			a.state.spinnerFrame++
			// Rotate loading message periodically
			if a.state.spinnerFrame%10 == 0 {
				a.state.loadingMessage = loadingMessages[a.state.spinnerFrame/10%len(loadingMessages)]
			}
			return a, tickCmd()
		}
		return a, nil
	}

	// Update text inputs based on view
	if a.view == viewSetup && a.state.setupStep == 1 {
		var cmd tea.Cmd
		a.state.apiKeyInput, cmd = a.state.apiKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if a.view == viewSettings && a.state.settingsMode == "model" {
		var cmd tea.Cmd
		a.state.modelInput, cmd = a.state.modelInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if a.view == viewSettings && a.state.settingsMode == "apikey" {
		var cmd tea.Cmd
		a.state.apiKeyInput, cmd = a.state.apiKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if a.view == viewWelcome || a.view == viewDocument || a.view == viewResult || a.view == viewNewSkill || a.view == viewChat {
		// Skip input update if palette is handling navigation keys
		skipInput := false
		if a.state.cmdPaletteActive && a.view == viewWelcome {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "up", "down", "ctrl+p", "ctrl+n", "tab":
					skipInput = true
				}
			}
		}

		if !skipInput {
			var cmd tea.Cmd
			a.state.input, cmd = a.state.input.Update(msg)
			cmds = append(cmds, cmd)

			// Update command palette when on welcome view
			if a.view == viewWelcome {
				a.updateCommandPalette()
			}
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Handle command palette navigation when active
	if a.state.cmdPaletteActive && a.view == viewWelcome {
		switch msg.String() {
		case "up", "ctrl+p":
			if a.state.cmdPaletteSelected > 0 {
				a.state.cmdPaletteSelected--
			}
			return nil
		case "down", "ctrl+n":
			if a.state.cmdPaletteSelected < len(a.state.cmdPaletteItems)-1 {
				a.state.cmdPaletteSelected++
			}
			return nil
		case "tab":
			// Autocomplete selected command
			if len(a.state.cmdPaletteItems) > 0 {
				selected := a.state.cmdPaletteItems[a.state.cmdPaletteSelected]
				a.state.input.SetValue(selected.cmd)
				a.state.input.SetCursor(len(selected.cmd))
				a.updateCommandPalette()
			}
			return nil
		case "enter":
			// Execute selected command
			if len(a.state.cmdPaletteItems) > 0 {
				selected := a.state.cmdPaletteItems[a.state.cmdPaletteSelected]
				a.state.input.SetValue(selected.cmd)
				a.state.cmdPaletteActive = false
				return a.handleInput()
			}
		case "esc":
			a.state.cmdPaletteActive = false
			a.state.input.Reset()
			return nil
		}
	}

	switch {
	case key.Matches(msg, keys.Quit):
		if a.state.cmdPaletteActive {
			a.state.cmdPaletteActive = false
			a.state.input.Reset()
			return nil
		}
		if a.view == viewSettings {
			if a.state.settingsMode != "" {
				// Go back to main settings
				a.state.settingsMode = ""
				a.state.modelInput.Blur()
				a.state.apiKeyInput.Blur()
				return nil
			}
			// Go back to welcome
			a.view = viewWelcome
			a.state.input.Reset()
			a.state.input.Placeholder = "/help for commands, or drop a file..."
			return nil
		}
		if a.view == viewHelp || a.view == viewSkills || a.view == viewNewSkill {
			a.view = viewWelcome
			a.state.input.Reset()
			a.state.input.Placeholder = "/help for commands, or drop a file..."
			return nil
		}
		if a.view == viewChat {
			if a.state.chatStreaming {
				// TODO: cancel streaming
				return nil
			}
			a.state.chatSkill = nil       // Clear active skill
			a.state.chatScrollOffset = 0  // Reset scroll
			a.view = viewWelcome
			a.state.input.Reset()
			a.state.input.Placeholder = "/help for commands, or drop a file..."
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
			a.state.cmdPaletteActive = false
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
		// Handle new skill creation
		if a.view == viewNewSkill && !a.state.generatingSkill {
			desc := strings.TrimSpace(a.state.input.Value())
			if desc != "" {
				a.state.input.Reset()
				a.state.generatingSkill = true
				a.state.newSkillError = nil
				return a.generateSkill(desc)
			}
		}
		// Handle chat view follow-up
		if a.view == viewChat && !a.state.chatStreaming {
			userMsg := strings.TrimSpace(a.state.input.Value())
			if userMsg != "" {
				a.state.chatHistory = append(a.state.chatHistory, message{
					role:    "user",
					content: userMsg,
				})
				a.state.chatStreaming = true
				a.state.chatResult = ""
				a.initStreamStats()
				a.state.input.Reset()
				return tea.Batch(a.startChat(userMsg), tickCmd())
			}
		}
	}

	// Handle 'n' for new document/chat
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
		if a.view == viewChat && !a.state.chatStreaming {
			a.state.chatHistory = nil
			a.state.chatResult = ""
			a.state.chatSkill = nil // Clear active skill
			a.state.lastStats = ""  // Clear stats
			a.state.contextUsed = 0
			a.state.streamTokens = 0
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

	// Handle welcome view shortcuts (always available, even with provider error)
	if a.view == viewWelcome && !a.state.cmdPaletteActive {
		inputVal := a.state.input.Value()
		// Only handle shortcuts when input is empty (user not typing)
		if inputVal == "" {
			switch msg.String() {
			case "s":
				a.view = viewSettings
				return nil
			case "?":
				a.view = viewHelp
				return nil
			}
		}
	}

	// Chat view scroll handling
	if a.view == viewChat {
		switch msg.String() {
		case "up", "k":
			a.state.chatScrollOffset += 3
			a.state.chatAutoScroll = false
			return nil
		case "down", "j":
			a.state.chatScrollOffset -= 3
			if a.state.chatScrollOffset < 0 {
				a.state.chatScrollOffset = 0
				a.state.chatAutoScroll = true
			}
			return nil
		case "pgup", "ctrl+u":
			a.state.chatScrollOffset += 10
			a.state.chatAutoScroll = false
			return nil
		case "pgdown", "ctrl+d":
			a.state.chatScrollOffset -= 10
			if a.state.chatScrollOffset < 0 {
				a.state.chatScrollOffset = 0
				a.state.chatAutoScroll = true
			}
			return nil
		case "home", "g":
			// Scroll to top - will be clamped in render
			a.state.chatScrollOffset = 99999
			a.state.chatAutoScroll = false
			return nil
		case "end", "G":
			a.state.chatScrollOffset = 0
			a.state.chatAutoScroll = true
			return nil
		}
	}

	// View-specific handling
	switch a.view {
	case viewSetup:
		return a.handleSetupKey(msg)
	case viewSettings:
		return a.handleSettingsKey(msg)
	}

	return nil
}

func (a *App) updateCommandPalette() {
	input := a.state.input.Value()

	// Only show palette when input starts with "/"
	if len(input) == 0 || input[0] != '/' {
		a.state.cmdPaletteActive = false
		a.state.cmdPaletteItems = nil
		return
	}

	// Build command list
	commands := []cmdItem{
		{"/help", "Show help"},
		{"/settings", "Open settings"},
		{"/skills", "List installed skills"},
		{"/new-skill", "Create a new skill"},
		{"/quit", "Exit pulp"},
	}

	// Add skill commands
	if a.state.skillIndex != nil {
		for _, name := range a.state.skillIndex.List() {
			desc := "Use skill"
			if meta := a.state.skillIndex.Get(name); meta != nil && meta.Description != "" {
				desc = meta.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
			}
			commands = append(commands, cmdItem{"/" + name, desc})
		}
	}

	// Filter by prefix
	var filtered []cmdItem
	for _, c := range commands {
		if strings.HasPrefix(c.cmd, input) {
			filtered = append(filtered, c)
		}
	}

	a.state.cmdPaletteItems = filtered
	a.state.cmdPaletteActive = len(filtered) > 0

	// Reset selection if out of bounds
	if a.state.cmdPaletteSelected >= len(filtered) {
		a.state.cmdPaletteSelected = 0
	}
}

func (a *App) handleInput() tea.Cmd {
	input := strings.TrimSpace(a.state.input.Value())
	if input == "" {
		return nil
	}

	// Strip quotes from file paths (common when dragging/dropping)
	input = strings.Trim(input, "'\"")
	input = strings.TrimSpace(input)

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
		case strings.HasPrefix(cmd, "/new-skill"):
			// Extract description after /new-skill
			desc := strings.TrimSpace(strings.TrimPrefix(input, "/new-skill"))
			if desc == "" {
				// Show skill creation view for input
				a.view = viewNewSkill
				a.state.input.Reset()
				a.state.input.Placeholder = "Describe the skill you want to create..."
				return nil
			}
			// Generate skill directly
			a.state.input.Reset()
			a.state.generatingSkill = true
			return a.generateSkill(desc)
		case cmd == "/quit" || cmd == "/q":
			a.quitting = true
			return tea.Quit
		default:
			// Check if it's a skill command (with optional message)
			// Format: /skill-name or /skill-name message to chat with skill
			parts := strings.SplitN(strings.TrimPrefix(input, "/"), " ", 2)
			skillName := strings.ToLower(parts[0])

			if a.state.skillIndex != nil {
				if meta := a.state.skillIndex.Get(skillName); meta != nil {
					// Load full skill
					fullSkill, err := skill.LoadFull(meta)
					if err != nil {
						a.state.docError = fmt.Errorf("failed to load skill: %v", err)
						a.state.input.Reset()
						return nil
					}

					a.state.chatSkill = fullSkill
					a.state.input.Reset()

					// If message provided, start chat immediately
					if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
						userMsg := strings.TrimSpace(parts[1])
						a.state.chatHistory = append(a.state.chatHistory, message{
							role:    "user",
							content: userMsg,
						})
						a.state.chatStreaming = true
						a.state.chatResult = ""
						a.state.docError = nil
						a.initStreamStats()
						a.view = viewChat
						return tea.Batch(a.startChat(userMsg), tickCmd())
					}

					// Just activate skill, go to chat view
					a.view = viewChat
					a.state.input.Placeholder = fmt.Sprintf("Chat with %s skill...", fullSkill.Name)
					return nil
				}
			}

			// Check if it looks like an absolute file path (has more path separators)
			if strings.Count(input, "/") > 1 || strings.Contains(input, ".") {
				// Treat as file path
				break
			}
			// Unknown command
			a.state.docError = fmt.Errorf("unknown command: %s (try /help)", input)
			a.state.input.Reset()
			return nil
		}
	}

	// Check if input looks like a file path
	if !looksLikeFilePath(input) {
		// Start general chat mode
		a.state.chatHistory = append(a.state.chatHistory, message{
			role:    "user",
			content: input,
		})
		a.state.chatStreaming = true
		a.state.chatResult = ""
		a.state.docError = nil
		a.initStreamStats()
		a.state.input.Reset()
		a.view = viewChat
		return tea.Batch(a.startChat(input), tickCmd())
	}

	// Handle file path input
	a.state.loadingDoc = true
	a.state.documentPath = input
	a.state.docError = nil
	a.state.input.Reset()
	return a.loadDocument(input)
}

// looksLikeFilePath checks if input appears to be a file path
func looksLikeFilePath(input string) bool {
	// Strip any remaining quotes for checking
	check := strings.Trim(input, "'\"")

	// Starts with path indicators
	if strings.HasPrefix(check, "./") ||
		strings.HasPrefix(check, "../") ||
		strings.HasPrefix(check, "~/") ||
		strings.HasPrefix(check, "/") {
		return true
	}

	// Contains path separator
	if strings.Contains(check, "/") || strings.Contains(check, "\\") {
		return true
	}

	// Has common document extensions
	lower := strings.ToLower(check)
	extensions := []string{".pdf", ".txt", ".md", ".doc", ".docx", ".html", ".htm", ".rtf", ".odt"}
	for _, ext := range extensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	return false
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

func (a *App) generateSkill(description string) tea.Cmd {
	return func() tea.Msg {
		generator := skill.NewGenerator(a.state.provider, a.state.config.Model)
		ctx := context.Background()

		newSkill, err := generator.Generate(ctx, description)
		if err != nil {
			return skillGenerationErrorMsg{err}
		}

		return skillGeneratedMsg{skillName: newSkill.Name}
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

func (a *App) startChat(userMessage string) tea.Cmd {
	return func() tea.Msg {
		// Build system prompt
		systemPrompt := a.buildChatSystemPrompt()

		// Build messages
		messages := []llm.Message{
			{Role: "system", Content: systemPrompt},
		}

		// Add chat history
		for _, m := range a.state.chatHistory {
			messages = append(messages, llm.Message{
				Role:    m.role,
				Content: m.content,
			})
		}

		ctx := context.Background()
		stream, err := a.state.provider.Stream(ctx, &llm.CompletionRequest{
			Model:       a.state.config.Model,
			Messages:    messages,
			MaxTokens:   2000,
			Temperature: 0.7,
		})
		if err != nil {
			return chatErrorMsg{err}
		}

		// Stream chunks via program.Send
		go func() {
			for event := range stream {
				if event.Error != nil {
					a.program.Send(chatErrorMsg{event.Error})
					return
				}
				if event.Done {
					a.program.Send(chatDoneMsg{})
					return
				}
				a.program.Send(chatChunkMsg{event.Chunk})
			}
			a.program.Send(chatDoneMsg{})
		}()

		return nil
	}
}

// buildChatSystemPrompt constructs the system prompt for chat mode
func (a *App) buildChatSystemPrompt() string {
	var skillName, skillBody string
	if a.state.chatSkill != nil {
		skillName = a.state.chatSkill.Name
		skillBody = a.state.chatSkill.Body
	}
	return prompts.BuildChatPrompt(skillName, skillBody)
}

// initStreamStats initializes streaming statistics before starting a chat
func (a *App) initStreamStats() {
	a.state.streamStart = time.Now()
	a.state.streamTokens = 0
	a.state.streamPhase = "connecting"
	a.state.spinnerFrame = 0
	a.state.lastStats = ""          // Clear previous stats
	a.state.chatScrollOffset = 0    // Scroll to bottom
	a.state.chatAutoScroll = true   // Enable auto-scroll

	// Calculate input context (system prompt + history)
	systemPrompt := a.buildChatSystemPrompt()
	inputTokens := estimateTokens(systemPrompt)
	for _, m := range a.state.chatHistory {
		inputTokens += estimateTokens(m.content)
	}
	a.state.contextUsed = inputTokens

	// Get context limit from model
	model := ""
	if a.state.config != nil {
		model = a.state.config.Model
	}
	a.state.contextLimit = getContextLimit(model)
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

func (a *App) handleSettingsKey(msg tea.KeyMsg) tea.Cmd {
	switch a.state.settingsMode {
	case "": // Main settings menu
		switch msg.String() {
		case "p":
			a.state.settingsMode = "provider"
			a.state.settingsSelected = 0
			// Find current provider index
			for i, p := range config.Providers {
				if p.ID == a.state.config.Provider {
					a.state.settingsSelected = i
					break
				}
			}
			return nil
		case "m":
			a.state.settingsMode = "model"
			a.state.settingsSelected = 0
			// Find current model index
			provider := config.GetProvider(a.state.config.Provider)
			if provider != nil {
				for i, m := range provider.Models {
					if m == a.state.config.Model {
						a.state.settingsSelected = i
						break
					}
				}
			}
			return nil
		case "k":
			a.state.settingsMode = "apikey"
			a.state.apiKeyInput.SetValue("")
			a.state.apiKeyInput.Focus()
			return textinput.Blink
		case "r":
			// Reset to setup wizard
			a.state.needsSetup = true
			a.state.setupStep = 0
			a.state.selectedProvider = 0
			a.view = viewSetup
			return nil
		}

	case "provider":
		switch msg.String() {
		case "up", "k":
			if a.state.settingsSelected > 0 {
				a.state.settingsSelected--
			}
		case "down", "j":
			if a.state.settingsSelected < len(config.Providers)-1 {
				a.state.settingsSelected++
			}
		case "enter":
			provider := config.Providers[a.state.settingsSelected]
			a.state.config.Provider = provider.ID
			a.state.config.Model = provider.DefaultModel
			a.state.config.Save()
			a.state.settingsMode = ""
			// Reconnect provider
			return a.testProvider()
		case "esc":
			a.state.settingsMode = ""
		}

	case "model":
		provider := config.GetProvider(a.state.config.Provider)
		if provider == nil {
			a.state.settingsMode = ""
			return nil
		}
		switch msg.String() {
		case "up", "k":
			if a.state.settingsSelected > 0 {
				a.state.settingsSelected--
			}
		case "down", "j":
			if a.state.settingsSelected < len(provider.Models)-1 {
				a.state.settingsSelected++
			}
		case "enter":
			if a.state.settingsSelected < len(provider.Models) {
				a.state.config.Model = provider.Models[a.state.settingsSelected]
				a.state.config.Save()
			}
			a.state.settingsMode = ""
			return nil
		case "esc":
			a.state.settingsMode = ""
			return nil
		}

	case "apikey":
		switch msg.String() {
		case "enter":
			key := a.state.apiKeyInput.Value()
			if key != "" {
				a.state.config.APIKey = key
				a.state.config.Save()
			}
			a.state.settingsMode = ""
			a.state.apiKeyInput.Blur()
			// Reconnect provider
			return a.testProvider()
		case "esc":
			a.state.settingsMode = ""
			a.state.apiKeyInput.Blur()
			return nil
		}
	}

	return nil
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
type skillGeneratedMsg struct {
	skillName string
}
type skillGenerationErrorMsg struct {
	error
}
type chatChunkMsg struct {
	chunk string
}
type chatDoneMsg struct{}
type chatErrorMsg struct {
	error
}
type tickMsg time.Time

// tickCmd returns a command that ticks for animations
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
	case viewNewSkill:
		return a.renderNewSkill()
	case viewChat:
		return a.renderChat()
	default:
		return a.renderWelcome()
	}
}
