# Desktop Package

This package provides desktop-specific functionality for the Cortex application using the Wails framework.

## Architecture

The desktop app follows a **process-based architecture**:

```
Desktop App (Wails)
    ↓ manages process
cortexd server (port 4891)
    ↓ HTTP/WebSocket  
React Frontend
```

## Components

### `process_manager.go`
- Manages the `cortexd serve` process lifecycle
- Handles start, stop, restart operations
- Configures server and P2P ports (4891, 4001)
- Monitors process health

### `updater.go`
- Checks for application updates from GitHub releases
- Downloads and applies updates
- Provides update notifications to users

### System Integration
- Native desktop wrapper with Wails v2
- Automatic server process management
- Update notifications and management

## Key Features

1. **Process Isolation**: Server runs as separate process, providing stability
2. **Auto-start**: Automatically starts cortexd on app launch
3. **Health Monitoring**: Monitors server process and shows status
4. **Update Management**: Automatic update checking and installation
5. **Native Integration**: System tray, notifications, native dialogs

## Usage

The desktop app exposes minimal methods to the frontend:
- `IsServerRunning()` - Check if cortexd is running
- `GetServerURL()` - Get the server URL (http://localhost:4891)
- `RestartServer()` - Restart the cortexd process
- `CheckForUpdates()` - Check for app updates
- `ShowNotification()` - Display system notifications

All Cortex functionality (queries, RAG, economic engine, etc.) is accessed through the React frontend's direct API calls to the cortexd server.