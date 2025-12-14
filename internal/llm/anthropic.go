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

// AnthropicProvider implements the Provider interface for Anthropic Claude
type AnthropicProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	usage      UsageMetrics
	mu         sync.Mutex
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Configure sets up the Anthropic provider
func (p *AnthropicProvider) Configure(config ProviderConfig) error {
	if config.APIKey == "" {
		// Try environment variable
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if config.APIKey == "" {
		return ErrNoAPIKey
	}

	if config.Model == "" {
		config.Model = AnthropicDefaultModel
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}

	p.config = config
	return nil
}

// anthropicRequest represents the Anthropic API request
type anthropicRequest struct {
	Model       string            `json:"model"`
	MaxTokens   int               `json:"max_tokens"`
	Messages    []Message         `json:"messages"`
	System      string            `json:"system,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
}

// anthropicResponse represents the Anthropic API response
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete sends a completion request to Anthropic
func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if p.config.APIKey == "" {
		return nil, ErrNoAPIKey
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.config.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = p.config.Temperature
	}

	apiReq := anthropicRequest{
		Model:       p.config.Model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Messages: []Message{
			{Role: "user", Content: req.Prompt},
		},
	}

	if req.SystemRole != "" {
		apiReq.System = req.SystemRole
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content
	content := ""
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	// Update usage metrics
	p.mu.Lock()
	p.usage.TotalRequests++
	p.usage.TotalTokensIn += apiResp.Usage.InputTokens
	p.usage.TotalTokensOut += apiResp.Usage.OutputTokens
	// Claude 3.5 Sonnet pricing
	p.usage.EstimatedCostUSD += float64(apiResp.Usage.InputTokens) * 3.00 / 1_000_000
	p.usage.EstimatedCostUSD += float64(apiResp.Usage.OutputTokens) * 15.00 / 1_000_000
	p.mu.Unlock()

	return &CompletionResponse{
		Content:      content,
		TokensInput:  apiResp.Usage.InputTokens,
		TokensOutput: apiResp.Usage.OutputTokens,
		Model:        apiResp.Model,
		FinishReason: apiResp.StopReason,
	}, nil
}

// BatchComplete processes multiple requests
func (p *AnthropicProvider) BatchComplete(ctx context.Context, reqs []CompletionRequest) ([]*CompletionResponse, error) {
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
func (p *AnthropicProvider) CountTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}

// GetUsage returns usage metrics
func (p *AnthropicProvider) GetUsage() *UsageMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	usage := p.usage
	return &usage
}
