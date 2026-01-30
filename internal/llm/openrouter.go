package llm

import (
	"net/http"
	"time"
)

type OpenRouterProvider struct {
	*OpenAIProvider
}

func NewOpenRouterProvider(apiKey, model string) *OpenRouterProvider {
	if model == "" {
		model = "meta-llama/llama-3.1-70b-instruct"
	}
	return &OpenRouterProvider{
		OpenAIProvider: &OpenAIProvider{
			apiKey:  apiKey,
			model:   model,
			baseURL: "https://openrouter.ai/api/v1",
			httpClient: &http.Client{
				Timeout: 5 * time.Minute,
			},
		},
	}
}

func (o *OpenRouterProvider) Name() string {
	return "openrouter"
}
