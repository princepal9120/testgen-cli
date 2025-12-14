package generator

import (
	"context"
	"sync"

	"github.com/princepal9120/testgen-cli/internal/adapters"
	"github.com/princepal9120/testgen-cli/pkg/models"
)

// WorkerPool manages parallel test generation
type WorkerPool struct {
	workers    int
	jobs       chan job
	results    chan *models.GenerationResult
	wg         sync.WaitGroup
	engine     *Engine
	registry   *adapters.Registry
}

type job struct {
	file    *models.SourceFile
	adapter adapters.LanguageAdapter
}

// NewWorkerPool creates a worker pool with the specified number of workers
func NewWorkerPool(engine *Engine, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 2
	}

	return &WorkerPool{
		workers:  workers,
		jobs:     make(chan job, workers*2),
		results:  make(chan *models.GenerationResult, workers*2),
		engine:   engine,
		registry: adapters.DefaultRegistry(),
	}
}

// Start launches the worker goroutines
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx)
	}
}

func (wp *WorkerPool) worker(ctx context.Context) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case j, ok := <-wp.jobs:
			if !ok {
				return
			}
			result, err := wp.engine.Generate(j.file, j.adapter)
			if err != nil {
				result = &models.GenerationResult{
					SourceFile:   j.file,
					Error:        err,
					ErrorMessage: err.Error(),
				}
			}
			wp.results <- result
		}
	}
}

// Submit adds a file to the processing queue
func (wp *WorkerPool) Submit(file *models.SourceFile) {
	adapter := wp.registry.GetAdapter(file.Language)
	if adapter == nil {
		wp.results <- &models.GenerationResult{
			SourceFile:   file,
			ErrorMessage: "no adapter for language: " + file.Language,
		}
		return
	}
	wp.jobs <- job{file: file, adapter: adapter}
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan *models.GenerationResult {
	return wp.results
}

// Close shuts down the worker pool
func (wp *WorkerPool) Close() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}

// ProcessFiles processes multiple files in parallel
func (wp *WorkerPool) ProcessFiles(ctx context.Context, files []*models.SourceFile) []*models.GenerationResult {
	wp.Start(ctx)

	go func() {
		for _, f := range files {
			wp.Submit(f)
		}
		close(wp.jobs)
	}()

	results := make([]*models.GenerationResult, 0, len(files))
	for i := 0; i < len(files); i++ {
		select {
		case r := <-wp.results:
			results = append(results, r)
		case <-ctx.Done():
			break
		}
	}

	wp.wg.Wait()
	return results
}
