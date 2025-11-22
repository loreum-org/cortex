package desktop

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ProcessManager handles the cortexd server process lifecycle
type ProcessManager struct {
	cmd           *exec.Cmd
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	port          int
	p2pPort       int
	binaryPath    string
	autoRestart   bool
	restartCount  int
	maxRestarts   int
	healthClient  *http.Client
}

// NewProcessManager creates a new process manager
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		port:        4891,
		p2pPort:     4001,
		autoRestart: true,
		maxRestarts: 3,
		healthClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 1 * time.Second,
				}).DialContext,
			},
		},
	}
}

// SetPorts configures the server and P2P ports
func (pm *ProcessManager) SetPorts(serverPort, p2pPort int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.port = serverPort
	pm.p2pPort = p2pPort
}

// Start launches the cortexd server process
func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isRunning {
		return fmt.Errorf("cortexd process is already running")
	}

	// Find the cortexd binary
	binaryPath, err := pm.findCortexdBinary()
	if err != nil {
		return fmt.Errorf("failed to find cortexd binary: %w", err)
	}
	pm.binaryPath = binaryPath

	// Create context for the process
	pm.ctx, pm.cancel = context.WithCancel(context.Background())

	// Prepare the command
	args := []string{
		"serve",
		"--port", fmt.Sprintf("%d", pm.port),
		"--p2p-port", fmt.Sprintf("%d", pm.p2pPort),
	}

	pm.cmd = exec.CommandContext(pm.ctx, binaryPath, args...)
	
	// Set process group to allow clean shutdown
	pm.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Redirect stdout and stderr to logs for debugging
	pm.cmd.Stdout = os.Stdout
	pm.cmd.Stderr = os.Stderr

	log.Printf("Starting cortexd server: %s %v", binaryPath, args)

	// Start the process
	if err := pm.cmd.Start(); err != nil {
		pm.cancel()
		return fmt.Errorf("failed to start cortexd process: %w", err)
	}

	pm.isRunning = true
	log.Printf("cortexd server started with PID: %d", pm.cmd.Process.Pid)

	// Monitor the process in a goroutine
	go pm.monitorProcess()

	// Wait for the server to be ready
	if err := pm.waitForServer(); err != nil {
		pm.Stop()
		return fmt.Errorf("cortexd server failed to start properly: %w", err)
	}

	log.Println("cortexd server is ready and accepting connections")
	return nil
}

// Stop terminates the cortexd server process
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isRunning || pm.cmd == nil {
		return nil
	}

	log.Println("Stopping cortexd server...")

	// Cancel the context to signal shutdown
	if pm.cancel != nil {
		pm.cancel()
	}

	// Try graceful shutdown first
	if pm.cmd.Process != nil {
		// Send SIGTERM to the process group
		if err := syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGTERM); err != nil {
			log.Printf("Failed to send SIGTERM to cortexd process group: %v", err)
			// Fallback to killing just the main process
			if err := pm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
				log.Printf("Failed to send SIGTERM to cortexd process: %v", err)
			}
		}

		// Wait for graceful shutdown with timeout
		done := make(chan error, 1)
		go func() {
			done <- pm.cmd.Wait()
		}()

		select {
		case <-time.After(15 * time.Second): // Increased timeout for graceful shutdown
			log.Println("Graceful shutdown timeout, forcing kill...")
			// Force kill if graceful shutdown fails
			if err := syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGKILL); err != nil {
				log.Printf("Failed to send SIGKILL to process group: %v", err)
				// Fallback to killing just the main process
				if killErr := pm.cmd.Process.Kill(); killErr != nil {
					log.Printf("Failed to kill main process: %v", killErr)
				}
			}
			// Wait for the process to actually exit with another timeout
			select {
			case <-done:
				log.Println("Process terminated after force kill")
			case <-time.After(5 * time.Second):
				log.Println("Process failed to exit even after force kill")
			}
		case err := <-done:
			if err != nil {
				log.Printf("cortexd process exited with error: %v", err)
			} else {
				log.Println("cortexd process stopped gracefully")
			}
		}
	}

	pm.isRunning = false
	pm.cmd = nil
	log.Println("cortexd server stopped")
	return nil
}

// IsRunning returns whether the cortexd process is currently running
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isRunning
}

// GetServerURL returns the URL of the cortexd server
func (pm *ProcessManager) GetServerURL() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return fmt.Sprintf("http://localhost:%d", pm.port)
}

