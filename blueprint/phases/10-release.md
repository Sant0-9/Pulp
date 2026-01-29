# Phase 10: Release

## Goal
Polish the app, add error handling, create release automation, write README. Ready for `brew install pulp`.

## Success Criteria
- All edge cases handled gracefully
- GoReleaser configured for multi-platform builds
- Homebrew formula ready
- README complete with examples
- Install script works

---

## Tasks

### 1. Error Handling Polish

Ensure all error states have good UX:

```
pulp/internal/tui/view_error.go
```

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderError() string {
	var b strings.Builder

	// Error icon and title
	title := lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true).
		Render("Something went wrong")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Error message
	errMsg := "Unknown error"
	if a.state.processingError != nil {
		errMsg = a.state.processingError.Error()
	} else if a.state.providerError != nil {
		errMsg = a.state.providerError.Error()
	} else if a.state.docError != nil {
		errMsg = a.state.docError.Error()
	}

	errBox := styleBox.Copy().
		Width(min(60, a.width-4)).
		BorderForeground(colorError).
		Render(errMsg)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, errBox))
	b.WriteString("\n\n")

	// Suggestions based on error type
	var suggestions []string
	errLower := strings.ToLower(errMsg)

	if strings.Contains(errLower, "api key") || strings.Contains(errLower, "401") {
		suggestions = append(suggestions, "Check your API key in ~/.config/pulp/config.yaml")
		suggestions = append(suggestions, "Or run setup again with [r] Reset")
	} else if strings.Contains(errLower, "connection") || strings.Contains(errLower, "connect") {
		suggestions = append(suggestions, "Check your internet connection")
		suggestions = append(suggestions, "Or try using Ollama for offline mode")
	} else if strings.Contains(errLower, "ollama") {
		suggestions = append(suggestions, "Make sure Ollama is running: ollama serve")
		suggestions = append(suggestions, "Or switch to a cloud provider with [p]")
	} else if strings.Contains(errLower, "not found") {
		suggestions = append(suggestions, "Check the file path is correct")
	}

	if len(suggestions) > 0 {
		suggBox := styleBox.Copy().
			Width(min(60, a.width-4)).
			BorderForeground(colorMuted).
			Render("Suggestions:\n" + strings.Join(suggestions, "\n"))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, suggBox))
		b.WriteString("\n\n")
	}

	// Actions
	status := styleStatusBar.Render("[r] Retry  [s] Settings  [n] New document  [Esc] Quit")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}
