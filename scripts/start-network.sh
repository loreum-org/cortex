#!/bin/bash

# Cortex P2P Network Startup Script
# This script launches multiple Cortex nodes and connects them in a P2P network

# Default number of nodes to start
NUM_NODES=3
# Base directories for node data
BASE_DIR="/tmp/cortex-network"
# Path to the Cortex executable
CORTEX_BIN="./cortex"
# Array to store node PIDs
declare -a NODE_PIDS

# Color codes for pretty output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to show usage information
show_usage() {
  echo -e "${BLUE}Cortex P2P Network Startup Script${NC}"
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  -n, --nodes NUM     Number of nodes to start (2-5, default: 3)"
  echo "  -h, --help          Show this help message"
  echo ""
  echo "Example:"
  echo "  $0 --nodes 4        Start a network with 4 nodes"
}

# Function to clean up on exit
cleanup() {
  echo -e "\n${YELLOW}Shutting down Cortex network...${NC}"
  
  # Kill all node processes
  for pid in "${NODE_PIDS[@]}"; do
    if ps -p $pid > /dev/null; then
      echo -e "Stopping node with PID ${YELLOW}$pid${NC}..."
      kill $pid
    fi
  done
  
  # Wait for processes to terminate
  wait
  
  echo -e "${GREEN}All nodes stopped. Network shutdown complete.${NC}"
  exit 0
}

# Function to get peer address from a node
get_peer_address() {
  local api_port=$1
  local max_attempts=10
  local attempt=1
  
  echo -e "${YELLOW}Waiting for node on port $api_port to start...${NC}"
  
  while [ $attempt -le $max_attempts ]; do
    # Check if the node is up by querying its health endpoint
    if curl -s "http://localhost:$api_port/health" > /dev/null; then
      # Get node info with peer ID and addresses
      local node_info=$(curl -s "http://localhost:$api_port/node/info")
      local peer_id=$(echo $node_info | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
      
      # Get the first TCP address (usually the most reliable for connections)
      local addresses=$(echo $node_info | grep -o '"addresses":\[[^]]*\]' | grep -o '/ip4/[^"]*')
      local tcp_address=$(echo "$addresses" | grep "/tcp/" | head -1)
      
      if [ -n "$peer_id" ] && [ -n "$tcp_address" ]; then
        # Construct the full multiaddress
        echo "/ip4/127.0.0.1/tcp/$(echo $tcp_address | grep -o 'tcp/[0-9]*' | cut -d'/' -f2)/p2p/$peer_id"
        return 0
      fi
    fi
    
    echo -e "${YELLOW}Waiting for node to initialize (attempt $attempt/$max_attempts)...${NC}"
    attempt=$((attempt + 1))
    sleep 2
  done
  
  echo -e "${RED}Failed to get peer address after $max_attempts attempts${NC}" >&2
  return 1
}

# Function to start a node
start_node() {
  local node_num=$1
  local p2p_port=$2
  local api_port=$3
  local bootstrap_peers=$4
  local node_name="cortex-node-$node_num"
  local node_dir="$BASE_DIR/node-$node_num"
  
  # Create node directory
  mkdir -p "$node_dir"
  
  # Build the command
  local cmd="$CORTEX_BIN serve --port $api_port --p2p-port $p2p_port --node-name $node_name"
  
  # Add bootstrap peers if provided
  if [ -n "$bootstrap_peers" ]; then
    cmd="$cmd --bootstrap-peers $bootstrap_peers"
  fi
  
  # Start the node
  echo -e "${GREEN}Starting $node_name on P2P port $p2p_port and API port $api_port...${NC}"
  if [ -n "$bootstrap_peers" ]; then
    echo -e "${BLUE}Connecting to bootstrap peers: $bootstrap_peers${NC}"
  else
    echo -e "${PURPLE}This is a bootstrap node (no peers)${NC}"
  fi
  
  # Run the node in the background and capture its PID
  $cmd > "$node_dir/node.log" 2>&1 &
  local pid=$!
  NODE_PIDS+=($pid)
  
  echo -e "${GREEN}Node $node_name started with PID $pid${NC}"
  echo -e "  - API URL: ${CYAN}http://localhost:$api_port${NC}"
  echo -e "  - Dashboard: ${CYAN}http://localhost:$api_port/dashboard.html${NC}"
  echo -e "  - Log file: ${CYAN}$node_dir/node.log${NC}"
  
  return 0
}

# Function to check network status
check_network_status() {
  echo -e "\n${BLUE}Checking network status...${NC}"
  
  for i in $(seq 1 $NUM_NODES); do
    local api_port=$((8080 + i - 1))
    
    echo -e "\n${YELLOW}Node $i (API port $api_port):${NC}"
    
    # Get peer count
    local peer_info=$(curl -s "http://localhost:$api_port/node/peers" 2>/dev/null)
    if [ $? -ne 0 ]; then
      echo -e "  ${RED}Node not responding${NC}"
      continue
    fi
    
    local peer_count=$(echo "$peer_info" | grep -o '"id"' | wc -l)
    echo -e "  Connected to ${GREEN}$peer_count peers${NC}"
    
    # Get network metrics
    local network_metrics=$(curl -s "http://localhost:$api_port/metrics/network" 2>/dev/null)
    if [ -n "$network_metrics" ]; then
      local bytes_received=$(echo "$network_metrics" | grep -o '"bytes_received":[0-9]*' | cut -d':' -f2)
      local bytes_sent=$(echo "$network_metrics" | grep -o '"bytes_sent":[0-9]*' | cut -d':' -f2)
      
      echo -e "  Bytes received: ${CYAN}$bytes_received${NC}"
      echo -e "  Bytes sent: ${CYAN}$bytes_sent${NC}"
    fi
  done
}

# Function to test the network by sending a query to one node and checking if it propagates
test_network() {
  echo -e "\n${BLUE}Testing network communication...${NC}"
  
  # Send a query to the first node
  local query_text="What is the benefit of decentralized AI inference networks?"
  echo -e "${YELLOW}Sending query to node 1: ${CYAN}$query_text${NC}"
  
  curl -s -X POST "http://localhost:8080/queries" \
    -H "Content-Type: application/json" \
    -d "{\"text\": \"$query_text\", \"type\": \"question\"}" > /dev/null
  
  # Wait a moment for propagation
  sleep 2
  
  # Check events on all nodes to see if they received the query
  for i in $(seq 1 $NUM_NODES); do
    local api_port=$((8080 + i - 1))
    
    echo -e "\n${YELLOW}Checking events on node $i (API port $api_port):${NC}"
    
    local events=$(curl -s "http://localhost:$api_port/events?limit=5" 2>/dev/null)
    if [ $? -ne 0 ]; then
      echo -e "  ${RED}Node not responding${NC}"
      continue
    fi
    
    # Check if there are query-related events
    if echo "$events" | grep -q "query"; then
      echo -e "  ${GREEN}✓ Node received query events${NC}"
    else
      echo -e "  ${RED}✗ No query events found${NC}"
    fi
  done
}

# Function to show dashboard URLs
show_dashboards() {
  echo -e "\n${BLUE}Dashboard URLs:${NC}"
  for i in $(seq 1 $NUM_NODES); do
    local api_port=$((8080 + i - 1))
    echo -e "  Node $i: ${CYAN}file://$PWD/web/dashboard.html?port=$api_port${NC}"
  done
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--nodes)
      NUM_NODES="$2"
      if ! [[ "$NUM_NODES" =~ ^[2-5]$ ]]; then
        echo -e "${RED}Error: Number of nodes must be between 2 and 5${NC}" >&2
        exit 1
      fi
      shift 2
      ;;
    -h|--help)
      show_usage
      exit 0
      ;;
    *)
      echo -e "${RED}Error: Unknown option $1${NC}" >&2
      show_usage
      exit 1
      ;;
  esac
