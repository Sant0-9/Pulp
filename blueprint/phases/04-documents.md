# Phase 4: Document Loading

## Goal
Create Python bridge for Docling and document loading UI. User can input file path, document gets parsed and preview shown.

## Success Criteria
- Python bridge script works standalone
- Go can call Python and parse JSON output
- File path input works in TUI
- Document loads and shows metadata
- Preview of document content shown

---

## Files to Create

### 1. Python Bridge

```
pulp/python/docling_bridge.py
```

```python
#!/usr/bin/env python3
"""Bridge between Go and Docling for document conversion."""

import sys
import json
from pathlib import Path
from datetime import datetime

try:
    from docling.document_converter import DocumentConverter
except ImportError:
    print(json.dumps({
        "success": False,
        "error": "Docling not installed. Run: pip install docling"
    }))
    sys.exit(1)


def convert(path: str) -> dict:
    """Convert document and return structured data."""
    p = Path(path)

    if not p.exists():
        return {"success": False, "error": f"File not found: {path}"}

    try:
        converter = DocumentConverter()
        result = converter.convert(str(p))
        doc = result.document

        # Export to markdown
        markdown = doc.export_to_markdown()

        # Get metadata
        metadata = {
            "title": getattr(doc, 'title', None) or p.stem,
            "source_path": str(p.absolute()),
            "source_format": p.suffix.lower().lstrip('.'),
            "file_size_bytes": p.stat().st_size,
            "converted_at": datetime.now().isoformat(),
        }

        # Try to get page count
        if hasattr(doc, 'pages'):
            metadata["page_count"] = len(doc.pages)

        # Word count estimate
        metadata["word_count"] = len(markdown.split())

        # Get preview (first 500 chars)
        preview = markdown[:500].strip()
        if len(markdown) > 500:
            preview += "..."

        return {
            "success": True,
            "markdown": markdown,
            "metadata": metadata,
            "preview": preview,
        }

    except Exception as e:
        return {"success": False, "error": str(e)}


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"success": False, "error": "Usage: docling_bridge.py <file_path>"}))
        sys.exit(1)

    result = convert(sys.argv[1])
    print(json.dumps(result))

    if not result["success"]:
        sys.exit(1)


if __name__ == "__main__":
    main()
```

```
pulp/python/requirements.txt
```

```
docling>=2.0.0
```

---

### 2. Document Types

```
pulp/internal/document/document.go
```

```go
package document

import "time"

// Document represents a parsed document
type Document struct {
	Content   string
	Preview   string
	Metadata  Metadata
}

// Metadata contains document metadata
type Metadata struct {
	Title        string    `json:"title"`
	SourcePath   string    `json:"source_path"`
	SourceFormat string    `json:"source_format"`
	FileSizeBytes int64    `json:"file_size_bytes"`
	PageCount    *int      `json:"page_count,omitempty"`
	WordCount    int       `json:"word_count"`
	ConvertedAt  time.Time `json:"converted_at"`
}

// FileSizeHuman returns human-readable file size
func (m Metadata) FileSizeHuman() string {
	bytes := m.FileSizeBytes
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
```

Add this import at the top:
```go
package document

import (
	"fmt"
	"time"
)
```

---

### 3. Converter (Python Bridge)

```
pulp/internal/document/converter.go
```

```go
package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Converter handles document conversion via Docling
type Converter struct {
	pythonPath string
	scriptPath string
	timeout    time.Duration
}

// NewConverter creates a new document converter
func NewConverter() (*Converter, error) {
	// Find Python
	pythonPath, err := findPython()
	if err != nil {
		return nil, err
	}

	// Find script
	scriptPath, err := findScript()
	if err != nil {
		return nil, err
	}

	return &Converter{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
		timeout:    5 * time.Minute,
	}, nil
}

func findPython() (string, error) {
	// Try python3 first, then python
	for _, name := range []string{"python3", "python"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("python not found in PATH")
}

func findScript() (string, error) {
	// Check various locations
	locations := []string{
		// Development: relative to binary
		"python/docling_bridge.py",
		// Installed: in config dir
		filepath.Join(os.Getenv("HOME"), ".config", "pulp", "python", "docling_bridge.py"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			abs, _ := filepath.Abs(loc)
			return abs, nil
		}
	}

	return "", fmt.Errorf("docling_bridge.py not found")
}

// Convert converts a document to markdown
func (c *Converter) Convert(ctx context.Context, path string) (*Document, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Check file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	// Run Python script
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, absPath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Try to parse error from stdout
			var result struct {
				Success bool   `json:"success"`
				Error   string `json:"error"`
			}
			if json.Unmarshal(output, &result) == nil && result.Error != "" {
				return nil, fmt.Errorf("%s", result.Error)
			}
			return nil, fmt.Errorf("conversion failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run converter: %w", err)
	}

	// Parse result
	var result struct {
		Success  bool     `json:"success"`
		Error    string   `json:"error,omitempty"`
		Markdown string   `json:"markdown"`
		Preview  string   `json:"preview"`
		Metadata Metadata `json:"metadata"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse converter output: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("%s", result.Error)
	}

	return &Document{
		Content:  result.Markdown,
		Preview:  result.Preview,
		Metadata: result.Metadata,
	}, nil
}
```

---

### 4. Update State

Add to `pulp/internal/tui/state.go`:

```go
import (
	"github.com/sant0-9/pulp/internal/document"
)

