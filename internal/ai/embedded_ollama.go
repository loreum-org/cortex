package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// EmbeddedOllama manages an Ollama server process within the Cortex node lifecycle
type EmbeddedOllama struct {
	// Configuration
	Host        string
	Port        int
	ModelsDir   string
	LogLevel    string
	GpuLayers   int
	NumParallel int

	// Process management
	cmd          *exec.Cmd
	running      bool
	mu           sync.RWMutex
	startedAt    time.Time
	healthCheck  *time.Ticker
	shutdownChan chan struct{}

	// Callbacks
	OnStarted   func()
	OnStopped   func()
	OnHealthy   func()
	OnUnhealthy func(error)
}

// EmbeddedOllamaConfig configures the embedded Ollama instance
type EmbeddedOllamaConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	ModelsDir   string `json:"models_dir"`
	LogLevel    string `json:"log_level"`
	GpuLayers   int    `json:"gpu_layers"`
	NumParallel int    `json:"num_parallel"`
	AutoStart   bool   `json:"auto_start"`
	HealthCheck bool   `json:"health_check"`
}

// DefaultEmbeddedOllamaConfig returns default configuration
func DefaultEmbeddedOllamaConfig() EmbeddedOllamaConfig {
	return EmbeddedOllamaConfig{
		Host:        "127.0.0.1",
		Port:        11434,
		ModelsDir:   filepath.Join(os.Getenv("HOME"), ".ollama", "models"),
		LogLevel:    "info",
		GpuLayers:   -1, // Auto-detect
		NumParallel: 1,
		AutoStart:   true,
		HealthCheck: true,
	}
}

// NewEmbeddedOllama creates a new embedded Ollama instance
func NewEmbeddedOllama(config EmbeddedOllamaConfig) *EmbeddedOllama {
	return &EmbeddedOllama{
		Host:         config.Host,
		Port:         config.Port,
		ModelsDir:    config.ModelsDir,
		LogLevel:     config.LogLevel,
		GpuLayers:    config.GpuLayers,
		NumParallel:  config.NumParallel,
		shutdownChan: make(chan struct{}),
	}
}

// Start starts the embedded Ollama server
func (e *EmbeddedOllama) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("ollama server is already running")
	}

	// Check if Ollama is already running externally
	if e.isHealthy() {
		log.Printf("Ollama server already running on %s:%d", e.Host, e.Port)
		e.running = true
		e.startedAt = time.Now()
		if e.OnStarted != nil {
			e.OnStarted()
		}
		return nil
	}

	// Check if ollama binary exists
	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		return fmt.Errorf("ollama binary not found in PATH: %w", err)
	}

	// Prepare environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("OLLAMA_HOST=%s:%d", e.Host, e.Port))
	env = append(env, fmt.Sprintf("OLLAMA_MODELS=%s", e.ModelsDir))
	env = append(env, fmt.Sprintf("OLLAMA_LOGS=%s", e.LogLevel))
	env = append(env, fmt.Sprintf("OLLAMA_NUM_PARALLEL=%d", e.NumParallel))

	if e.GpuLayers >= 0 {
		env = append(env, fmt.Sprintf("OLLAMA_GPU_LAYERS=%d", e.GpuLayers))
	}

	// Create command
	e.cmd = exec.CommandContext(ctx, ollamaPath, "serve")
	e.cmd.Env = env
	e.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group for clean shutdown
	}

	// Redirect logs if needed
	if e.LogLevel == "debug" {
		e.cmd.Stdout = os.Stdout
		e.cmd.Stderr = os.Stderr
	}

	// Start the process
	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama server: %w", err)
	}

	e.running = true
	e.startedAt = time.Now()

	log.Printf("Started embedded Ollama server (PID: %d) on %s:%d", e.cmd.Process.Pid, e.Host, e.Port)

	// Wait for server to be ready
	if err := e.waitForReady(ctx, 30*time.Second); err != nil {
		e.Stop()
		return fmt.Errorf("ollama server failed to become ready: %w", err)
	}

	// Start health monitoring
	go e.startHealthMonitoring()

	// Monitor process
	go e.monitorProcess()

	if e.OnStarted != nil {
		e.OnStarted()
	}

	return nil
}

