# Phase 1: Foundation

## Goal
Create the Go project structure and a working TUI that launches full-screen and responds to keyboard input.

## Success Criteria
- `go build` produces working binary
- `./pulp` opens full-screen TUI with logo
- `Esc` or `Ctrl+C` quits cleanly
- Window resize works

---

## Files to Create

### 1. go.mod

```
pulp/go.mod
```

```go
module github.com/sant0-9/pulp

go 1.22

require (
	github.com/charmbracelet/bubbletea v1.2.4
	github.com/charmbracelet/bubbles v0.20.0
	github.com/charmbracelet/lipgloss v1.0.0
)
```

Run `go mod tidy` after creating.

---

### 2. Main Entry Point

```
pulp/cmd/pulp/main.go
```

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sant0-9/pulp/internal/tui"
)

var version = "dev"

func main() {
	p := tea.NewProgram(
		tui.NewApp(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

---

### 3. TUI App Model

```
pulp/internal/tui/app.go
```

```go
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type view int

const (
	viewWelcome view = iota
	viewDocument
	viewProcessing
	viewResult
)

type App struct {
	width    int
	height   int
	view     view
	quitting bool
}

func NewApp() *App {
	return &App{
		view: viewWelcome,
	}
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			a.quitting = true
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
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
	default:
		return a.renderWelcome()
	}
}
```

---

### 4. Key Bindings

```
pulp/internal/tui/keys.go
```

```go
package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit   key.Binding
	Help   key.Binding
	Enter  key.Binding
	Up     key.Binding
	Down   key.Binding
	Tab    key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "down"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next"),
	),
}
```

---

### 5. Styles

```
pulp/internal/tui/styles.go
```

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#06B6D4")
	colorSuccess   = lipgloss.Color("#10B981")
	colorError     = lipgloss.Color("#EF4444")
	colorMuted     = lipgloss.Color("#6B7280")
	colorWhite     = lipgloss.Color("#F9FAFB")
	colorDark      = lipgloss.Color("#1F2937")

	// Logo style
	styleLogo = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Subtitle
	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Box
	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	// Status bar
	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Centered container
	styleCenter = lipgloss.NewStyle().
			Align(lipgloss.Center)
)
```

---

### 6. Welcome View

```
pulp/internal/tui/view_welcome.go
```

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const logo = `
 ██████╗ ██╗   ██╗██╗     ██████╗
 ██╔══██╗██║   ██║██║     ██╔══██╗
 ██████╔╝██║   ██║██║     ██████╔╝
 ██╔═══╝ ██║   ██║██║     ██╔═══╝
 ██║     ╚██████╔╝███████╗██║
 ╚═╝      ╚═════╝ ╚══════╝╚═╝
`

func (a *App) renderWelcome() string {
	var b strings.Builder

	// Logo
	logoRendered := styleLogo.Render(logo)

	// Subtitle
	subtitle := styleSubtitle.Render("Document Intelligence")

	// Instructions
	instructions := styleSubtitle.Render("\nDrop a file or type a path to get started")

	// Status bar
	statusBar := styleStatusBar.Render("[Esc] Quit  [?] Help")

	// Combine
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logoRendered,
		subtitle,
		instructions,
	)

	// Center on screen
	contentHeight := strings.Count(content, "\n") + 1
	topPadding := (a.height - contentHeight - 2) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	content = strings.Repeat("\n", topPadding) + content

	// Add status bar at bottom
	bottomPadding := a.height - strings.Count(content, "\n") - 3
	if bottomPadding < 0 {
		bottomPadding = 0
	}
	content += strings.Repeat("\n", bottomPadding)
	content += "\n" + lipgloss.PlaceHorizontal(a.width, lipgloss.Center, statusBar)

	return lipgloss.PlaceHorizontal(a.width, lipgloss.Center, content)
}
```

---

## Directory Structure After Phase 1

```
pulp/
├── cmd/
│   └── pulp/
│       └── main.go
├── internal/
│   └── tui/
│       ├── app.go
│       ├── keys.go
│       ├── styles.go
│       └── view_welcome.go
├── go.mod
├── go.sum
└── blueprint/
    ├── ARCHITECTURE.md
    ├── BRAINSTORM.md
    ├── TRACKER.md
    └── phases/
        └── 01-foundation.md
```

---

## Build & Test

```bash
# Initialize module and get dependencies
go mod tidy

# Build
go build -o pulp ./cmd/pulp

# Run
./pulp

# Expected behavior:
# - Full-screen TUI opens
# - Purple PULP logo centered
# - "Document Intelligence" subtitle
# - Status bar at bottom
# - Esc quits cleanly
# - Window resize re-centers content
```

---

## Done Checklist

- [ ] `go mod tidy` succeeds
- [ ] `go build ./cmd/pulp` produces binary
- [ ] Binary launches full-screen TUI
- [ ] Logo displays centered
- [ ] Esc quits without error
- [ ] Ctrl+C quits without error
- [ ] Window resize works
- [ ] No panics or errors

---

## Commit Message

```
feat: initialize project with basic TUI shell

- Set up Go module with Bubbletea, Bubbles, Lipgloss
- Create full-screen TUI with welcome view
- Add keyboard handling (Esc to quit)
- Add centered logo and status bar
```
