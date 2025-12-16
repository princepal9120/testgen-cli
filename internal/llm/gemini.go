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

// GeminiProvider implements the Provider interface for Google Gemini
type GeminiProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	usage      UsageMetrics
	mu         sync.Mutex
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Configure sets up the Gemini provider
func (p *GeminiProvider) Configure(config ProviderConfig) error {
	if config.APIKey == "" {
		// Try environment variable
		config.APIKey = os.Getenv("GEMINI_API_KEY")
		if config.APIKey == "" {
			// Also try GOOGLE_API_KEY as fallback
			config.APIKey = os.Getenv("GOOGLE_API_KEY")
		}
	}
	if config.APIKey == "" {
		return ErrNoAPIKey
	}

	if config.Model == "" {
		config.Model = GeminiDefaultModel
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 8192
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	p.config = config
	return nil
}

// geminiRequest represents the Gemini API request
type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float32 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopP            float32 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

// geminiResponse represents the Gemini API response
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason  string `json:"finishReason"`
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// Complete sends a completion request to Gemini
func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
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

	apiReq := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: req.Prompt}},
				Role:  "user",
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     temperature,
			MaxOutputTokens: maxTokens,
			TopP:            0.95,
			TopK:            40,
		},
	}

	if req.SystemRole != "" {
		apiReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemRole}},
		}
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Gemini uses query parameter for API key
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.config.BaseURL, p.config.Model, p.config.APIKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

	var apiResp geminiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error (%s): %s", apiResp.Error.Status, apiResp.Error.Message)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Extract content
	content := ""
	finishReason := ""
	if len(apiResp.Candidates) > 0 {
		for _, part := range apiResp.Candidates[0].Content.Parts {
			content += part.Text
		}
		finishReason = apiResp.Candidates[0].FinishReason
	}

	// Update usage metrics
	p.mu.Lock()
	p.usage.TotalRequests++
	p.usage.TotalTokensIn += apiResp.UsageMetadata.PromptTokenCount
	p.usage.TotalTokensOut += apiResp.UsageMetadata.CandidatesTokenCount
	// Gemini 1.5 Flash pricing (per million tokens)
	// Input: $0.075 / 1M, Output: $0.30 / 1M (flash model)
	// Gemini 1.5 Pro: Input: $1.25 / 1M, Output: $5.00 / 1M
	if p.config.Model == "gemini-1.5-flash" || p.config.Model == "gemini-1.5-flash-latest" {
		p.usage.EstimatedCostUSD += float64(apiResp.UsageMetadata.PromptTokenCount) * 0.075 / 1_000_000
		p.usage.EstimatedCostUSD += float64(apiResp.UsageMetadata.CandidatesTokenCount) * 0.30 / 1_000_000
	} else {
		// Default to Pro pricing
		p.usage.EstimatedCostUSD += float64(apiResp.UsageMetadata.PromptTokenCount) * 1.25 / 1_000_000
		p.usage.EstimatedCostUSD += float64(apiResp.UsageMetadata.CandidatesTokenCount) * 5.00 / 1_000_000
	}
	p.mu.Unlock()

	return &CompletionResponse{
		Content:      content,
		TokensInput:  apiResp.UsageMetadata.PromptTokenCount,
		TokensOutput: apiResp.UsageMetadata.CandidatesTokenCount,
		Model:        p.config.Model,
		FinishReason: finishReason,
	}, nil
}

// BatchComplete processes multiple requests
func (p *GeminiProvider) BatchComplete(ctx context.Context, reqs []CompletionRequest) ([]*CompletionResponse, error) {
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
func (p *GeminiProvider) CountTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}

// GetUsage returns usage metrics
func (p *GeminiProvider) GetUsage() *UsageMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	usage := p.usage
	return &usage
}
