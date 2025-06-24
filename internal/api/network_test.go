package api

// Temporarily disabled due to missing mock dependencies
// TODO: Re-enable after implementing proper mocks

/*

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/loreum-org/cortex/pkg/types"
)

// TestSubmitQueryHandler_NetworkBroadcast focuses on the network broadcast aspect of the submitQueryHandler.
func TestSubmitQueryHandler_NetworkBroadcast(t *testing.T) {
	// Common setup for both subtests
	setup := func(t *testing.T, broadcastShouldFail bool) (*httptest.Server, *Server, *MockP2PNode, *MockSolverAgent, *atomic.Bool) {
		// This now calls setupAPITestServer from server_test.go (or wherever it's defined in the package)
		testServer, apiSrv, mockP2P, mockSolver, _, _, _ := setupAPITestServer(t, false) // Economic engine disabled for simplicity here

		var broadcastCalled atomic.Bool
		broadcastCalled.Store(false)

		// The mockP2P instance is from mocks_test.go, returned by the shared setupAPITestServer
		mockP2P.BroadcastQueryFunc = func(query *types.Query) error {
			broadcastCalled.Store(true)
			if broadcastShouldFail {
				return errors.New("simulated broadcast failure")
			}
			return nil
		}

		// The mockSolver instance is from mocks_test.go
		mockSolver.ProcessFunc = func(ctx context.Context, query *types.Query) (*types.Response, error) {
			// Basic successful processing by solver agent
			return &types.Response{
				QueryID: query.ID,
				Text:    "Solver processed query: " + query.Text,
				Status:  "success",
			}, nil
		}
		return testServer, apiSrv, mockP2P, mockSolver, &broadcastCalled
	}

	t.Run("BroadcastSuccess", func(t *testing.T) {
		testServer, apiSrv, _, _, broadcastCalled := setup(t, false)
		defer testServer.Close()

		queryReq := QueryRequest{Text: "Test query for successful broadcast", Type: "broadcast_test"}
		reqBody, _ := json.Marshal(queryReq)

		// Clear previous events for a clean check
		apiSrv.Metrics.mu.Lock()
		initialEventCount := len(apiSrv.Metrics.Events)
		apiSrv.Metrics.mu.Unlock()

		resp, err := http.Post(testServer.URL+"/queries", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		if !broadcastCalled.Load() {
			t.Error("P2PNode.BroadcastQuery was not called")
		}

		// Check if "query_broadcast" event was added
		apiSrv.Metrics.mu.RLock()
		foundBroadcastEvent := false
		if len(apiSrv.Metrics.Events) > initialEventCount { // Check if any new event was added
			// The new event should be at the beginning of the slice
			latestEvent := apiSrv.Metrics.Events[0]
			if latestEvent.Type == "query_broadcast" {
				foundBroadcastEvent = true
				if qid, ok := latestEvent.Data["query_id"].(string); !ok || qid == "" {
					t.Errorf("query_broadcast event missing or has empty query_id: %+v", latestEvent.Data)
				}
			}
		}
		apiSrv.Metrics.mu.RUnlock()

		if !foundBroadcastEvent {
			t.Errorf("Expected 'query_broadcast' event to be added on successful broadcast, but it was not. Events: %+v", apiSrv.Metrics.Events)
		}

		// Verify response from solver agent
		var queryResp types.Response
		if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if queryResp.Status != "success" {
			t.Errorf("Expected query status 'success', got '%s'", queryResp.Status)
		}
		if !bytes.Contains([]byte(queryResp.Text), []byte("Test query for successful broadcast")) {
			t.Errorf("Response text does not contain original query text. Got: %s", queryResp.Text)
		}
	})

	t.Run("BroadcastFailure", func(t *testing.T) {
		testServer, apiSrv, _, _, broadcastCalled := setup(t, true)
		defer testServer.Close()

		queryReq := QueryRequest{Text: "Test query for failed broadcast", Type: "broadcast_fail_test"}
		reqBody, _ := json.Marshal(queryReq)

		// Clear previous events for a clean check
		apiSrv.Metrics.mu.Lock()
		initialEventCount := len(apiSrv.Metrics.Events)
		apiSrv.Metrics.mu.Unlock()

		resp, err := http.Post(testServer.URL+"/queries", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// HTTP request should still succeed as local processing continues
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d even on broadcast failure, got %d", http.StatusOK, resp.StatusCode)
		}

		if !broadcastCalled.Load() {
			t.Error("P2PNode.BroadcastQuery was not called")
		}

		// Check that "query_broadcast" event was NOT added
		apiSrv.Metrics.mu.RLock()
		foundBroadcastEvent := false
		// Check if any new event was added and if it was query_broadcast
		if len(apiSrv.Metrics.Events) > initialEventCount {
			latestEvent := apiSrv.Metrics.Events[0] // Most recent event
			if latestEvent.Type == "query_broadcast" {
				foundBroadcastEvent = true
			}
		}
		apiSrv.Metrics.mu.RUnlock()

		if foundBroadcastEvent {
			t.Errorf("Expected 'query_broadcast' event NOT to be added on broadcast failure. Events: %+v", apiSrv.Metrics.Events)
		}

		// Verify response from solver agent (local processing should still occur)
		var queryResp types.Response
		if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if queryResp.Status != "success" {
			t.Errorf("Expected query status 'success' from local processing, got '%s'", queryResp.Status)
		}
		if !bytes.Contains([]byte(queryResp.Text), []byte("Test query for failed broadcast")) {
			t.Errorf("Response text does not contain original query text. Got: %s", queryResp.Text)
		}
	})
}
*/
