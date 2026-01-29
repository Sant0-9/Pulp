package config

type ProviderInfo struct {
	ID           string
	Name         string
	Description  string
	NeedsAPIKey  bool
	SignupURL    string
	Models       []string
	DefaultModel string
}

var Providers = []ProviderInfo{
	{
		ID:           "ollama",
		Name:         "Ollama",
		Description:  "Local, free, private",
		NeedsAPIKey:  false,
		Models:       []string{"llama3.1:8b", "llama3.1:70b", "qwen2.5:7b", "mistral:7b"},
		DefaultModel: "llama3.1:8b",
	},
	{
		ID:           "groq",
		Name:         "Groq",
		Description:  "Very fast, cheap",
		NeedsAPIKey:  true,
		SignupURL:    "https://console.groq.com/keys",
		Models:       []string{"llama-3.1-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"},
		DefaultModel: "llama-3.1-70b-versatile",
	},
	{
		ID:           "openai",
		Name:         "OpenAI",
		Description:  "GPT-4o, most capable",
		NeedsAPIKey:  true,
		SignupURL:    "https://platform.openai.com/api-keys",
		Models:       []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo"},
		DefaultModel: "gpt-4o-mini",
	},
	{
		ID:           "anthropic",
		Name:         "Anthropic",
		Description:  "Claude, great writing",
		NeedsAPIKey:  true,
		SignupURL:    "https://console.anthropic.com/",
		Models:       []string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022"},
		DefaultModel: "claude-3-5-sonnet-20241022",
	},
	{
		ID:           "openrouter",
		Name:         "OpenRouter",
		Description:  "Access all models",
		NeedsAPIKey:  true,
		SignupURL:    "https://openrouter.ai/keys",
		Models:       []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o", "meta-llama/llama-3.1-70b"},
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
