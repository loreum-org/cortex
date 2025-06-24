# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Run
- `go build -o cortex ./cmd/cortexd` - Build the main executable
- `make build` - Build using Makefile
- `./cortex serve --port 8080` - Start the Cortex node (default port 8080)
- `./cortex serve --port 8080 --p2p-port 4001 --bootstrap-peers <peer1>,<peer2>` - Start with P2P configuration
- `make run` - Build and run with default settings

### Testing and Quality
- `go test -v ./...` - Run all tests with verbose output
- `make test` - Run tests using Makefile
- `go test ./internal/api` - Run tests for specific package
- `make lint` - Format code with gofmt and run go vet
- `go mod tidy` - Clean up dependencies

### Development Tools
- `./cortex test` - Run built-in integration tests
- `make clean` - Remove build artifacts

## Architecture Overview

Loreum Cortex is a decentralized AI inference network with a three-layer architecture:

### Core Components
1. **Network Layer** (`internal/p2p`, `internal/consensus`, `internal/api`)
   - P2P communication using libp2p
   - DAG-aBFT consensus mechanism
   - REST API Gateway with metrics and monitoring

2. **Business Layer** (`internal/agenthub`, `internal/sensorhub`, `internal/rag`, `internal/economy`)
   - Agent Hub with solver agents for query processing
   - Sensor Hub for blockchain monitoring
   - RAG system with vector database integration
   - Economic engine for payment processing and rewards

3. **Data Layer** (`internal/storage`, `pkg/types`)
   - SQL and Redis storage adapters
   - Shared type definitions for agents, transactions, and network components

### Key Integrations
- **libp2p**: Handles P2P networking, peer discovery, and pubsub messaging
- **Economic System**: Processes payments for queries and distributes rewards to nodes
- **RAG System**: Vector-based document storage and retrieval with 384-dimensional embeddings
- **Consensus**: DAG-based transaction validation and ordering

### Entry Points
- `cmd/cortexd/main.go`: Main application with CLI using cobra
- `internal/api/server.go`: HTTP API server with comprehensive endpoints
- Node initialization creates P2P networking, consensus service, solver agents, RAG system, and economic engine

### Testing Strategy
- Unit tests in `*_test.go` files alongside source code
- Integration tests accessible via `./cortex test` command
- Test helpers in `internal/rag/test_helpers.go`