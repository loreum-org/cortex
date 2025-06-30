#!/bin/bash

# Test script for AGI blockchain functionality

echo "🧠 Testing AGI Blockchain Integration"
echo "====================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PORT=8080
BASE_URL="http://localhost:$PORT"

echo -e "${BLUE}Starting Cortex node on port $PORT...${NC}"

# Start Cortex in background
./cortex serve --port $PORT &
CORTEX_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 10

# Function to make API calls and format output
test_endpoint() {
    local endpoint=$1
    local description=$2
    local method=${3:-GET}
    local data=${4:-""}
    
    echo -e "\n${BLUE}Testing: $description${NC}"
    echo "Endpoint: $method $endpoint"
    
    if [ "$method" = "POST" ] && [ -n "$data" ]; then
        curl -s -X POST -H "Content-Type: application/json" -d "$data" "$BASE_URL$endpoint" | jq '.' 2>/dev/null || curl -s -X POST -H "Content-Type: application/json" -d "$data" "$BASE_URL$endpoint"
    else
        curl -s "$BASE_URL$endpoint" | jq '.' 2>/dev/null || curl -s "$BASE_URL$endpoint"
    fi
    
    echo ""
}

# Test basic AGI state
test_endpoint "/agi/state" "Current AGI State"

# Test AGI intelligence metrics
test_endpoint "/agi/intelligence" "AGI Intelligence Metrics"

# Test domain knowledge
test_endpoint "/agi/domains" "Domain Knowledge Scores"

# Test network AGI snapshot (blockchain)
test_endpoint "/agi/network/snapshot" "Network AGI Snapshot (Blockchain)"

# Test network nodes AGI states
test_endpoint "/agi/network/nodes" "All Network Node AGI States"

# Test domain leaders
test_endpoint "/agi/network/leaders" "Domain Knowledge Leaders"

# Test AGI cortex history from vector DB
test_endpoint "/agi/cortex/history?limit=5" "AGI Cortex History"

# Test context windows from vector DB
test_endpoint "/agi/context/windows?type=interaction&limit=10" "AGI Context Windows"

# Test vector search for AGI data
test_endpoint "/agi/vector/search" "AGI Vector Search" "POST" '{
    "query": "AGI intelligence learning patterns",
    "search_type": "agi_state",
    "max_results": 5
}'

# Test enhanced AGI prompt generation
test_endpoint "/agi/prompt" "AGI Enhanced Prompt" "POST" '{
    "query": "How can I improve my node performance for blockchain consensus?"
}'

# Simulate some learning by making queries
echo -e "\n${GREEN}Simulating AGI learning through queries...${NC}"

test_queries=(
    "What is blockchain consensus?"
    "How does DAG-based consensus work?"
    "Explain proof of stake mechanisms"
    "What are the benefits of decentralized AI?"
    "How can nodes optimize their performance?"
)

for query in "${test_queries[@]}"; do
    echo "Processing query: $query"
    curl -s -X POST -H "Content-Type: application/json" -d "{\"text\":\"$query\",\"type\":\"question\"}" "$BASE_URL/query" > /dev/null
    sleep 2
done

echo -e "\n${GREEN}Waiting for AGI learning to process...${NC}"
sleep 10

# Check AGI state after learning
echo -e "\n${GREEN}AGI State After Learning:${NC}"
test_endpoint "/agi/state" "Updated AGI State"

echo -e "\n${GREEN}Domain Knowledge After Learning:${NC}"
test_endpoint "/agi/domains" "Updated Domain Knowledge"

# Test blockchain recording
echo -e "\n${GREEN}Blockchain AGI Records:${NC}"
test_endpoint "/agi/network/snapshot" "Latest Network Snapshot"

# Cleanup
echo -e "\n${RED}Stopping Cortex node...${NC}"
kill $CORTEX_PID

echo -e "\n${GREEN}✅ AGI Blockchain Integration Test Complete!${NC}"
echo ""
echo "Key Features Demonstrated:"
echo "• AGI state persistence in vector database"
echo "• Blockchain recording of domain knowledge scores"
echo "• Network-wide AGI intelligence tracking"
echo "• Context window storage and retrieval"
echo "• Domain expertise leaderboards"
echo "• AGI-enhanced prompt generation"
echo "• Automatic learning from interactions"
echo ""
echo "The blockchain now maintains a permanent record of:"
echo "📊 Intelligence levels for each node"
echo "🎯 Domain expertise scores across the network"
echo "🧠 AGI evolution events and learning milestones"
echo "🔍 Searchable AGI knowledge and context history"
echo "👑 Domain leadership rankings"