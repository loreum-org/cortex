#!/bin/bash

# Make the script executable
# chmod +x scripts/demo.sh

# Build the Cortex node
echo "Building Cortex node..."
go build -o cortex ./cmd/cortexd

# Run the tests first
echo "Running tests..."
./cortex test

# Kill any existing Cortex process
echo "Stopping any existing Cortex nodes..."
pkill -f "cortex serve" || true
sleep 1

# Start the Cortex node in the background
echo "Starting Cortex node..."
./cortex serve &
PID=$!
sleep 2

# Check if the node is running
echo "Checking node health..."
curl -s http://localhost:8080/health

# Get node info
echo -e "\n\nGetting node info..."
curl -s http://localhost:8080/node/info

# Create a RAG document
echo -e "\n\nAdding document to RAG system..."
curl -s -X POST -H "Content-Type: application/json" -d '{"text":"Loreum Cortex is a decentralized AI inference network."}' http://localhost:8080/rag/documents

# Query the RAG system
echo -e "\n\nQuerying the RAG system..."
curl -s -X POST -H "Content-Type: application/json" -d '{"text":"What is Loreum Cortex?"}' http://localhost:8080/rag/query

# Submit a query to the Agent Hub
echo -e "\n\nSubmitting query to Agent Hub..."
curl -s -X POST -H "Content-Type: application/json" -d '{"text":"How does DAG-aBFT consensus work?", "type":"question", "metadata":{}}' http://localhost:8080/queries

# Shut down the Cortex node
echo -e "\n\nShutting down Cortex node..."
kill $PID

echo -e "\nDemo completed!" 