```

---

### 2. GoReleaser Configuration

```
pulp/.goreleaser.yaml
```

```yaml
project_name: pulp
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: pulp
    main: ./cmd/pulp
    binary: pulp
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE
      - python/*
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

brews:
  - name: pulp
    repository:
      owner: sant0-9
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    directory: Formula
    homepage: https://github.com/sant0-9/pulp
    description: "Document intelligence in your terminal"
    license: MIT
    install: |
      bin.install "pulp"
      # Install Python bridge
      (share/"pulp"/"python").install Dir["python/*"]
    test: |
      system "#{bin}/pulp", "--version"

scoops:
  - repository:
      owner: sant0-9
      name: scoop-bucket
      token: "{{ .Env.SCOOP_BUCKET_TOKEN }}"
    homepage: https://github.com/sant0-9/pulp
    description: "Document intelligence in your terminal"
    license: MIT

nfpms:
  - id: packages
    package_name: pulp
    vendor: Sant0-9
    homepage: https://github.com/sant0-9/pulp
    maintainer: Sant0-9
    description: "Document intelligence in your terminal"
    license: MIT
    formats:
      - deb
      - rpm
      - apk
    contents:
      - src: python/*
        dst: /usr/share/pulp/python/

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"

release:
  github:
    owner: sant0-9
    name: pulp
  draft: false
  prerelease: auto
```

---

### 3. GitHub Actions Workflow

```
pulp/.github/workflows/release.yml
```

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

---

### 4. Install Script

```
pulp/scripts/install.sh
```

```bash
#!/bin/sh
set -e

REPO="sant0-9/pulp"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "${GREEN}Installing Pulp...${NC}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

# Get latest version
echo "Fetching latest version..."
VERSION=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "${RED}Failed to fetch latest version${NC}"
    exit 1
fi

echo "Latest version: $VERSION"

# Download
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    EXT="zip"
fi

FILENAME="pulp_${VERSION#v}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "Downloading $URL..."
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

if ! curl -fsSL "$URL" -o "$FILENAME"; then
    echo "${RED}Failed to download $URL${NC}"
    exit 1
fi

# Extract
echo "Extracting..."
if [ "$EXT" = "zip" ]; then
    unzip -q "$FILENAME"
else
    tar xzf "$FILENAME"
fi

# Install
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv pulp "$INSTALL_DIR/"
else
    sudo mv pulp "$INSTALL_DIR/"
fi

# Install Python bridge
PULP_SHARE="${HOME}/.local/share/pulp"
mkdir -p "$PULP_SHARE/python"
if [ -d "python" ]; then
    cp -r python/* "$PULP_SHARE/python/"
fi

# Cleanup
cd /
rm -rf "$TMP_DIR"

# Verify
if command -v pulp >/dev/null 2>&1; then
    echo "${GREEN}Pulp installed successfully!${NC}"
    echo ""
    pulp --version
    echo ""
    echo "Run 'pulp' to get started."
else
    echo "${YELLOW}Pulp installed but not in PATH.${NC}"
    echo "Add $INSTALL_DIR to your PATH."
fi
```

---

### 5. README

```
pulp/README.md
```

```markdown
# Pulp

Document intelligence in your terminal. Type `pulp` and talk to your documents.

## Install

```bash
# macOS
brew install sant0-9/tap/pulp

# Linux / macOS / Windows
curl -fsSL https://raw.githubusercontent.com/sant0-9/pulp/main/scripts/install.sh | sh

# From source
go install github.com/sant0-9/pulp/cmd/pulp@latest
```

## Quick Start

```bash
pulp
```

That's it. Drop a file, tell it what you want in plain English.

## Examples

```
> research-paper.pdf
> summarize the key findings for my boss

> meeting-notes.md
> what are my action items

> technical-spec.docx
> explain this like I'm 5

> quarterly-report.pdf
> bullet points of the main risks
```

## Features

- **Natural language** - No commands to memorize
- **Any document** - PDF, DOCX, PPTX, MD, TXT, HTML
- **Follow-ups** - "make it shorter", "add more detail"
- **Streaming output** - See results as they're generated
- **BYOK** - Bring your own LLM API key
- **Offline mode** - Use Ollama for local processing
- **Beautiful TUI** - Full-screen terminal interface

## Providers

| Provider | Speed | Cost | Setup |
|----------|-------|------|-------|
| Ollama | Varies | Free | Local, private |
| Groq | Very fast | Cheap | [Get API key](https://console.groq.com/keys) |
| OpenAI | Fast | Medium | [Get API key](https://platform.openai.com/api-keys) |
| Anthropic | Fast | Medium | [Get API key](https://console.anthropic.com/) |
| OpenRouter | Varies | Varies | [Get API key](https://openrouter.ai/keys) |

## Configuration

Config file: `~/.config/pulp/config.yaml`

```yaml
provider: groq
api_key: gsk_your_key_here
model: llama-3.1-70b-versatile
```

Or use environment variables:
```bash
export GROQ_API_KEY="gsk_..."
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Enter | Submit |
| Esc | Quit / Cancel |
| c | Copy result |
| s | Save to file |
| n | New document |
| ? | Help |

## Requirements

- **Python 3.9+** with `docling` for document parsing
- **Ollama** (optional) for local LLM

```bash
pip install docling
```

## Privacy

- Your documents are processed by your chosen LLM provider
- No telemetry or data collection
- Use Ollama for fully local, offline processing
- API keys stored locally in `~/.config/pulp/`

## License

MIT
```

---

### 6. Version in main.go

```go
// cmd/pulp/main.go

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
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("pulp %s (%s) built %s\n", version, commit, date)
		return
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
```

---

### 7. Makefile

```
pulp/Makefile
```

```makefile
.PHONY: build install clean test lint release

VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/pulp ./cmd/pulp

install: build
	cp bin/pulp /usr/local/bin/
	mkdir -p ~/.local/share/pulp/python
	cp -r python/* ~/.local/share/pulp/python/

clean:
	rm -rf bin/
	rm -rf dist/

test:
	go test ./...

lint:
	golangci-lint run

release:
	goreleaser release --snapshot --clean
```

---

### 8. LICENSE

```
pulp/LICENSE
```

```
MIT License

Copyright (c) 2024 Sant0-9

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## Test Release

```bash
# Test local build
make build
./bin/pulp --version

# Test release (snapshot, no publish)
make release

# Check dist/ folder for built artifacts
ls dist/

# Test install script locally
./scripts/install.sh
```

---

## Actual Release

```bash
# Tag version
git tag v0.1.0

# Push tag (triggers GitHub Actions)
git push origin v0.1.0

# GitHub Actions will:
# - Build for all platforms
# - Create GitHub release
# - Update Homebrew tap
# - Update Scoop bucket
```

---

## Done Checklist

- [ ] Error view shows helpful suggestions
- [ ] GoReleaser builds for all platforms
- [ ] GitHub Actions workflow works
- [ ] Install script works
- [ ] README is complete
- [ ] LICENSE added
- [ ] --version flag works
- [ ] Makefile has all targets

---

## Commit Message

```
feat: prepare for v1.0.0 release

- Add polished error handling with suggestions
- Configure GoReleaser for multi-platform builds
- Add GitHub Actions release workflow
- Create install script for curl installation
- Write comprehensive README
- Add MIT LICENSE
- Add version flag and build info
```
