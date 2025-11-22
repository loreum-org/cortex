package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// main starts the Wails application
func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Loreum AI",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:         app.OnStartup,
		OnDomReady:        app.OnDomReady,
		OnShutdown:        app.OnShutdown,
		Frameless:         false,
		MinWidth:          800,
		MinHeight:         600,
		HideWindowOnClose: false,
		StartHidden:       false,
		DisableResize:     false,
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
		CSSDragProperty: "drag",
		CSSDragValue:    "drag",
		Logger:          nil,
		LogLevel:        0,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// Check if server operations are in progress
			status := app.GetServerStatus()
			if status["running"].(bool) && !status["shuttingDown"].(bool) {
				// Ask user to confirm exit
				result, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
					Type:          runtime.QuestionDialog,
					Title:         "Confirm Exit",
					Message:       "The Cortex server is still running. Are you sure you want to exit?",
					DefaultButton: "No",
				})
				if err != nil {
					log.Printf("Error showing exit confirmation dialog: %v", err)
					return false // Allow exit if dialog fails
				}
				// Prevent exit if user clicks "No" 
				return result != "Yes"
			}
			return false
		},
	})

	if err != nil {
		log.Fatalf("Error starting Cortex Desktop: %v", err)
	}
}
