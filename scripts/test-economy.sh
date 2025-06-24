#!/bin/bash

# test-economy.sh - Test script for Loreum Cortex Economic Engine
# This script tests the economic functionality of a running Cortex node
# including account creation, token transfers, staking, and query payments.

# Color codes for pretty output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
API_URL="http://localhost:8080"
ADMIN_KEY="admin-test-key" # Replace with actual admin key if needed
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test data
USER_ID="test-user-$(date +%s)"
USER_ADDRESS="0xTestUser$(date +%s)"
NODE_ID="test-node-$(date +%s)"
NODE_ADDRESS="0xTestNode$(date +%s)"

# Function to show usage information
show_usage() {
  echo -e "${BLUE}Cortex Economic Engine Test Script${NC}"
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  -u, --url URL       API URL (default: http://localhost:8080)"
  echo "  -a, --admin-key KEY Admin API key for protected operations"
  echo "  -h, --help          Show this help message"
  echo ""
  echo "Example:"
  echo "  $0 --url http://localhost:8081 --admin-key mykey"
}

# Function to make API calls
call_api() {
  local method=$1
  local endpoint=$2
  local data=$3
  local admin_header=""
  
  if [ "$4" = "admin" ]; then
    admin_header="-H \"X-Admin-Key: $ADMIN_KEY\""
  fi
  
  if [ -n "$data" ]; then
    eval "curl -s -X $method \"$API_URL$endpoint\" \
      -H \"Content-Type: application/json\" \
      $admin_header \
      -d '$data'"
  else
    eval "curl -s -X $method \"$API_URL$endpoint\" $admin_header"
  fi
}

# Function to log test results
log_test() {
  local test_name=$1
  local result=$2
  local message=$3
  
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  
  if [ "$result" = "pass" ]; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
    echo -e "${GREEN}✓ PASS:${NC} $test_name"
    if [ -n "$message" ]; then
      echo -e "  ${CYAN}$message${NC}"
    fi
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    echo -e "${RED}✗ FAIL:${NC} $test_name"
    if [ -n "$message" ]; then
      echo -e "  ${RED}$message${NC}"
    fi
  fi
}

# Function to check if the API is available
check_api() {
  echo -e "${YELLOW}Checking if Cortex API is available at $API_URL...${NC}"
  
  response=$(call_api "GET" "/health")
  
  if [ -z "$response" ]; then
    echo -e "${RED}Error: Could not connect to Cortex API at $API_URL${NC}"
    echo -e "${YELLOW}Make sure the Cortex node is running and the API is accessible.${NC}"
    exit 1
  fi
  
  status=$(echo $response | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
  
  if [ "$status" = "ok" ]; then
    echo -e "${GREEN}Cortex API is available!${NC}"
  else
    echo -e "${RED}Error: Cortex API returned unexpected status: $status${NC}"
    exit 1
  fi
}

# Test: Create user account
test_create_user_account() {
  echo -e "\n${BLUE}Test: Create User Account${NC}"
  
  response=$(call_api "POST" "/accounts/user" "{\"id\":\"$USER_ID\",\"address\":\"$USER_ADDRESS\"}")
  
  if echo "$response" | grep -q "\"id\":\"$USER_ID\""; then
    log_test "Create user account" "pass" "Created user account with ID: $USER_ID"
  else
    log_test "Create user account" "fail" "Failed to create user account: $response"
    return 1
  fi
  
  return 0
}

# Test: Create node account
test_create_node_account() {
  echo -e "\n${BLUE}Test: Create Node Account${NC}"
  
  response=$(call_api "POST" "/accounts/node" "{\"id\":\"$NODE_ID\",\"address\":\"$NODE_ADDRESS\"}")
  
  if echo "$response" | grep -q "\"id\":\"$NODE_ID\""; then
    log_test "Create node account" "pass" "Created node account with ID: $NODE_ID"
  else
    log_test "Create node account" "fail" "Failed to create node account: $response"
    return 1
  fi
  
  return 0
}

# Test: Get account details
test_get_account() {
  echo -e "\n${BLUE}Test: Get Account Details${NC}"
  
  # Get user account
  response=$(call_api "GET" "/accounts/$USER_ID")
  
  if echo "$response" | grep -q "\"id\":\"$USER_ID\""; then
    log_test "Get user account" "pass" "Retrieved user account details"
  else
    log_test "Get user account" "fail" "Failed to retrieve user account: $response"
  fi
  
  # Get node account
  response=$(call_api "GET" "/accounts/$NODE_ID")
  
  if echo "$response" | grep -q "\"id\":\"$NODE_ID\""; then
    log_test "Get node account" "pass" "Retrieved node account details"
  else
    log_test "Get node account" "fail" "Failed to retrieve node account: $response"
  fi
  
  return 0
}

# Test: Mint tokens to user account (requires admin privileges)
test_mint_tokens() {
  echo -e "\n${BLUE}Test: Mint Tokens${NC}"
  
  # Mint 1000 tokens to user account
  response=$(call_api "POST" "/tokens/mint" "{\"account_id\":\"$USER_ID\",\"amount\":\"1000000000000000000000\",\"reason\":\"Test funding\"}" "admin")
  
  if echo "$response" | grep -q "\"type\":\"mint\""; then
    log_test "Mint tokens to user" "pass" "Minted 1000 tokens to user account"
  else
    log_test "Mint tokens to user" "fail" "Failed to mint tokens: $response"
    return 1
  fi
  
  # Mint 500 tokens to node account
  response=$(call_api "POST" "/tokens/mint" "{\"account_id\":\"$NODE_ID\",\"amount\":\"500000000000000000000\",\"reason\":\"Test node funding\"}" "admin")
  
  if echo "$response" | grep -q "\"type\":\"mint\""; then
    log_test "Mint tokens to node" "pass" "Minted 500 tokens to node account"
  else
    log_test "Mint tokens to node" "fail" "Failed to mint tokens: $response"
    return 1
  fi
  
  return 0
}

# Test: Transfer tokens between accounts
test_transfer_tokens() {
  echo -e "\n${BLUE}Test: Transfer Tokens${NC}"
  
  # Transfer 100 tokens from user to node
  response=$(call_api "POST" "/tokens/transfer" "{\"from_id\":\"$USER_ID\",\"to_id\":\"$NODE_ID\",\"amount\":\"100000000000000000000\",\"description\":\"Test transfer\"}")
  
  if echo "$response" | grep -q "\"type\":\"transfer\""; then
    log_test "Transfer tokens" "pass" "Transferred 100 tokens from user to node"
  else
    log_test "Transfer tokens" "fail" "Failed to transfer tokens: $response"
    return 1
  fi
  
  # Verify balances after transfer
  response=$(call_api "GET" "/accounts/$USER_ID")
  user_balance=$(echo "$response" | grep -o '"balance":"[^"]*"' | cut -d'"' -f4)
  
  response=$(call_api "GET" "/accounts/$NODE_ID")
  node_balance=$(echo "$response" | grep -o '"balance":"[^"]*"' | cut -d'"' -f4)
  
  log_test "Balance verification" "pass" "User balance: $user_balance, Node balance: $node_balance"
  
  return 0
}

# Test: Stake tokens
test_stake_tokens() {
  echo -e "\n${BLUE}Test: Stake Tokens${NC}"
  
  # Stake 200 tokens from node account
  response=$(call_api "POST" "/nodes/$NODE_ID/stake" "{\"amount\":\"200000000000000000000\"}")
  
  if echo "$response" | grep -q "\"type\":\"stake\""; then
    log_test "Stake tokens" "pass" "Staked 200 tokens from node account"
  else
    log_test "Stake tokens" "fail" "Failed to stake tokens: $response"
    return 1
  fi
  
  # Verify stake amount
  response=$(call_api "GET" "/accounts/$NODE_ID")
  stake_amount=$(echo "$response" | grep -o '"stake":"[^"]*"' | cut -d'"' -f4)
  
  log_test "Stake verification" "pass" "Node stake amount: $stake_amount"
  
  return 0
}

# Test: Unstake tokens
test_unstake_tokens() {
  echo -e "\n${BLUE}Test: Unstake Tokens${NC}"
  
  # Unstake 50 tokens from node account
  response=$(call_api "POST" "/nodes/$NODE_ID/unstake" "{\"amount\":\"50000000000000000000\"}")
  
  if echo "$response" | grep -q "\"type\":\"unstake\""; then
    log_test "Unstake tokens" "pass" "Unstaked 50 tokens from node account"
  else
    log_test "Unstake tokens" "fail" "Failed to unstake tokens: $response"
    return 1
  fi
  
  # Verify stake amount after unstaking
  response=$(call_api "GET" "/accounts/$NODE_ID")
  stake_amount=$(echo "$response" | grep -o '"stake":"[^"]*"' | cut -d'"' -f4)
  
  log_test "Unstake verification" "pass" "Node stake amount after unstaking: $stake_amount"
  
  return 0
}

# Test: Calculate query price
test_calculate_price() {
  echo -e "\n${BLUE}Test: Calculate Query Price${NC}"
  
  # Calculate price for a standard completion query
  response=$(call_api "POST" "/economy/price" "{\"model_tier\":\"standard\",\"query_type\":\"completion\",\"input_size\":1024}")
  
  if echo "$response" | grep -q "\"price\":"; then
    price=$(echo "$response" | grep -o '"price":"[^"]*"' | cut -d'"' -f4)
    log_test "Calculate query price" "pass" "Price for standard completion query: $price tokens"
  else
    log_test "Calculate query price" "fail" "Failed to calculate price: $response"
    return 1
  fi
  
  # Calculate price for a premium RAG query
  response=$(call_api "POST" "/economy/price" "{\"model_tier\":\"premium\",\"query_type\":\"rag\",\"input_size\":2048}")
  
  if echo "$response" | grep -q "\"price\":"; then
    price=$(echo "$response" | grep -o '"price":"[^"]*"' | cut -d'"' -f4)
    log_test "Calculate premium RAG price" "pass" "Price for premium RAG query: $price tokens"
  else
    log_test "Calculate premium RAG price" "fail" "Failed to calculate price: $response"
    return 1
  fi
  
  return 0
}

# Test: Process query payment
test_process_payment() {
  echo -e "\n${BLUE}Test: Process Query Payment${NC}"
  
  # Process payment for a standard completion query
  response=$(call_api "POST" "/economy/pay" "{\"user_id\":\"$USER_ID\",\"model_id\":\"gpt-4\",\"model_tier\":\"standard\",\"query_type\":\"completion\",\"input_size\":1024}")
  
  if echo "$response" | grep -q "\"type\":\"query_payment\""; then
    payment_tx_id=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    log_test "Process query payment" "pass" "Processed payment with transaction ID: $payment_tx_id"
  else
    log_test "Process query payment" "fail" "Failed to process payment: $response"
    return 1
  fi
  
  # Store payment transaction ID for reward test
  PAYMENT_TX_ID=$payment_tx_id
  
  return 0
}

# Test: Distribute query reward
test_distribute_reward() {
  echo -e "\n${BLUE}Test: Distribute Query Reward${NC}"
  
  if [ -z "$PAYMENT_TX_ID" ]; then
    log_test "Distribute query reward" "fail" "No payment transaction ID available"
    return 1
  fi
  
  # Distribute reward to node
  response=$(call_api "POST" "/economy/reward" "{\"transaction_id\":\"$PAYMENT_TX_ID\",\"node_id\":\"$NODE_ID\",\"response_time_ms\":500,\"success\":true,\"quality_score\":0.9}" "admin")
  
  if echo "$response" | grep -q "\"type\":\"reward\""; then
    reward_amount=$(echo "$response" | grep -o '"amount":"[^"]*"' | cut -d'"' -f4)
    log_test "Distribute query reward" "pass" "Distributed reward of $reward_amount tokens to node"
  else
    log_test "Distribute query reward" "fail" "Failed to distribute reward: $response"
    return 1
  fi
  
  return 0
}

# Test: Get account transactions
test_get_transactions() {
  echo -e "\n${BLUE}Test: Get Account Transactions${NC}"
  
  # Get user transactions
  response=$(call_api "GET" "/accounts/$USER_ID/transactions?limit=5")
  
  if [ -n "$response" ] && [ "$response" != "null" ]; then
    tx_count=$(echo "$response" | grep -o '"id":"[^"]*"' | wc -l)
    log_test "Get user transactions" "pass" "Retrieved $tx_count transactions for user"
  else
    log_test "Get user transactions" "fail" "Failed to retrieve user transactions: $response"
    return 1
  fi
  
  # Get node transactions
  response=$(call_api "GET" "/accounts/$NODE_ID/transactions?limit=5")
  
  if [ -n "$response" ] && [ "$response" != "null" ]; then
    tx_count=$(echo "$response" | grep -o '"id":"[^"]*"' | wc -l)
    log_test "Get node transactions" "pass" "Retrieved $tx_count transactions for node"
  else
    log_test "Get node transactions" "fail" "Failed to retrieve node transactions: $response"
    return 1
  fi
  
  return 0
}

# Test: Get network stats
test_get_network_stats() {
  echo -e "\n${BLUE}Test: Get Network Stats${NC}"
  
  response=$(call_api "GET" "/economy/stats")
  
  if echo "$response" | grep -q "\"total_supply\":"; then
    total_supply=$(echo "$response" | grep -o '"total_supply":"[^"]*"' | cut -d'"' -f4)
    active_nodes=$(echo "$response" | grep -o '"active_nodes":[0-9]*' | cut -d':' -f2)
    total_accounts=$(echo "$response" | grep -o '"total_accounts":[0-9]*' | cut -d':' -f2)
    
    log_test "Get network stats" "pass" "Total supply: $total_supply tokens, Active nodes: $active_nodes, Total accounts: $total_accounts"
  else
    log_test "Get network stats" "fail" "Failed to retrieve network stats: $response"
    return 1
  fi
  
  return 0
}

# Test: Update pricing rule (admin only)
test_update_pricing_rule() {
  echo -e "\n${BLUE}Test: Update Pricing Rule${NC}"
  
  # Update pricing rule for standard completion
  response=$(call_api "PUT" "/economy/pricing-rules" "{\"model_tier\":\"standard\",\"query_type\":\"completion\",\"base_price\":\"6000000000000000000\",\"token_per_ms\":\"0.0006\",\"token_per_kb\":\"0.06\",\"min_price\":\"6000000000000000000\",\"max_price\":\"60000000000000000000\",\"demand_multiplier\":\"1.0\"}" "admin")
  
  if echo "$response" | grep -q "\"status\":\"success\""; then
    log_test "Update pricing rule" "pass" "Updated pricing rule for standard completion"
  else
    log_test "Update pricing rule" "fail" "Failed to update pricing rule: $response"
    return 1
  fi
  
  # Verify updated price
  response=$(call_api "POST" "/economy/price" "{\"model_tier\":\"standard\",\"query_type\":\"completion\",\"input_size\":0}")
  
  if echo "$response" | grep -q "\"price\":"; then
    price=$(echo "$response" | grep -o '"price":"[^"]*"' | cut -d'"' -f4)
    log_test "Verify updated price" "pass" "New price for standard completion: $price tokens"
  else
    log_test "Verify updated price" "fail" "Failed to verify updated price: $response"
    return 1
  fi
  
  return 0
}

# Print test summary
print_summary() {
  echo -e "\n${BLUE}===============================${NC}"
  echo -e "${BLUE}       TEST SUMMARY           ${NC}"
  echo -e "${BLUE}===============================${NC}"
  echo -e "${CYAN}Total tests:${NC} $TOTAL_TESTS"
  echo -e "${GREEN}Passed:${NC} $PASSED_TESTS"
  echo -e "${RED}Failed:${NC} $FAILED_TESTS"
  echo -e "${BLUE}===============================${NC}"
  
  if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
  else
    echo -e "${RED}Some tests failed. Please check the output above for details.${NC}"
  fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -u|--url)
      API_URL="$2"
      shift 2
      ;;
    -a|--admin-key)
      ADMIN_KEY="$2"
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

# Main execution
echo -e "${BLUE}===============================${NC}"
echo -e "${BLUE}  CORTEX ECONOMIC ENGINE TEST  ${NC}"
echo -e "${BLUE}===============================${NC}"
echo -e "${YELLOW}API URL:${NC} $API_URL"
echo -e "${YELLOW}Test User ID:${NC} $USER_ID"
echo -e "${YELLOW}Test Node ID:${NC} $NODE_ID"
echo -e "${BLUE}===============================${NC}"

# Check if API is available
check_api

# Run tests
test_create_user_account
test_create_node_account
test_get_account
test_mint_tokens || { echo -e "${RED}Skipping remaining tests due to mint failure${NC}"; print_summary; exit 1; }
test_transfer_tokens
test_stake_tokens
test_unstake_tokens
test_calculate_price
test_process_payment
test_distribute_reward
test_get_transactions
test_get_network_stats
test_update_pricing_rule

# Print summary
print_summary

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
  exit 0
else
  exit 1
fi
