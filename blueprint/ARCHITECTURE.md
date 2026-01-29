# Pulp Architecture Document

## Overview

Pulp is an interactive document processing TUI. Type `pulp` and you're in.

**Open source. BYOK (Bring Your Own Key). Works with any LLM.**

---

## Distribution

### Package Managers

```bash
# macOS
brew install pulp

# Linux (Debian/Ubuntu)
sudo apt install pulp

# Linux (Arch)
yay -S pulp

# Windows
scoop install pulp
winget install pulp

# Cross-platform
go install github.com/sant0-9/pulp@latest

# Or download binary
curl -fsSL https://pulp.dev/install.sh | sh
```

### Single Binary

- **No runtime dependencies** (Python bundled or optional)
- Works offline with local models
- ~15MB binary

---

## First Run Setup

```
$ pulp

╭──────────────────────────────────────────────────────────────────────────────╮
│                                                                              │
│     ██████╗ ██╗   ██╗██╗     ██████╗                                        │
│     ██╔══██╗██║   ██║██║     ██╔══██╗                                       │
│     ██████╔╝██║   ██║██║     ██████╔╝                                       │
│     ██╔═══╝ ██║   ██║██║     ██╔═══╝                                        │
│     ██║     ╚██████╔╝███████╗██║                                            │
│     ╚═╝      ╚═════╝ ╚══════╝╚═╝                                            │
│                                                                              │
╰──────────────────────────────────────────────────────────────────────────────╯

  Welcome! Let's set up your LLM provider.

  Which provider do you want to use?

  > [x] Groq        (fast, cheap, recommended)
    [ ] OpenAI      (GPT-4o, GPT-4o-mini)
    [ ] Anthropic   (Claude 3.5 Sonnet)
    [ ] Ollama      (local, free, private)
    [ ] OpenRouter  (access to all models)
    [ ] Custom      (any OpenAI-compatible API)

  [Enter] Select   [j/k] Navigate   [?] Compare providers
```

### API Key Entry

```
  You selected: Groq

  Get your API key at: https://console.groq.com/keys

  Paste your API key:
  > ****************************************

  Testing connection... Connected!

  Want to set up a local model for faster processing? (optional)

  [ ] Yes, I have Ollama installed
  [x] No, use cloud for everything

  Setup complete! Your config is saved at ~/.config/pulp/config.yaml

  Press [Enter] to start using Pulp
```

---

## BYOK - Provider Support

### Supported Providers

| Provider | Models | Speed | Cost | Notes |
|----------|--------|-------|------|-------|
| **Groq** | Llama 3.1, Mixtral | Very Fast | Cheap | Recommended for speed |
| **OpenAI** | GPT-4o, GPT-4o-mini | Fast | Medium | Most capable |
| **Anthropic** | Claude 3.5 Sonnet | Fast | Medium | Great for writing |
| **Ollama** | Llama, Qwen, Mistral | Varies | Free | Local, private |
| **OpenRouter** | All models | Varies | Varies | One API, all models |
| **Together** | Open source models | Fast | Cheap | Good Llama hosting |
| **Fireworks** | Open source models | Very Fast | Cheap | Fast inference |
| **Custom** | Any | - | - | OpenAI-compatible endpoint |

### Configuration

```yaml
# ~/.config/pulp/config.yaml

# Main provider (for writing/synthesis)
provider: groq
api_key: ${GROQ_API_KEY}  # Or paste directly
model: llama-3.1-70b-versatile

# Local model (for extraction, optional but faster)
local:
  enabled: true
  provider: ollama
  host: http://localhost:11434
  model: qwen2.5:7b

# Alternative configs:

# OpenAI
# provider: openai
# api_key: ${OPENAI_API_KEY}
# model: gpt-4o-mini

# Anthropic
# provider: anthropic
# api_key: ${ANTHROPIC_API_KEY}
# model: claude-3-5-sonnet-20241022

# Ollama (fully local, no API key needed)
# provider: ollama
# host: http://localhost:11434
# model: llama3.1:8b

# OpenRouter (access any model)
# provider: openrouter
# api_key: ${OPENROUTER_API_KEY}
# model: anthropic/claude-3.5-sonnet

# Custom OpenAI-compatible endpoint
# provider: custom
# base_url: https://my-company.com/v1
# api_key: ${CUSTOM_API_KEY}
# model: our-internal-model
```

