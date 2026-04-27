package server

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"
)

const (
	statusOK     = "ok"
	statusFailed = "failed"
)

// Check is one probe's result. Status is statusOK or statusFailed.
type Check struct {
	Status     string
	DurationMs int64
	Message    string
}

// CheckFn must honour ctx deadlines so probe latency stays bounded.
type CheckFn func(ctx context.Context) error

// HealthChecker serves /healthz (always 200) and /readyz (503 if any probe
// fails). Probes run concurrently with a shared deadline.
type HealthChecker struct {
	startTime time.Time
	version   string
	timeout   time.Duration

	mu     sync.RWMutex
	checks map[string]CheckFn
}

// NewHealthChecker builds a checker with no probes wired yet. timeout is the
// per-check deadline.
func NewHealthChecker(version string, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		startTime: time.Now(),
		version:   version,
		timeout:   timeout,
		checks:    map[string]CheckFn{},
	}
}

// Register adds a named probe; overwrites any existing probe with the same
// name.
func (h *HealthChecker) Register(name string, fn CheckFn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = fn
}

// RegisterHandlers mounts /healthz and /readyz on mux.
func (h *HealthChecker) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.Liveness)
	mux.HandleFunc("/readyz", h.Readiness)
}

// Liveness does NOT run readiness probes — a flaky downstream should not
// restart the pod.
func (h *HealthChecker) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Readiness returns 503 if any probe fails, 200 otherwise.
func (h *HealthChecker) Readiness(w http.ResponseWriter, r *http.Request) {
	for _, c := range h.Snapshot(r.Context()) {
		if c.Status != statusOK {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// Snapshot runs every registered probe under the per-check timeout and
// returns the per-probe results.
func (h *HealthChecker) Snapshot(parent context.Context) map[string]Check {
	h.mu.RLock()
	checks := make(map[string]CheckFn, len(h.checks))
	maps.Copy(checks, h.checks)
	h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(parent, h.timeout)
	defer cancel()

	results := make(map[string]Check, len(checks))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for name, fn := range checks {
		wg.Add(1)
		go func(name string, fn CheckFn) {
			defer wg.Done()
			start := time.Now()
			err := fn(ctx)
			c := Check{DurationMs: time.Since(start).Milliseconds()}
			if err == nil {
				c.Status = statusOK
			} else {
				c.Status = statusFailed
				c.Message = err.Error()
			}
			mu.Lock()
			results[name] = c
			mu.Unlock()
		}(name, fn)
	}
	wg.Wait()

	for name := range checks {
		if _, ok := results[name]; !ok {
			results[name] = Check{Status: statusFailed, Message: "deadline exceeded", DurationMs: h.timeout.Milliseconds()}
		}
	}
	return results
}

// HTTPProbe returns a CheckFn that GETs url and reports ok on any 2xx.
func HTTPProbe(client *http.Client, url string) CheckFn {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}
}
