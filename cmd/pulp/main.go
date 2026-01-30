package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sant0-9/pulp/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Handle flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("pulp %s (%s) built %s\n", version, commit, date)
			return
		case "--help", "-h", "help":
			printHelp()
			return
		}
	}

	app := tui.NewApp()
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	app.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Pulp - Document Intelligence for the Terminal

Usage:
  pulp [flags]
  pulp [file]

Flags:
  -h, --help      Show this help
  -v, --version   Show version

Examples:
  pulp                    Start interactive mode
  pulp document.pdf       Open with a document

For more info: https://github.com/sant0-9/pulp`)
}