### Environment Variables

```bash
# Set once, use forever
export GROQ_API_KEY="gsk_..."
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Or in ~/.bashrc / ~/.zshrc
```

---

## Provider Abstraction

```go
// internal/llm/provider.go

type Provider interface {
    // Chat completion
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

    // Streaming completion
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, error)

    // Health check
    Ping(ctx context.Context) error

    // Provider info
    Name() string
    Models() []string
}

type CompletionRequest struct {
    Model       string
    Messages    []Message
    MaxTokens   int
    Temperature float64
    Stream      bool
}

type Message struct {
    Role    string
    Content string
}

type CompletionResponse struct {
    Content      string
    Model        string
    TokensUsed   int
    FinishReason string
}

type StreamEvent struct {
    Chunk string
    Done  bool
    Error error
}
```

### Provider Implementations

```go
// internal/llm/providers/groq.go
type GroqProvider struct { ... }

// internal/llm/providers/openai.go
type OpenAIProvider struct { ... }

// internal/llm/providers/anthropic.go
type AnthropicProvider struct { ... }

// internal/llm/providers/ollama.go
type OllamaProvider struct { ... }

// internal/llm/providers/openrouter.go
type OpenRouterProvider struct { ... }

// internal/llm/providers/custom.go
type CustomProvider struct { ... }  // Any OpenAI-compatible API
```

### Factory

```go
// internal/llm/factory.go

func NewProvider(cfg Config) (Provider, error) {
    switch cfg.Provider {
    case "groq":
        return NewGroqProvider(cfg.APIKey, cfg.Model)
    case "openai":
        return NewOpenAIProvider(cfg.APIKey, cfg.Model)
    case "anthropic":
        return NewAnthropicProvider(cfg.APIKey, cfg.Model)
    case "ollama":
        return NewOllamaProvider(cfg.Host, cfg.Model)
    case "openrouter":
        return NewOpenRouterProvider(cfg.APIKey, cfg.Model)
    case "custom":
        return NewCustomProvider(cfg.BaseURL, cfg.APIKey, cfg.Model)
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              PULP                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                         GO BINARY                                   │  │
│   │                                                                     │  │
│   │   ┌──────────────────────────────────────────────────────────┐     │  │
│   │   │                      TUI (Bubbletea)                      │     │  │
│   │   └──────────────────────────────────────────────────────────┘     │  │
│   │                              │                                      │  │
│   │                              ▼                                      │  │
│   │   ┌──────────────────────────────────────────────────────────┐     │  │
│   │   │                      PIPELINE                             │     │  │
│   │   │   Convert ──▶ Chunk ──▶ Extract ──▶ Write                │     │  │
│   │   └──────────────────────────────────────────────────────────┘     │  │
│   │                              │                                      │  │
│   │                              ▼                                      │  │
│   │   ┌──────────────────────────────────────────────────────────┐     │  │
│   │   │               PROVIDER ABSTRACTION                        │     │  │
│   │   │                                                           │     │  │
│   │   │   Provider Interface:                                     │     │  │
│   │   │   - Complete(request) -> response                         │     │  │
│   │   │   - Stream(request) -> channel                            │     │  │
│   │   │                                                           │     │  │
│   │   └──────────────────────────────────────────────────────────┘     │  │
│   │                              │                                      │  │
│   └──────────────────────────────┼──────────────────────────────────────┘  │
│                                  │                                         │
│   ┌──────────────────────────────┼──────────────────────────────────────┐  │
│   │                      PROVIDERS (User's Choice)                       │  │
│   │                                                                      │  │
│   │   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐      │  │
│   │   │  Groq   │ │ OpenAI  │ │Anthropic│ │ Ollama  │ │ Custom  │      │  │
│   │   │         │ │         │ │         │ │ (local) │ │         │      │  │
│   │   └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘      │  │
│   │                                                                      │  │
│   └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Project Structure

```
pulp/
├── cmd/
│   └── pulp/
│       └── main.go
│
├── internal/
│   ├── tui/
│   │   ├── app.go
│   │   ├── views/
│   │   │   ├── welcome.go
│   │   │   ├── setup.go         # First-run setup wizard
│   │   │   ├── document.go
│   │   │   ├── processing.go
│   │   │   └── result.go
│   │   ├── components/
│   │   └── styles/
│   │
│   ├── llm/
│   │   ├── provider.go          # Provider interface
│   │   ├── factory.go           # Provider factory
│   │   └── providers/
│   │       ├── groq.go
│   │       ├── openai.go
│   │       ├── anthropic.go
│   │       ├── ollama.go
│   │       ├── openrouter.go
│   │       └── custom.go
│   │
│   ├── pipeline/
│   │   ├── pipeline.go
│   │   └── session.go
│   │
│   ├── converter/
│   │   └── docling.go
│   │
│   ├── intent/
│   │   └── parser.go
│   │
│   └── config/
│       ├── config.go
│       ├── loader.go
│       └── setup.go             # First-run setup logic
│
├── python/
│   └── docling_bridge.py
│
├── scripts/
│   └── install.sh               # curl installer
│
├── .goreleaser.yaml             # Release automation
├── go.mod
└── Makefile
```

---

## Release Automation

### GoReleaser Config

```yaml
# .goreleaser.yaml

