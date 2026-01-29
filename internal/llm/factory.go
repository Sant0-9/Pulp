package llm

import (
	"fmt"

	"github.com/sant0-9/pulp/internal/config"
)

// NewProvider creates a provider from config
func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case "ollama":
		host := "http://localhost:11434"
		if cfg.BaseURL != "" {
			host = cfg.BaseURL
		}
		return NewOllamaProvider(host, cfg.Model), nil

	case "groq":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("groq requires an API key")
		}
		// Will be implemented in Phase 9
		return nil, fmt.Errorf("groq provider not yet implemented")

	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai requires an API key")
		}
		return nil, fmt.Errorf("openai provider not yet implemented")

	case "anthropic":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic requires an API key")
		}
		return nil, fmt.Errorf("anthropic provider not yet implemented")

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

// NewLocalProvider creates a provider for local extraction
func NewLocalProvider(cfg *config.Config) (Provider, error) {
	if cfg.Local == nil || !cfg.Local.Enabled {
		return nil, nil
	}

	switch cfg.Local.Provider {
	case "ollama":
		return NewOllamaProvider(cfg.Local.Host, cfg.Local.Model), nil
	default:
		return nil, fmt.Errorf("unknown local provider: %s", cfg.Local.Provider)
	}
}
