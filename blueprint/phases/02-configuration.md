# Phase 2: Configuration

## Goal
Build the configuration system and first-run setup wizard. User can select provider, enter API key, and config persists.

## Success Criteria
- First run shows setup wizard
- User can select provider from list
- User can enter API key
- Config saves to `~/.config/pulp/config.yaml`
- Subsequent runs skip setup if config exists
- Can access settings with `s` key

---

## Files to Create

### 1. Config Package

```
pulp/internal/config/config.go
```

```go
package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"api_key,omitempty"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url,omitempty"`

	Local *LocalConfig `yaml:"local,omitempty"`
}

type LocalConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
	Host     string `yaml:"host"`
	Model    string `yaml:"model"`
}

func DefaultConfig() *Config {
	return &Config{
		Provider: "ollama",
		Model:    "llama3.1:8b",
		Local: &LocalConfig{
			Enabled:  true,
			Provider: "ollama",
			Host:     "http://localhost:11434",
			Model:    "qwen2.5:3b",
		},
	}
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pulp"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Exists() bool {
	path, err := ConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
```

---

### 2. Provider Info

```
pulp/internal/config/providers.go
```

```go
package config

type ProviderInfo struct {
	ID          string
	Name        string
	Description string
	NeedsAPIKey bool
	SignupURL   string
	Models      []string
	DefaultModel string
}

var Providers = []ProviderInfo{
	{
		ID:          "ollama",
		Name:        "Ollama",
		Description: "Local, free, private",
		NeedsAPIKey: false,
		Models:      []string{"llama3.1:8b", "llama3.1:70b", "qwen2.5:7b", "mistral:7b"},
		DefaultModel: "llama3.1:8b",
	},
	{
		ID:          "groq",
		Name:        "Groq",
		Description: "Very fast, cheap",
		NeedsAPIKey: true,
		SignupURL:   "https://console.groq.com/keys",
		Models:      []string{"llama-3.1-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"},
		DefaultModel: "llama-3.1-70b-versatile",
	},
	{
		ID:          "openai",
		Name:        "OpenAI",
		Description: "GPT-4o, most capable",
		NeedsAPIKey: true,
		SignupURL:   "https://platform.openai.com/api-keys",
		Models:      []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo"},
		DefaultModel: "gpt-4o-mini",
	},
	{
		ID:          "anthropic",
		Name:        "Anthropic",
		Description: "Claude, great writing",
		NeedsAPIKey: true,
		SignupURL:   "https://console.anthropic.com/",
		Models:      []string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022"},
		DefaultModel: "claude-3-5-sonnet-20241022",
	},
	{
		ID:          "openrouter",
		Name:        "OpenRouter",
		Description: "Access all models",
		NeedsAPIKey: true,
		SignupURL:   "https://openrouter.ai/keys",
		Models:      []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o", "meta-llama/llama-3.1-70b"},
		DefaultModel: "meta-llama/llama-3.1-70b-instruct",
	},
}

func GetProvider(id string) *ProviderInfo {
	for _, p := range Providers {
		if p.ID == id {
			return &p
		}
	}
	return nil
}
```

---

### 3. Update App State

```
pulp/internal/tui/state.go
```

```go
package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/sant0-9/pulp/internal/config"
)

type state struct {
	// Config
	config       *config.Config
	needsSetup   bool

	// Setup wizard state
	setupStep       int
	selectedProvider int
	apiKeyInput     textinput.Model

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
}

type message struct {
	role    string
	content string
}