// Stop stops the embedded Ollama server
func (e *EmbeddedOllama) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	log.Printf("Stopping embedded Ollama server...")

	// Stop health monitoring
	close(e.shutdownChan)
	if e.healthCheck != nil {
		e.healthCheck.Stop()
	}

	// Gracefully terminate the process
	if e.cmd != nil && e.cmd.Process != nil {
		// First try SIGTERM
		if err := e.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("Failed to send SIGTERM to ollama: %v", err)
		}

		// Wait up to 10 seconds for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- e.cmd.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				log.Printf("Ollama process exited with error: %v", err)
			}
		case <-time.After(10 * time.Second):
			// Force kill if graceful shutdown takes too long
			log.Printf("Force killing ollama process...")
			if err := e.cmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill ollama process: %v", err)
			}
			<-done // Wait for process to actually exit
		}
	}

	e.running = false
	e.cmd = nil

	log.Printf("Embedded Ollama server stopped")

	if e.OnStopped != nil {
		e.OnStopped()
	}

	return nil
}

// IsRunning returns whether the Ollama server is running
func (e *EmbeddedOllama) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// GetAddress returns the full address of the Ollama server
func (e *EmbeddedOllama) GetAddress() string {
	return fmt.Sprintf("http://%s:%d", e.Host, e.Port)
}

// GetUptime returns how long the server has been running
func (e *EmbeddedOllama) GetUptime() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.running {
		return 0
	}
	return time.Since(e.startedAt)
}

// waitForReady waits for the Ollama server to become ready
func (e *EmbeddedOllama) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if e.isHealthy() {
				log.Printf("Embedded Ollama server is ready")
				return nil
			}
		}
	}
}

// isHealthy checks if the Ollama server is healthy
func (e *EmbeddedOllama) isHealthy() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/api/tags", e.GetAddress()))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// startHealthMonitoring starts periodic health checks
func (e *EmbeddedOllama) startHealthMonitoring() {
	e.healthCheck = time.NewTicker(30 * time.Second)

	for {
		select {
		case <-e.healthCheck.C:
			if e.isHealthy() {
				if e.OnHealthy != nil {
					e.OnHealthy()
				}
			} else {
				err := fmt.Errorf("ollama health check failed")
				log.Printf("Embedded Ollama health check failed")
				if e.OnUnhealthy != nil {
					e.OnUnhealthy(err)
				}
			}
		case <-e.shutdownChan:
			return
		}
	}
}

// monitorProcess monitors the Ollama process and handles unexpected exits
func (e *EmbeddedOllama) monitorProcess() {
	if e.cmd == nil {
		return
	}

	err := e.cmd.Wait()

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		log.Printf("Embedded Ollama process exited unexpectedly: %v", err)
		e.running = false
		e.cmd = nil

		if e.OnStopped != nil {
			e.OnStopped()
		}
	}
}

// Restart stops and starts the Ollama server
func (e *EmbeddedOllama) Restart(ctx context.Context) error {
	if err := e.Stop(); err != nil {
		return fmt.Errorf("failed to stop ollama: %w", err)
	}

	// Wait a moment before restarting
	time.Sleep(2 * time.Second)

	return e.Start(ctx)
}

// PullModel pulls a model using the embedded Ollama instance
func (e *EmbeddedOllama) PullModel(ctx context.Context, model string) error {
	if !e.IsRunning() {
		return fmt.Errorf("ollama server is not running")
	}

	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		return fmt.Errorf("ollama binary not found: %w", err)
	}

	env := []string{fmt.Sprintf("OLLAMA_HOST=%s", e.GetAddress())}
	cmd := exec.CommandContext(ctx, ollamaPath, "pull", model)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Pulling model %s...", model)
	return cmd.Run()
}

// ListModels returns available models
func (e *EmbeddedOllama) ListModels(ctx context.Context) ([]string, error) {
	if !e.IsRunning() {
		return nil, fmt.Errorf("ollama server is not running")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/api/tags", e.GetAddress()))
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, model := range result.Models {
		models[i] = model.Name
	}

	return models, nil
}