project_name: pulp

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
      - -X main.commit={{.Commit}}

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

brews:
  - name: pulp
    repository:
      owner: sant0-9
      name: homebrew-tap
    homepage: https://github.com/sant0-9/pulp
    description: "Document intelligence in your terminal"
    install: |
      bin.install "pulp"

scoops:
  - repository:
      owner: sant0-9
      name: scoop-bucket
    homepage: https://github.com/sant0-9/pulp
    description: "Document intelligence in your terminal"

nfpms:
  - id: packages
    package_name: pulp
    vendor: Sant0-9
    homepage: https://github.com/sant0-9/pulp
    maintainer: Sant0-9
    description: "Document intelligence in your terminal"
    formats:
      - deb
      - rpm
      - apk

snapcrafts:
  - name: pulp
    summary: Document intelligence in your terminal
    description: |
      Pulp is an interactive TUI for processing documents with AI.
      Bring your own LLM API key and start summarizing, rewriting,
      and extracting insights from any document.
    grade: stable
    confinement: strict

aurs:
  - name: pulp-bin
    homepage: https://github.com/sant0-9/pulp
    description: "Document intelligence in your terminal"
    maintainers:
      - "Sant0-9"

checksum:
  name_template: "checksums.txt"

release:
  github:
    owner: sant0-9
    name: pulp
```

### Install Script

```bash
#!/bin/sh
# scripts/install.sh - curl -fsSL https://pulp.dev/install.sh | sh

set -e

REPO="sant0-9/pulp"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# Download
URL="https://github.com/$REPO/releases/download/$VERSION/pulp_${VERSION#v}_${OS}_${ARCH}.tar.gz"
echo "Downloading Pulp $VERSION..."
curl -fsSL "$URL" | tar xz -C /tmp

# Install
echo "Installing to $INSTALL_DIR..."
sudo mv /tmp/pulp "$INSTALL_DIR/pulp"
sudo chmod +x "$INSTALL_DIR/pulp"

echo "Pulp installed successfully!"
echo "Run 'pulp' to get started."
```

---

## Settings Command

In-app settings accessible anytime:

```
$ pulp

  [Press 's' for settings]

╭─ Settings ───────────────────────────────────────────────────────────────────╮
│                                                                              │
│  Provider: groq                                                              │
│  Model: llama-3.1-70b-versatile                                             │
│  API Key: gsk_****************************                                   │
│                                                                              │
│  Local model: ollama (qwen2.5:7b)                                           │
│                                                                              │
│  ────────────────────────────────────────────────────────────────────────   │
│                                                                              │
│  [p] Change provider                                                         │
│  [m] Change model                                                            │
│  [k] Update API key                                                          │
│  [l] Configure local model                                                   │
│  [r] Reset to defaults                                                       │
│                                                                              │
│  [Esc] Back                                                                  │
│                                                                              │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Offline Mode

