// Package main provides the Wails desktop application for Cortex.
// This app serves as a native wrapper that starts the cortexd server process
// and provides desktop integration features like updates and notifications.
// The React frontend connects directly to the cortexd API on port 4891.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/desktop"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App represents the main desktop application structure.
// It manages the cortexd server process and provides desktop-specific functionality.
type App struct {
	ctx            context.Context
	processManager *desktop.ProcessManager // Manages cortexd server lifecycle
	updater        *desktop.Updater        // Handles application updates
	serverReady    chan error              // Channel to signal server startup completion
	shuttingDown   bool                    // Flag to track shutdown state
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		processManager: desktop.NewProcessManager(),
		serverReady:    make(chan error, 1),
	}
	app.updater = desktop.NewUpdater(app)
	return app
}

// OnStartup is called when the app starts, before the first window is shown
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.updater.SetContext(ctx)

	log.Println("Desktop app starting up...")

	// Start cortexd server with proper synchronization
	go func() {
		log.Println("Starting cortexd server...")
		if err := a.processManager.Start(); err != nil {
			log.Printf("Failed to start cortexd server: %v", err)
			a.serverReady <- err
		} else {
			log.Println("cortexd server started successfully")
			a.serverReady <- nil
		}
	}()

	// Wait for server to be ready or fail with timeout
	go func() {
		select {
		case err := <-a.serverReady:
			if err != nil {
				log.Printf("Server startup failed: %v", err)
				runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
					Type:    runtime.ErrorDialog,
					Title:   "Server Startup Failed",
					Message: fmt.Sprintf("Failed to start cortexd server: %v\n\nThe application may not work properly.", err),
				})
			} else {
				log.Println("Server is ready, desktop app fully initialized")
			}
		case <-time.After(45 * time.Second):
			log.Println("Server startup timeout")
			runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
				Type:    runtime.WarningDialog,
				Title:   "Startup Timeout",
				Message: "Server is taking longer than expected to start. The application may not work properly.",
			})
		}
	}()
}

// OnDomReady is called after the front-end dom has been loaded
func (a *App) OnDomReady(ctx context.Context) {
	log.Println("DOM ready")
	// System tray and other desktop integrations can be added here if needed
}

// OnShutdown is called when the application is terminating
func (a *App) OnShutdown(ctx context.Context) {
	log.Println("Desktop app shutting down...")
	a.shuttingDown = true

	if a.processManager != nil {
		log.Println("Stopping cortexd server...")
		
		// Use a goroutine to avoid blocking Wails shutdown with timeout
		done := make(chan error, 1)
		go func() {
			done <- a.processManager.Stop()
		}()
		
		// Wait for server shutdown with timeout
		select {
		case err := <-done:
			if err != nil {
				log.Printf("Error stopping cortexd server: %v", err)
			} else {
				log.Println("Server shutdown complete")
			}
		case <-time.After(8 * time.Second):
			log.Println("Server shutdown timeout - forcing app exit")
		}
	}
}

// Wails bindings - minimal desktop app methods

// IsServerRunning returns whether the cortexd server is running
func (a *App) IsServerRunning() bool {
	return a.processManager.IsRunning()
}

// GetServerURL returns the server URL
func (a *App) GetServerURL() string {
	return a.processManager.GetServerURL()
}

// RestartServer restarts the cortexd server
func (a *App) RestartServer() error {
	log.Println("Restarting cortexd server...")

	// Stop the current server
	if err := a.processManager.Stop(); err != nil {
		log.Printf("Error stopping cortexd server during restart: %v", err)
	}

	// Wait a moment for clean shutdown
	time.Sleep(2 * time.Second)

	// Start the server again
	go func() {
		if err := a.processManager.Start(); err != nil {
			log.Printf("Failed to restart cortexd server: %v", err)
			if a.ctx != nil {
				runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
					Type:    runtime.ErrorDialog,
					Title:   "Restart Error",
					Message: fmt.Sprintf("Failed to restart cortexd server: %v", err),
				})
			}
		} else {
			log.Println("cortexd server restarted successfully")
		}
	}()

	return nil
}

// CheckForUpdates checks for app updates
func (a *App) CheckForUpdates() (map[string]interface{}, error) {
	updateInfo, err := a.updater.CheckForUpdates()
	if err != nil {
		return nil, err
	}

	if updateInfo == nil {
		return map[string]interface{}{
			"available": false,
			"message":   "No updates available",
		}, nil
	}

	return map[string]interface{}{
		"available":   true,
		"version":     updateInfo.Version,
		"description": updateInfo.Description,
		"size":        updateInfo.Size,
	}, nil
}

// UpdateApplication starts the update process
func (a *App) UpdateApplication() error {
	// Use the public method that checks and prompts for updates
	go a.updater.CheckAndPromptForUpdate()
	return nil
}

// ShowNotification displays a system notification
func (a *App) ShowNotification(title, message string) {
	if a.ctx == nil {
		return
	}

	go func() {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   title,
			Message: message,
		})
	}()
}

// GetServerStatus returns detailed server status information
func (a *App) GetServerStatus() map[string]interface{} {
	if a.processManager == nil {
		return map[string]interface{}{
			"running":      false,
			"url":          "",
			"restarts":     0,
			"error":        "Process manager not initialized",
		}
	}

	serverPort, p2pPort := a.processManager.GetPorts()
	return map[string]interface{}{
		"running":      a.processManager.IsRunning(),
		"url":          a.processManager.GetServerURL(),
		"serverPort":   serverPort,
		"p2pPort":      p2pPort,
		"restarts":     a.processManager.GetRestartCount(),
		"shuttingDown": a.shuttingDown,
	}
}

// ConfigureAutoRestart configures the auto-restart behavior
func (a *App) ConfigureAutoRestart(enabled bool, maxRestarts int) {
	if a.processManager != nil {
		a.processManager.SetAutoRestart(enabled, maxRestarts)
	}
}
