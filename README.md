# Loreum Cortex

Loreum Cortex is a decentralized AI inference network implemented in Go. It provides a robust p2p network that supports the DAG-aBFT consensus mechanism and enables core functions such as Sensor Hub, Agent Hub, and RAG (Retrieval-Augmented Generation) system.

## System Architecture

The Loreum Cortex node consists of three primary layers:

1. **Network Layer**: Handles P2P communication, API Gateway, and Consensus mechanism
2. **Business Layer**: Implements Agent Hub, Sensor Hub, and RAG system
3. **Data Layer**: Manages data storage with SQL, Redis, and Vector databases

## Getting Started

### Prerequisites

- Go 1.20 or higher
- Docker and Docker Compose
- PostgreSQL 14+
- Redis 7+
- Vector database (Milvus/Qdrant/Weaviate)

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/loreum-org/cortex.git
   cd cortex
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Build the project:
   ```
   go build -o cortex ./cmd/cortexd
   ```

4. Run the node:
   ```
   ./cortex serve
   ```

## Development

### Project Structure

```
cortex/
├── cmd/
│   └── cortexd/               # Main executable
├── internal/
│   ├── api/                   # API Gateway
│   ├── consensus/             # DAG-aBFT implementation
│   ├── p2p/                   # P2P networking
│   ├── agenthub/              # Agent implementations
│   ├── sensorhub/             # Sensor implementations
│   ├── rag/                   # RAG system
│   ├── storage/               # Data storage
│   └── reputation/            # Reputation system
├── pkg/
│   ├── types/                 # Shared data types
│   ├── crypto/                # Cryptographic utilities
│   ├── config/                # Configuration management
│   └── util/                  # Shared utilities
├── test/                      # Test suites
├── scripts/                   # Deployment scripts
└── docs/                      # Documentation
```

## License

[MIT License](LICENSE) 