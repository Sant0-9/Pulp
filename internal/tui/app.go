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
	return tea.WindowSize()
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
