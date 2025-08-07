.PHONY: build clean test run lint serve-bg serve stop restart restart-bg status

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
BINARY_NAME = cortex

all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/cortexd

test:
	$(GOTEST) -v ./...

run:
	./$(BINARY_NAME) serve

clean:
	rm -f $(BINARY_NAME)

lint:
	gofmt -w -s .
	go vet ./...

tidy:
	go mod tidy

# Run cortexd server in background
serve-bg:
	@echo "Checking for existing Cortex processes..."
	@pkill -f cortex 2>/dev/null || echo "No existing cortex process found"
	@lsof -ti:4891 | xargs kill -9 2>/dev/null || echo "Port 4891 is free"
	@echo "Starting Cortex server in background..."
	@nohup ./cortexd serve --port 4891 --p2p-port 4001 > cortex.log 2>&1 &
	@echo "Cortex server started in background, logs in cortex.log"
	@echo "To stop: make stop"

# Run cortexd server in foreground (for development)
serve:
	@echo "Checking for existing Cortex processes..."
	@pkill -f cortex 2>/dev/null || echo "No existing cortex process found"
	@lsof -ti:4891 | xargs kill -9 2>/dev/null || echo "Port 4891 is free"
	@echo "Starting Cortex server in foreground..."
	./cortexd serve --port 4891 --p2p-port 4001

# Stop background cortexd server
stop:
	@echo "Stopping Cortex server..."
	@pkill -f cortex || echo "No cortex process found"
	@lsof -ti:4891 | xargs kill -9 2>/dev/null || echo "Port 4891 is free"
	@lsof -ti:4001 | xargs kill -9 2>/dev/null || echo "Port 4001 is free"

# Restart server (stop + build + serve)
restart: stop build serve

# Clean restart (stop + clean + build + serve-bg)  
restart-bg: stop clean build serve-bg

# Check if server is running
status:
	@echo "Checking Cortex server status..."
	@ps aux | grep -v grep | grep cortex || echo "No cortex process running"
	@lsof -i:4891 || echo "Port 4891 is free"
	@lsof -i:4001 || echo "Port 4001 is free" 