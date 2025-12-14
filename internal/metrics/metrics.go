/*
Package metrics provides usage and cost tracking for TestGen.
*/
package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// RunMetrics represents metrics for a single run
type RunMetrics struct {
	RunID                string    `json:"run_id"`
	Timestamp            time.Time `json:"timestamp"`
	TotalFiles           int       `json:"total_files"`
	TokensInput          int       `json:"tokens_input"`
	TokensOutput         int       `json:"tokens_output"`
	TokensCached         int       `json:"tokens_cached"`
	CacheHitRate         float64   `json:"cache_hit_rate"`
	TotalCostUSD         float64   `json:"total_cost_usd"`
	ExecutionTimeSeconds float64   `json:"execution_time_seconds"`
	SuccessCount         int       `json:"success_count"`
	ErrorCount           int       `json:"error_count"`
}

// Collector collects and stores metrics
type Collector struct {
	metricsDir string
	current    *RunMetrics
	startTime  time.Time
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	// Use .testgen/metrics in current directory
	metricsDir := filepath.Join(".testgen", "metrics")
	_ = os.MkdirAll(metricsDir, 0755)

	runID := time.Now().Format("20060102-150405")

	return &Collector{
		metricsDir: metricsDir,
		current: &RunMetrics{
			RunID:     runID,
			Timestamp: time.Now(),
		},
		startTime: time.Now(),
	}
}

// RecordFile records a file being processed
func (c *Collector) RecordFile(success bool) {
	c.current.TotalFiles++
	if success {
		c.current.SuccessCount++
	} else {
		c.current.ErrorCount++
	}
}

// RecordTokens records token usage
func (c *Collector) RecordTokens(input, output int, cached bool) {
	c.current.TokensInput += input
	c.current.TokensOutput += output
	if cached {
		c.current.TokensCached += input
	}
}

// RecordCost records cost
func (c *Collector) RecordCost(costUSD float64) {
	c.current.TotalCostUSD += costUSD
}

// SetCacheHitRate sets the cache hit rate
func (c *Collector) SetCacheHitRate(rate float64) {
	c.current.CacheHitRate = rate
}

// Finalize completes metrics collection
func (c *Collector) Finalize() *RunMetrics {
	c.current.ExecutionTimeSeconds = time.Since(c.startTime).Seconds()
	return c.current
}

// Save saves metrics to disk
func (c *Collector) Save() error {
	c.Finalize()

	filename := filepath.Join(c.metricsDir, c.current.RunID+".json")

	data, err := json.MarshalIndent(c.current, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// GetCurrent returns current metrics
func (c *Collector) GetCurrent() *RunMetrics {
	return c.current
}