func newState() *state {
	input := textinput.New()
	input.Placeholder = "Drop a file or type a path..."
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
```

---

### 4. Update App Model

Replace `pulp/internal/tui/app.go`:

```go
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
	return textinput.Blink
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

func (a *App) Update2(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle setup completion
	switch msg.(type) {
	case setupCompleteMsg:
		a.state.needsSetup = false
		a.view = viewWelcome
		a.state.input.Focus()
		return a, textinput.Blink
	}
	return a, nil
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
	case viewSettings:
		return a.renderSettings()
	default:
		return a.renderWelcome()
	}
}
```

---

### 5. Setup View

```
pulp/internal/tui/view_setup.go
```

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sant0-9/pulp/internal/config"
)

func (a *App) renderSetup() string {
	switch a.state.setupStep {
	case 0:
		return a.renderProviderSelection()
	case 1:
		return a.renderAPIKeyEntry()
	default:
		return ""
	}
}

func (a *App) renderProviderSelection() string {
	var b strings.Builder

	// Header
	header := styleLogo.Render(logo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, header))
	b.WriteString("\n\n")

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorWhite).
		Bold(true).
		Render("Welcome! Choose your LLM provider:")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Provider list
	var providerLines []string
	for i, p := range config.Providers {
		var line string
		cursor := "  "
		if i == a.state.selectedProvider {
			cursor = "> "
			line = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				Render(fmt.Sprintf("%s[x] %-12s %s", cursor, p.Name, p.Description))
		} else {
			line = lipgloss.NewStyle().
				Foreground(colorMuted).
				Render(fmt.Sprintf("%s[ ] %-12s %s", cursor, p.Name, p.Description))
		}
		providerLines = append(providerLines, line)
	}

	providerBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(providerLines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, providerBox))
	b.WriteString("\n\n")

	// Instructions
	instructions := styleStatusBar.Render("[j/k] Navigate  [Enter] Select  [?] Compare")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}

func (a *App) renderAPIKeyEntry() string {
	var b strings.Builder

	provider := config.GetProvider(a.state.config.Provider)

	// Header
	header := styleLogo.Render(logo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, header))
	b.WriteString("\n\n")

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorWhite).
		Bold(true).
		Render(fmt.Sprintf("Enter your %s API key:", provider.Name))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Signup link
	if provider.SignupURL != "" {
		link := styleSubtitle.Render(fmt.Sprintf("Get one at: %s", provider.SignupURL))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, link))
		b.WriteString("\n\n")
	}

	// Input
	inputBox := styleBox.Copy().
		Width(60).
		BorderForeground(colorSecondary).
		Render(a.state.apiKeyInput.View())
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
	b.WriteString("\n\n")

	// Instructions
	instructions := styleStatusBar.Render("[Enter] Continue  [Esc] Back")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}

func (a *App) centerVertically(content string) string {
	lines := strings.Count(content, "\n") + 1
	padding := (a.height - lines) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat("\n", padding) + content
}
```

---

### 6. Settings View

```
pulp/internal/tui/view_settings.go
```

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sant0-9/pulp/internal/config"
)

func (a *App) renderSettings() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Settings")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Current config
	provider := config.GetProvider(a.state.config.Provider)
	providerName := a.state.config.Provider
	if provider != nil {
		providerName = provider.Name
	}

	// Mask API key
	maskedKey := "Not set"
	if a.state.config.APIKey != "" {
		maskedKey = a.state.config.APIKey[:4] + "****" + a.state.config.APIKey[len(a.state.config.APIKey)-4:]
	}

	configLines := []string{
		fmt.Sprintf("  Provider: %s", providerName),
		fmt.Sprintf("  Model:    %s", a.state.config.Model),
		fmt.Sprintf("  API Key:  %s", maskedKey),
	}

	if a.state.config.Local != nil && a.state.config.Local.Enabled {
		configLines = append(configLines, "")
		configLines = append(configLines, "  Local Model:")
		configLines = append(configLines, fmt.Sprintf("    Provider: %s", a.state.config.Local.Provider))
		configLines = append(configLines, fmt.Sprintf("    Model:    %s", a.state.config.Local.Model))
	}

	configBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(configLines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, configBox))
	b.WriteString("\n\n")

	// Actions
	actions := []string{
		"  [p] Change provider",
		"  [m] Change model",
		"  [k] Update API key",
		"  [r] Reset setup",
	}
	actionsBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(actions, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, actionsBox))
	b.WriteString("\n\n")

	// Instructions
	instructions := styleStatusBar.Render("[Esc] Back")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}
```

---

### 7. Update go.mod

Add yaml dependency:

```
require (
	...
	gopkg.in/yaml.v3 v3.0.1
)
```

---

## Test

```bash
# Remove any existing config
rm -rf ~/.config/pulp

# Build and run
go build -o pulp ./cmd/pulp
./pulp

# Expected:
# 1. Setup wizard shows
# 2. Provider list navigable with j/k
# 3. Enter selects provider
# 4. If Groq/OpenAI/etc selected, API key screen shows
# 5. If Ollama selected, goes straight to welcome
# 6. Config saved at ~/.config/pulp/config.yaml

# Run again
./pulp
# Should go straight to welcome (setup complete)

# Press 's' for settings
# Settings view shows current config
```

---

## Done Checklist

- [ ] Config loads from `~/.config/pulp/config.yaml`
- [ ] First run shows setup wizard
- [ ] Provider selection with j/k navigation
- [ ] API key input for providers that need it
- [ ] Ollama skips API key
- [ ] Config saves correctly
- [ ] Subsequent runs skip setup
- [ ] Settings view accessible with `s`
- [ ] Esc returns from settings

---

## Commit Message

```
feat: add configuration system and setup wizard

- Add config package with YAML persistence
- Create first-run setup wizard
- Support provider selection (Ollama, Groq, OpenAI, Anthropic, OpenRouter)
- Add API key input with masking
- Add settings view
- Config persists at ~/.config/pulp/config.yaml
```
