package common

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkerPool represents a pool of workers
type WorkerPool struct {
	workers    int
	jobQueue   chan Job
	workerPool chan chan Job
	quit       chan bool
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    *PoolMetrics
}

// Job represents a job to be executed
type Job struct {
	ID       string
	Type     string
	Data     interface{}
	Func     func(context.Context, interface{}) error
	Timeout  time.Duration
	Retries  int
	callback func(error)
}

// PoolMetrics tracks pool performance
type PoolMetrics struct {
	Processed   int64
	Failed      int64
	Queued      int64
	Active      int64
	MaxQueue    int64
	TotalTime   time.Duration
	LastUpdated time.Time
	mu          sync.RWMutex
}

// Worker represents a pool worker
type Worker struct {
	id         int
	jobChannel chan Job
	workerPool chan chan Job
	quit       chan bool
	ctx        context.Context
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workers:    workers,
		jobQueue:   make(chan Job, queueSize),
		workerPool: make(chan chan Job, workers),
		quit:       make(chan bool),
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &PoolMetrics{},
	}

	// Create workers
	for i := 0; i < workers; i++ {
		worker := NewWorker(i+1, pool.workerPool, pool.quit, ctx)
		worker.Start()
	}

	// Start dispatcher
	go pool.dispatch()

	return pool
}

// NewWorker creates a new worker
func NewWorker(id int, workerPool chan chan Job, quit chan bool, ctx context.Context) *Worker {
	return &Worker{
		id:         id,
		jobChannel: make(chan Job),
		workerPool: workerPool,
		quit:       quit,
		ctx:        ctx,
	}
}

// Start starts the worker
func (w *Worker) Start() {
	go func() {
		for {
			// Register the current worker to the worker pool
			w.workerPool <- w.jobChannel

			select {
			case job := <-w.jobChannel:
				w.processJob(job)
			case <-w.quit:
				return
			case <-w.ctx.Done():
				return
			}
		}
	}()
}

// processJob processes a single job
func (w *Worker) processJob(job Job) {
	start := time.Now()
	var err error

	// Handle job timeout
	ctx := w.ctx
	if job.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, job.Timeout)
		defer cancel()
	}

	// Execute job with retries
	for attempt := 0; attempt <= job.Retries; attempt++ {
		err = job.Func(ctx, job.Data)
		if err == nil || attempt == job.Retries {
			break
		}

		// Wait before retry (exponential backoff)
		if attempt < job.Retries {
			waitTime := time.Duration(attempt+1) * time.Second
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return
			}
		}
	}

	// Update metrics
	_ = time.Since(start) // Track duration for future metrics
	if job.callback != nil {
		job.callback(err)
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.jobChannel)
}

// dispatch dispatches jobs to workers
func (p *WorkerPool) dispatch() {
	for {
		select {
		case job := <-p.jobQueue:
			go func() {
				jobChannel := <-p.workerPool
				jobChannel <- job
			}()

			p.updateMetrics(func(m *PoolMetrics) {
				m.Queued++
				if int64(len(p.jobQueue)) > m.MaxQueue {
					m.MaxQueue = int64(len(p.jobQueue))
				}
			})

		case <-p.quit:
			return
		case <-p.ctx.Done():
			return
		}
	}
}

// Submit submits a job to the pool
func (p *WorkerPool) Submit(job Job) error {
	select {
	case p.jobQueue <- job:
		p.updateMetrics(func(m *PoolMetrics) {
			m.Processed++
			m.LastUpdated = time.Now()
		})
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("worker pool queue is full")
	}
}

// SubmitWithTimeout submits a job with timeout
func (p *WorkerPool) SubmitWithTimeout(job Job, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case p.jobQueue <- job:
		p.updateMetrics(func(m *PoolMetrics) {
			m.Processed++
			m.LastUpdated = time.Now()
		})
		return nil
	case <-ctx.Done():
		return fmt.Errorf("submit job timeout")
	}
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	p.cancel()
	close(p.quit)
	p.wg.Wait()
}

// GetMetrics returns pool metrics
func (p *WorkerPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return *p.metrics
}

// updateMetrics updates pool metrics safely
func (p *WorkerPool) updateMetrics(updateFunc func(*PoolMetrics)) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	updateFunc(p.metrics)
}

// ConnectionPool represents a generic connection pool
type ConnectionPool struct {
	factory     func() (interface{}, error)
	connections chan interface{}
	maxSize     int
	currentSize int
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(factory func() (interface{}, error), maxSize int) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
		factory:     factory,
		connections: make(chan interface{}, maxSize),
		maxSize:     maxSize,
		ctx:         ctx,
		cancel:      cancel,
	}

	return pool
}

// Get gets a connection from the pool
func (p *ConnectionPool) Get() (interface{}, error) {
	select {
	case conn := <-p.connections:
		return conn, nil
	default:
		// Create new connection if pool is not full
		p.mu.Lock()
		if p.currentSize < p.maxSize {
			p.currentSize++
			p.mu.Unlock()

			conn, err := p.factory()
			if err != nil {
				p.mu.Lock()
				p.currentSize--
				p.mu.Unlock()
				return nil, err
			}
			return conn, nil
		}
		p.mu.Unlock()

		// Wait for available connection or timeout
		select {
		case conn := <-p.connections:
			return conn, nil
		case <-time.After(30 * time.Second):
			return nil, fmt.Errorf("connection pool timeout")
		case <-p.ctx.Done():
			return nil, fmt.Errorf("connection pool is shutting down")
		}
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn interface{}) error {
	select {
	case p.connections <- conn:
		return nil
	default:
		// Pool is full, discard connection
		return fmt.Errorf("connection pool is full")
	}
}

// Close closes the connection pool
func (p *ConnectionPool) Close() {
	p.cancel()
	close(p.connections)
}

// Size returns current pool size
func (p *ConnectionPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentSize
}

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	rate       float64
	capacity   int
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate float64, capacity int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     float64(capacity),
		lastUpdate: time.Now(),
	}
}

// Allow checks if the request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	rl.lastUpdate = now

	// Add tokens based on elapsed time
	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.capacity) {
		rl.tokens = float64(rl.capacity)
	}

	// Check if we have enough tokens
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		time.Sleep(time.Millisecond * 10)
	}
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailTime time.Time
	state        CircuitState
	mu           sync.Mutex
}

// CircuitState represents circuit breaker state
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// Call executes the function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check circuit state
	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	case StateHalfOpen:
		// Allow one request through
	}

	// Execute function
	err := fn()

	// Update circuit state based on result
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}
	} else {
		cb.failures = 0
		cb.state = StateClosed
	}

	return err
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// HealthChecker represents a health checker
type HealthChecker struct {
	checks map[string]func() error
	mu     sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]func() error),
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(name string, check func() error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// CheckHealth runs all health checks
func (hc *HealthChecker) CheckHealth() map[string]error {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]error)
	for name, check := range hc.checks {
		results[name] = check()
	}

	return results
}