done

# Set up trap for cleanup on script exit
trap cleanup SIGINT SIGTERM

# Create base directory
mkdir -p "$BASE_DIR"

# Clear any existing log files
rm -f "$BASE_DIR"/node-*/node.log

# Start the bootstrap node (node 1)
start_node 1 4001 8080 ""

# Get the bootstrap node's peer address
bootstrap_peer=$(get_peer_address 8080)

if [ -z "$bootstrap_peer" ]; then
  echo -e "${RED}Failed to get bootstrap peer address. Exiting.${NC}" >&2
  cleanup
  exit 1
fi

echo -e "${GREEN}Bootstrap peer address: ${CYAN}$bootstrap_peer${NC}"

# Start additional nodes
for i in $(seq 2 $NUM_NODES); do
  p2p_port=$((4000 + i))
  api_port=$((8080 + i - 1))
  start_node $i $p2p_port $api_port "$bootstrap_peer"
  
  # Give each node a moment to start
  sleep 2
done

# Wait for all nodes to connect
echo -e "\n${YELLOW}Waiting for nodes to connect...${NC}"
sleep 10

# Check network status
check_network_status

# Test the network
test_network

# Show dashboard URLs
show_dashboards

echo -e "\n${GREEN}P2P Network is running with $NUM_NODES nodes${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop all nodes and exit${NC}"

# Create a modified dashboard file for each node
for i in $(seq 1 $NUM_NODES); do
  local api_port=$((8080 + i - 1))
  mkdir -p "$BASE_DIR/node-$i/web"
  sed "s|http://localhost:8080|http://localhost:$api_port|g" web/dashboard.html > "$BASE_DIR/node-$i/web/dashboard.html"
  echo -e "Node $i dashboard: file://$BASE_DIR/node-$i/web/dashboard.html"
done

# Keep the script running until Ctrl+C
while true; do
  sleep 10
  check_network_status
done
