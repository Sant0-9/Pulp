package llm

import (
	"net/http"
	"time"
)

type CustomProvider struct {
	*OpenAIProvider
}

func NewCustomProvider(baseURL, apiKey, model string) *CustomProvider {
	return &CustomProvider{
		OpenAIProvider: &OpenAIProvider{
			apiKey:  apiKey,
			model:   model,
			baseURL: baseURL,
			httpClient: &http.Client{
				Timeout: 5 * time.Minute,
			},
		},
	}
}

func (c *CustomProvider) Name() string {
	return "custom"
}