// GetPorts returns the configured server and P2P ports
func (pm *ProcessManager) GetPorts() (int, int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.port, pm.p2pPort
}

// findCortexdBinary locates the cortexd executable with proper validation
func (pm *ProcessManager) findCortexdBinary() (string, error) {
	// First try the same directory as the desktop app
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	
	execDir := filepath.Dir(execPath)
	
	// Try different possible locations
	possiblePaths := []string{
		filepath.Join(execDir, "cortexd"),
		filepath.Join(execDir, "..", "..", "cortexd"),
		filepath.Join(execDir, "..", "..", "..", "cortexd"),
		"./cortexd",
		"../cortexd",
		"../../cortexd",
	}

	// Check each possible path
	for _, path := range possiblePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if pm.isExecutable(absPath) {
				log.Printf("Found cortexd binary at: %s", absPath)
				return absPath, nil
			}
		}
	}

	// Try to find in PATH
	if path, err := exec.LookPath("cortexd"); err == nil {
		if pm.isExecutable(path) {
			log.Printf("Found cortexd binary in PATH: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("cortexd binary not found or not executable in any expected location. Please ensure cortexd is built and available")
}

// isExecutable checks if a file exists and is executable
func (pm *ProcessManager) isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	
	// Check if it's a regular file and executable
	if !info.Mode().IsRegular() {
		return false
	}
	
	// Check executable permission (Unix-like systems)
	if info.Mode()&0111 == 0 {
		return false
	}
	
	return true
}

// waitForServer waits for the cortexd server to be ready
func (pm *ProcessManager) waitForServer() error {
	serverURL := pm.GetServerURL()
	log.Printf("Waiting for cortexd server to be ready at %s", serverURL)

	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for cortexd server to be ready")
		case <-ticker.C:
			// Try to connect to the server
			if pm.checkServerHealth(serverURL) {
				log.Printf("Server health check passed")
				return nil
			}
		case <-pm.ctx.Done():
			return fmt.Errorf("process was cancelled while waiting for server")
		}
	}
}

// checkServerHealth performs a native HTTP health check on the server
func (pm *ProcessManager) checkServerHealth(serverURL string) bool {
	// First try a simple TCP connection to see if port is open
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", pm.port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	// Try HTTP health endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := pm.healthClient.Do(req)
	if err != nil {
		// If /health endpoint doesn't exist, try root endpoint
		req, err = http.NewRequestWithContext(ctx, "GET", serverURL+"/", nil)
		if err != nil {
			return false
		}
		resp, err = pm.healthClient.Do(req)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()

	// Accept any 2xx or 3xx status code as healthy
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// monitorProcess monitors the cortexd process and handles unexpected exits
func (pm *ProcessManager) monitorProcess() {
	if pm.cmd == nil {
		return
	}

	// Wait for the process to exit
	err := pm.cmd.Wait()

	pm.mu.Lock()
	wasIntentional := pm.ctx.Err() != nil // Check if shutdown was intentional
	pm.isRunning = false
	pm.mu.Unlock()

	if err != nil && !wasIntentional {
		log.Printf("cortexd process exited unexpectedly: %v", err)
		
		// Attempt auto-restart if enabled and under limit
		if pm.autoRestart && pm.restartCount < pm.maxRestarts {
			pm.restartCount++
			log.Printf("Attempting auto-restart (%d/%d)...", pm.restartCount, pm.maxRestarts)
			
			// Wait a moment before restarting
			time.Sleep(2 * time.Second)
			
			if restartErr := pm.Start(); restartErr != nil {
				log.Printf("Auto-restart failed: %v", restartErr)
			} else {
				log.Printf("Auto-restart successful")
				pm.restartCount = 0 // Reset counter on successful restart
			}
		} else if pm.restartCount >= pm.maxRestarts {
			log.Printf("Maximum restart attempts (%d) exceeded. Process will not be restarted automatically.", pm.maxRestarts)
		}
	}
}

// SetAutoRestart configures automatic restart behavior
func (pm *ProcessManager) SetAutoRestart(enabled bool, maxRestarts int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.autoRestart = enabled
	pm.maxRestarts = maxRestarts
}

// GetRestartCount returns the current restart count
func (pm *ProcessManager) GetRestartCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.restartCount
}

// ResetRestartCount resets the restart counter
func (pm *ProcessManager) ResetRestartCount() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.restartCount = 0
}