type state struct {
	// ... existing fields ...

	// Document
	document     *document.Document
	documentPath string
	loadingDoc   bool
	docError     error
}
```

---

### 5. Document Messages

Add to `pulp/internal/tui/app.go`:

```go
import (
	"github.com/sant0-9/pulp/internal/document"
)

type documentLoadedMsg struct {
	doc *document.Document
}

type documentErrorMsg struct {
	error
}
```

---

### 6. Handle File Input

Update `handleKey` in app.go to handle Enter on welcome view:

```go
func (a *App) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keys.Quit):
		// ... existing quit handling ...

	case key.Matches(msg, keys.Enter):
		if a.view == viewWelcome && a.state.providerReady {
			input := strings.TrimSpace(a.state.input.Value())
			if input != "" {
				a.state.loadingDoc = true
				a.state.documentPath = input
				return a.loadDocument(input)
			}
		}
	}
	// ... rest of handler ...
}

func (a *App) loadDocument(path string) tea.Cmd {
	return func() tea.Msg {
		converter, err := document.NewConverter()
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
```

Handle document messages in Update():

```go
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
```

---

### 7. Document View

```
pulp/internal/tui/view_document.go
```

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderDocument() string {
	if a.state.document == nil {
		return a.renderWelcome()
	}

	var b strings.Builder
	doc := a.state.document
	meta := doc.Metadata

	// Document info header
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(meta.Title)

	// Metadata line
	var metaParts []string
	if meta.PageCount != nil {
		metaParts = append(metaParts, fmt.Sprintf("%d pages", *meta.PageCount))
	}
	metaParts = append(metaParts, strings.ToUpper(meta.SourceFormat))
	metaParts = append(metaParts, meta.FileSizeHuman())
	metaParts = append(metaParts, fmt.Sprintf("~%d words", meta.WordCount))

	metaLine := styleSubtitle.Render(strings.Join(metaParts, "  |  "))

	// Document info box
	infoContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		metaLine,
	)
	infoBox := styleBox.Copy().
		Width(min(70, a.width-4)).
		BorderForeground(colorSuccess).
		Render(infoContent)

	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, infoBox))
	b.WriteString("\n\n")

	// Preview
	previewLabel := styleSubtitle.Render("Preview:")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, previewLabel))
	b.WriteString("\n")

	previewBox := styleBox.Copy().
		Width(min(70, a.width-4)).
		Foreground(colorMuted).
		Render(doc.Preview)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, previewBox))
	b.WriteString("\n\n")

	// Instruction prompt
	promptLabel := lipgloss.NewStyle().
		Foreground(colorWhite).
		Render("What do you want to do with this document?")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, promptLabel))
	b.WriteString("\n\n")

	// Input
	inputBox := styleBox.Copy().
		Width(min(70, a.width-4)).
		BorderForeground(colorSecondary).
		Render(a.state.input.View())
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
	b.WriteString("\n\n")

	// Status bar
	status := styleStatusBar.Render("[Enter] Submit  [n] New document  [Esc] Quit")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

---

### 8. Update View Switch

In app.go View():

```go
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
	case viewSettings:
		return a.renderSettings()
	default:
		return a.renderWelcome()
	}
}
```

---

### 9. Handle 'n' for New Document

In handleKey:

```go
case msg.String() == "n":
	if a.view == viewDocument || a.view == viewResult {
		a.state.document = nil
		a.state.documentPath = ""
		a.state.input.Reset()
		a.state.input.Placeholder = "Drop a file or type a path..."
		a.view = viewWelcome
		return a, nil
	}
```

---

## Test

```bash
# Install Python dependencies
pip install docling

# Create test file
echo "# Test Document\n\nThis is a test." > /tmp/test.md

# Build and run
go build -o pulp ./cmd/pulp
./pulp

# Type: /tmp/test.md
# Press Enter

# Expected:
# - Document loads
# - Shows title, format, size
# - Shows preview
# - Prompt asks what to do
```

---

## Done Checklist

- [ ] Python bridge script works: `python python/docling_bridge.py /path/to/file.pdf`
- [ ] Go converter calls Python successfully
- [ ] File path input works in TUI
- [ ] Document metadata displayed
- [ ] Preview shown
- [ ] Input prompt for instructions shown
- [ ] 'n' returns to welcome for new document
- [ ] Error handling for missing files

---

## Commit Message

```
feat: add document loading with Docling

- Create Python bridge for Docling document conversion
- Add Go converter that calls Python subprocess
- Display document metadata and preview in TUI
- Support PDF, DOCX, MD, TXT via Docling
- Handle file not found and conversion errors
```
