package llm

import (
	"context"
	"sync"
	"time"
)

// RateLimiter controls request rate to LLM providers
type RateLimiter struct {
	requestsPerMinute int
	tokens            chan struct{}
	mu                sync.Mutex
	lastRefill        time.Time
}

// NewRateLimiter creates a rate limiter with the given requests per minute
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}

	rl := &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		tokens:            make(chan struct{}, requestsPerMinute),
		lastRefill:        time.Now(),
	}

	// Fill initial tokens
	for i := 0; i < requestsPerMinute; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refill goroutine
	go rl.refillLoop()

	return rl
}

func (rl *RateLimiter) refillLoop() {
	ticker := time.NewTicker(time.Minute / time.Duration(rl.requestsPerMinute))
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
			// Channel full, skip
		}
	}
}

// Wait blocks until a request can proceed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Batcher batches multiple requests for efficiency
type Batcher struct {
	batchSize    int
	flushTimeout time.Duration
	pending      []CompletionRequest
	results      chan batchResult
	mu           sync.Mutex
	provider     Provider
}

type batchResult struct {
	index    int
	response *CompletionResponse
	err      error
}

// NewBatcher creates a request batcher
func NewBatcher(provider Provider, batchSize int, flushTimeout time.Duration) *Batcher {
	if batchSize <= 0 {
		batchSize = 5
	}
	if flushTimeout <= 0 {
		flushTimeout = 2 * time.Second
	}

	return &Batcher{
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		pending:      make([]CompletionRequest, 0, batchSize),
		results:      make(chan batchResult, batchSize*2),
		provider:     provider,
	}
}

// Add adds a request to the batch
func (b *Batcher) Add(req CompletionRequest) {
	b.mu.Lock()
	b.pending = append(b.pending, req)
	shouldFlush := len(b.pending) >= b.batchSize
	b.mu.Unlock()

	if shouldFlush {
		b.Flush(context.Background())
	}
}

// Flush processes all pending requests
func (b *Batcher) Flush(ctx context.Context) ([]*CompletionResponse, error) {
	b.mu.Lock()
	reqs := b.pending
	b.pending = make([]CompletionRequest, 0, b.batchSize)
	b.mu.Unlock()

	if len(reqs) == 0 {
		return nil, nil
	}

	return b.provider.BatchComplete(ctx, reqs)
}

// GetBatchSize returns the configured batch size
func (b *Batcher) GetBatchSize() int {
	return b.batchSize
}

// PendingCount returns the number of pending requests
func (b *Batcher) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}
