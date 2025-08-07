package desktop

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// Update server configuration
	UpdateServerURL = "https://api.github.com/repos/loreum-org/cortex/releases/latest"
	CurrentVersion  = "v0.1.0" // This should be set during build
)

// UpdateInfo represents information about an available update
type UpdateInfo struct {
	Version     string `json:"tag_name"`
	Description string `json:"body"`
	DownloadURL string `json:"download_url"`
	Checksum    string `json:"checksum"`
	Size        int64  `json:"size"`
}

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// Updater handles application updates
type Updater struct {
	ctx       context.Context
	app       interface{}
	updateURL string
}

// NewUpdater creates a new updater instance
func NewUpdater(app interface{}) *Updater {
	return &Updater{
		app:       app,
		updateURL: UpdateServerURL,
	}
}

// SetContext sets the context for the updater
func (u *Updater) SetContext(ctx context.Context) {
	u.ctx = ctx
}

// CheckForUpdates checks if a new version is available
func (u *Updater) CheckForUpdates() (*UpdateInfo, error) {
	resp, err := http.Get(u.updateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update server returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse update response: %v", err)
	}

	// Check if this is a newer version
	if !isNewerVersion(release.TagName, CurrentVersion) {
		return nil, nil // No update available
	}

	// Find the appropriate asset for current platform
	assetName := getAssetNameForPlatform()
	var downloadURL string
	var size int64

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, assetName) {
			downloadURL = asset.BrowserDownloadURL
			size = asset.Size
			break
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("no compatible update found for platform %s", runtime.GOOS)
	}

	return &UpdateInfo{
		Version:     release.TagName,
		Description: release.Body,
		DownloadURL: downloadURL,
		Size:        size,
	}, nil
}

// DownloadUpdate downloads the update file
func (u *Updater) DownloadUpdate(updateInfo *UpdateInfo, progressCallback func(int)) (string, error) {
	// Create temporary directory for download
	tempDir := os.TempDir()
	filename := filepath.Base(updateInfo.DownloadURL)
	tempFile := filepath.Join(tempDir, filename)

	// Download the file
	resp, err := http.Get(updateInfo.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download server returned status %d", resp.StatusCode)
	}

	// Create the temporary file
	out, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer out.Close()

	// Download with progress tracking
	var downloaded int64
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := out.Write(buffer[:n]); writeErr != nil {
				return "", fmt.Errorf("failed to write to temp file: %v", writeErr)
			}
			downloaded += int64(n)
			
			// Report progress
			if progressCallback != nil && updateInfo.Size > 0 {
				progress := int((downloaded * 100) / updateInfo.Size)
				progressCallback(progress)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read download: %v", err)
		}
	}

	return tempFile, nil
}

// VerifyUpdate verifies the downloaded update file
func (u *Updater) VerifyUpdate(filePath string, expectedChecksum string) error {
	if expectedChecksum == "" {
		// Skip verification if no checksum provided
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open update file: %v", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %v", err)
	}

	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// ApplyUpdate applies the downloaded update
func (u *Updater) ApplyUpdate(updateFilePath string) error {
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Create backup of current executable
	backupPath := currentExe + ".backup"
	if err := copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Replace current executable with update
	if err := copyFile(updateFilePath, currentExe); err != nil {
		// Restore backup on failure
		copyFile(backupPath, currentExe)
		return fmt.Errorf("failed to apply update: %v", err)
	}

	// Make executable (Unix systems)
	if runtime.GOOS != "windows" {
		os.Chmod(currentExe, 0755)
	}

	// Clean up
	os.Remove(updateFilePath)
	os.Remove(backupPath)

	return nil
}

// RestartApplication restarts the application after update
func (u *Updater) RestartApplication() error {
	if u.ctx == nil {
		return fmt.Errorf("no context available for restart")
	}

	// Show restart message to user
	_, err := wailsruntime.MessageDialog(u.ctx, wailsruntime.MessageDialogOptions{
		Type:    wailsruntime.InfoDialog,
		Title:   "Update Complete",
		Message: "Cortex has been updated successfully. The application will restart now.",
	})
	if err != nil {
		return err
	}

	// Quit the current application - the system will handle restart
	wailsruntime.Quit(u.ctx)
	return nil
}

// CheckAndPromptForUpdate checks for updates and prompts user
func (u *Updater) CheckAndPromptForUpdate() {
	updateInfo, err := u.CheckForUpdates()
	if err != nil {
		wailsruntime.LogError(u.ctx, fmt.Sprintf("Failed to check for updates: %v", err))
		return
	}

	if updateInfo == nil {
		// No update available
		return
	}

	// Prompt user for update
	result, err := wailsruntime.MessageDialog(u.ctx, wailsruntime.MessageDialogOptions{
		Type:    wailsruntime.QuestionDialog,
		Title:   "Update Available",
		Message: fmt.Sprintf("A new version (%s) is available. Would you like to update now?\n\nChanges:\n%s", updateInfo.Version, updateInfo.Description),
		Buttons: []string{"Update Now", "Later"},
	})

	if err != nil || result != "Update Now" {
		return
	}

	// Perform update
	go u.performUpdate(updateInfo)
}

// performUpdate performs the update process
func (u *Updater) performUpdate(updateInfo *UpdateInfo) {
	// Show progress dialog
	wailsruntime.EventsEmit(u.ctx, "update:started", updateInfo)

	// Download update
	tempFile, err := u.DownloadUpdate(updateInfo, func(progress int) {
		wailsruntime.EventsEmit(u.ctx, "update:progress", progress)
	})
	if err != nil {
		wailsruntime.EventsEmit(u.ctx, "update:error", err.Error())
		return
	}

	// Verify update
	if err := u.VerifyUpdate(tempFile, updateInfo.Checksum); err != nil {
		wailsruntime.EventsEmit(u.ctx, "update:error", err.Error())
		os.Remove(tempFile)
		return
	}

	// Apply update
	if err := u.ApplyUpdate(tempFile); err != nil {
		wailsruntime.EventsEmit(u.ctx, "update:error", err.Error())
		return
	}

	// Restart application
	wailsruntime.EventsEmit(u.ctx, "update:complete", updateInfo.Version)
	time.Sleep(2 * time.Second) // Give UI time to show completion
	u.RestartApplication()
}

// Helper functions

func isNewerVersion(newVersion, currentVersion string) bool {
	// Simple version comparison - remove 'v' prefix and compare
	newVer := strings.TrimPrefix(newVersion, "v")
	currentVer := strings.TrimPrefix(currentVersion, "v")
	return newVer > currentVer
}

func getAssetNameForPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	default:
		return runtime.GOOS
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}