If user has Ollama, works fully offline:

```yaml
# ~/.config/pulp/config.yaml

# Fully local - no internet required
provider: ollama
host: http://localhost:11434
model: llama3.1:8b

local:
  enabled: true
  provider: ollama
  model: qwen2.5:3b  # Smaller model for extraction
```

```
$ pulp

  Mode: Offline (Ollama)
  Model: llama3.1:8b

  > research.pdf
  > summarize this

  [Works without internet]
```

---

## Error Handling

### No API Key

```
╭─ Setup Required ─────────────────────────────────────────────────────────────╮
│                                                                              │
│  No API key configured.                                                      │
│                                                                              │
│  Options:                                                                    │
│  1. Get a free API key from Groq: https://console.groq.com/keys             │
│  2. Use Ollama for fully local processing (no API key needed)               │
│                                                                              │
│  [Enter] Run setup wizard                                                    │
│  [o] Use Ollama (local)                                                      │
│  [Esc] Quit                                                                  │
│                                                                              │
╰──────────────────────────────────────────────────────────────────────────────╯
```

### Invalid API Key

```
╭─ Connection Error ───────────────────────────────────────────────────────────╮
│                                                                              │
│  Could not connect to Groq API.                                              │
│                                                                              │
│  Error: Invalid API key (401 Unauthorized)                                   │
│                                                                              │
│  [k] Enter new API key                                                       │
│  [p] Switch provider                                                         │
│  [Esc] Back                                                                  │
│                                                                              │
╰──────────────────────────────────────────────────────────────────────────────╯
```

### Rate Limited

```
╭─ Rate Limited ───────────────────────────────────────────────────────────────╮
│                                                                              │
│  You've hit the rate limit for Groq free tier.                              │
│                                                                              │
│  Options:                                                                    │
│  - Wait 60 seconds and try again                                            │
│  - Upgrade your Groq plan                                                   │
│  - Switch to a different provider                                           │
│                                                                              │
│  [r] Retry in 60s                                                           │
│  [p] Switch provider                                                         │
│  [Esc] Back                                                                  │
│                                                                              │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Provider Comparison (In-App)

Press `?` during provider selection:

```
╭─ Provider Comparison ────────────────────────────────────────────────────────╮
│                                                                              │
│  Provider     Speed       Cost          Best For                            │
│  ─────────────────────────────────────────────────────────────────────────  │
│  Groq         Very Fast   $0.05/M in    Speed, budget                       │
│  OpenAI       Fast        $0.15/M in    Quality, features                   │
│  Anthropic    Fast        $0.25/M in    Writing quality                     │
│  Ollama       Varies      Free          Privacy, offline                    │
│  OpenRouter   Varies      Varies        Access to all models                │
│                                                                              │
│  Recommendation:                                                             │
│  - Start with Groq (fast & cheap)                                           │
│  - Use Ollama if privacy matters                                            │
│  - Use Anthropic for best writing                                           │
│                                                                              │
│  [Esc] Back                                                                  │
│                                                                              │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Privacy

- **API keys stored locally** in `~/.config/pulp/config.yaml`
- **No telemetry** - zero data sent to us
- **Documents processed** via user's chosen provider
- **Ollama option** for fully local, air-gapped usage
- **Open source** - audit the code yourself

---

## Summary

| Feature | Implementation |
|---------|----------------|
| Distribution | Homebrew, apt, scoop, AUR, curl script |
| BYOK | User brings any API key |
| Providers | Groq, OpenAI, Anthropic, Ollama, OpenRouter, Custom |
| First run | Interactive setup wizard |
| Offline | Full Ollama support |
| Config | `~/.config/pulp/config.yaml` |
| Privacy | No telemetry, local keys, open source |

**Install. Bring your key. Start using.**

```bash
brew install pulp
pulp
```
