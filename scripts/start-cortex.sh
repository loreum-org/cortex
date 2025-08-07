#!/bin/bash

# Cortex Node Startup Script
# Usage: ./scripts/start-cortex.sh [background|foreground|stop|status]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY_PATH="$PROJECT_ROOT/cortexd"
PID_FILE="$PROJECT_ROOT/cortex.pid"
LOG_FILE="$PROJECT_ROOT/cortex.log"

# Default ports
PORT=${CORTEX_PORT:-4891}
P2P_PORT=${CORTEX_P2P_PORT:-4001}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if cortexd binary exists
check_binary() {
    if [[ ! -f "$BINARY_PATH" ]]; then
        print_error "cortexd binary not found at $BINARY_PATH"
        print_status "Run 'make build' to build the binary first"
        exit 1
    fi
}

# Start cortex in background
start_background() {
    check_binary
    
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            print_warning "Cortex is already running (PID: $pid)"
            return 0
        else
            print_warning "Stale PID file found, removing it"
            rm "$PID_FILE"
        fi
    fi
    
    print_status "Starting Cortex node in background..."
    print_status "API Port: $PORT, P2P Port: $P2P_PORT"
    print_status "Logs: $LOG_FILE"
    
    # Start the process in background and save PID
    nohup "$BINARY_PATH" serve --port "$PORT" --p2p-port "$P2P_PORT" > "$LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$PID_FILE"
    
    # Wait a moment and check if process is still running
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        print_status "Cortex node started successfully (PID: $pid)"
        print_status "Use 'tail -f $LOG_FILE' to view logs"
        print_status "Use '$0 stop' to stop the node"
    else
        print_error "Failed to start Cortex node"
        rm -f "$PID_FILE"
        exit 1
    fi
}

# Start cortex in foreground
start_foreground() {
    check_binary
    print_status "Starting Cortex node in foreground..."
    print_status "API Port: $PORT, P2P Port: $P2P_PORT"
    print_status "Press Ctrl+C to stop"
    exec "$BINARY_PATH" serve --port "$PORT" --p2p-port "$P2P_PORT"
}

# Stop cortex
stop_cortex() {
    if [[ ! -f "$PID_FILE" ]]; then
        print_warning "No PID file found, trying to kill by process name"
        if pkill -f "cortexd serve"; then
            print_status "Cortex processes stopped"
        else
            print_warning "No cortexd processes found"
        fi
        return 0
    fi
    
    local pid=$(cat "$PID_FILE")
    if kill -0 "$pid" 2>/dev/null; then
        print_status "Stopping Cortex node (PID: $pid)..."
        kill "$pid"
        
        # Wait for graceful shutdown
        local count=0
        while kill -0 "$pid" 2>/dev/null && [[ $count -lt 10 ]]; do
            sleep 1
            count=$((count + 1))
        done
        
        if kill -0 "$pid" 2>/dev/null; then
            print_warning "Process didn't stop gracefully, force killing..."
            kill -9 "$pid"
        fi
        
        rm -f "$PID_FILE"
        print_status "Cortex node stopped"
    else
        print_warning "Process not running, cleaning up PID file"
        rm -f "$PID_FILE"
    fi
}

# Show status
show_status() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            print_status "Cortex node is running (PID: $pid)"
            print_status "API Port: $PORT, P2P Port: $P2P_PORT"
            print_status "Log file: $LOG_FILE"
            return 0
        else
            print_warning "PID file exists but process is not running"
            rm -f "$PID_FILE"
        fi
    fi
    
    print_status "Cortex node is not running"
    return 1
}

# Show help
show_help() {
    echo "Cortex Node Startup Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  background  - Start Cortex node in background (default)"
    echo "  foreground  - Start Cortex node in foreground"
    echo "  stop        - Stop Cortex node"
    echo "  status      - Show Cortex node status"
    echo "  help        - Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  CORTEX_PORT      - API server port (default: 4891)"
    echo "  CORTEX_P2P_PORT  - P2P network port (default: 4001)"
    echo ""
}

# Main script logic
case "${1:-background}" in
    background|bg)
        start_background
        ;;
    foreground|fg)
        start_foreground
        ;;
    stop)
        stop_cortex
        ;;
    status)
        show_status
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac 