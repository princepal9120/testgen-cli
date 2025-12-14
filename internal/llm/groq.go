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

// GroqProvider implements the Provider interface for Groq Cloud
type GroqProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	usage      UsageMetrics
	mu         sync.Mutex
}

// NewGroqProvider creates a new Groq provider
func NewGroqProvider() *GroqProvider {
	return &GroqProvider{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *GroqProvider) Name() string {
	return "groq"
}

// Configure sets up the Groq provider
func (p *GroqProvider) Configure(config ProviderConfig) error {
	if config.APIKey == "" {
		// Try environment variable
		config.APIKey = os.Getenv("GROQ_API_KEY")
	}
	if config.APIKey == "" {
		return ErrNoAPIKey
	}

	if config.Model == "" {
		config.Model = GroqDefaultModel
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 8192
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.groq.com/openai/v1"
	}

	p.config = config
	return nil
}

// groqRequest represents the Groq API request (OpenAI-compatible)
type groqRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
	TopP        float32   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream"`
}

// groqResponse represents the Groq API response (OpenAI-compatible)
type groqResponse struct {
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
		PromptTokens     int     `json:"prompt_tokens"`
		CompletionTokens int     `json:"completion_tokens"`
		TotalTokens      int     `json:"total_tokens"`
		PromptTime       float64 `json:"prompt_time"`
		CompletionTime   float64 `json:"completion_time"`
		TotalTime        float64 `json:"total_time"`
	} `json:"usage"`
	XGroq *struct {
		ID string `json:"id"`
	} `json:"x_groq,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Complete sends a completion request to Groq
func (p *GroqProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
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

	apiReq := groqRequest{
		Model:       p.config.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		TopP:        1.0,
		Stream:      false,
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

	var apiResp groqResponse
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
	// Groq pricing (very low cost due to LPU inference)
	// Llama 3.1 70B: Input: $0.59 / 1M, Output: $0.79 / 1M
	// Llama 3.1 8B: Input: $0.05 / 1M, Output: $0.08 / 1M
	// Mixtral 8x7B: Input: $0.24 / 1M, Output: $0.24 / 1M
	switch p.config.Model {
	case "llama-3.1-70b-versatile", "llama-3.3-70b-versatile":
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.PromptTokens) * 0.59 / 1_000_000
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.CompletionTokens) * 0.79 / 1_000_000
	case "llama-3.1-8b-instant":
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.PromptTokens) * 0.05 / 1_000_000
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.CompletionTokens) * 0.08 / 1_000_000
	case "mixtral-8x7b-32768":
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.PromptTokens) * 0.24 / 1_000_000
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.CompletionTokens) * 0.24 / 1_000_000
	default:
		// Default to Llama 3.1 70B pricing
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.PromptTokens) * 0.59 / 1_000_000
		p.usage.EstimatedCostUSD += float64(apiResp.Usage.CompletionTokens) * 0.79 / 1_000_000
	}
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
func (p *GroqProvider) BatchComplete(ctx context.Context, reqs []CompletionRequest) ([]*CompletionResponse, error) {
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

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return responses, fmt.Errorf("batch had %d errors: %v", len(errs), errs[0])
	}

	return responses, nil
}

// CountTokens estimates token count (rough approximation)
func (p *GroqProvider) CountTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}

// GetUsage returns usage metrics
func (p *GroqProvider) GetUsage() *UsageMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	usage := p.usage
	return &usage
}
