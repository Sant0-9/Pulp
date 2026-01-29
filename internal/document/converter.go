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
	// Get executable path for relative lookup
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)

	// Check various locations
	locations := []string{
		// Development: relative to binary
		filepath.Join(execDir, "python", "docling_bridge.py"),
		// Development: relative to working directory
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
