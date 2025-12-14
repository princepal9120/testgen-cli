package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	usage      UsageMetrics
	mu         sync.Mutex
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Configure sets up the OpenAI provider
func (p *OpenAIProvider) Configure(config ProviderConfig) error {
	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if config.APIKey == "" {
		return ErrNoAPIKey
	}

	if config.Model == "" {
		config.Model = OpenAIDefaultModel
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	p.config = config
	return nil
}

// openAIRequest represents the OpenAI API request
type openAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
	Seed        *int      `json:"seed,omitempty"`
}

// openAIResponse represents the OpenAI API response
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Complete sends a completion request to OpenAI
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if p.config.APIKey == "" {
		return nil, ErrNoAPIKey
	}

	messages := make([]Message, 0, 2)

	if req.SystemRole != "" {
		messages = append(messages, Message{Role: "system", Content: req.SystemRole})
	}
	messages = append(messages, Message{Role: "user", Content: req.Prompt})

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.config.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = p.config.Temperature
	}

	apiReq := openAIRequest{
		Model:       p.config.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Seed:        req.Seed,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 429 {
		return nil, ErrRateLimited
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	content := ""
	finishReason := ""
	if len(apiResp.Choices) > 0 {
		content = apiResp.Choices[0].Message.Content
		finishReason = apiResp.Choices[0].FinishReason
	}

	// Update usage metrics
	p.mu.Lock()
	p.usage.TotalRequests++
	p.usage.TotalTokensIn += apiResp.Usage.PromptTokens
	p.usage.TotalTokensOut += apiResp.Usage.CompletionTokens
	// GPT-4 Turbo pricing (approximate)
	p.usage.EstimatedCostUSD += float64(apiResp.Usage.PromptTokens) * 10.00 / 1_000_000
	p.usage.EstimatedCostUSD += float64(apiResp.Usage.CompletionTokens) * 30.00 / 1_000_000
	p.mu.Unlock()

	return &CompletionResponse{
		Content:      content,
		TokensInput:  apiResp.Usage.PromptTokens,
		TokensOutput: apiResp.Usage.CompletionTokens,
		Model:        apiResp.Model,
		FinishReason: finishReason,
	}, nil
}

// BatchComplete processes multiple requests
func (p *OpenAIProvider) BatchComplete(ctx context.Context, reqs []CompletionRequest) ([]*CompletionResponse, error) {
	responses := make([]*CompletionResponse, len(reqs))
	var wg sync.WaitGroup
	errChan := make(chan error, len(reqs))

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r CompletionRequest) {
			defer wg.Done()

			resp, err := p.Complete(ctx, r)
			if err != nil {
				errChan <- fmt.Errorf("request %d failed: %w", idx, err)
				return
			}
			responses[idx] = resp
		}(i, req)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return responses, fmt.Errorf("batch had %d errors: %v", len(errs), errs[0])
	}

	return responses, nil
}

// CountTokens estimates token count
func (p *OpenAIProvider) CountTokens(text string) int {
	// Rough estimate: ~4 characters per token for English
	return len(text) / 4
}

// GetUsage returns usage metrics
func (p *OpenAIProvider) GetUsage() *UsageMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	usage := p.usage
	return &usage
}
