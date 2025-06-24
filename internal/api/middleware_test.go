package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/loreum-org/cortex/internal/economy"
)

// TestCORSMiddleware tests the CORS middleware functionality.
func TestCORSMiddleware(t *testing.T) {
	// Create a minimal server instance for testing middleware
	// We don't need all dependencies for this specific test.
	s := &Server{
		Metrics: &ServerMetrics{ // Metrics might be accessed if AddEvent is called, provide a minimal one.
			StartTime: time.Now(),
			mu:        sync.RWMutex{},
			Events:    make([]Event, 0, 100),
		},
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Default OK response from the actual handler
		w.Write([]byte("OK"))
	})

	corsWrappedHandler := s.corsMiddleware(baseHandler)

	t.Run("OPTIONS_PreflightRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		rr := httptest.NewRecorder()

		// Variable to track if the base handler was called
		baseHandlerCalled := false
		optionsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			baseHandlerCalled = true // This should not be called for OPTIONS
			w.WriteHeader(http.StatusOK)
		})
		s.corsMiddleware(optionsHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeaders := map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type, Authorization",
		}

		for key, expectedValue := range expectedHeaders {
			if value := rr.Header().Get(key); value != expectedValue {
				t.Errorf("handler returned unexpected header %s: got %v want %v", key, value, expectedValue)
			}
		}

		if baseHandlerCalled {
			t.Error("base handler should not have been called for OPTIONS request")
		}
	})

	t.Run("GET_Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		baseHandlerCalled := false
		actualHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			baseHandlerCalled = true
			w.WriteHeader(http.StatusAccepted) // Different status to confirm it was called
		})
		s.corsMiddleware(actualHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
		}

		expectedHeaders := map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type, Authorization",
		}

		for key, expectedValue := range expectedHeaders {
			if value := rr.Header().Get(key); value != expectedValue {
				t.Errorf("handler returned unexpected header %s: got %v want %v", key, value, expectedValue)
			}
		}

		if !baseHandlerCalled {
			t.Error("base handler should have been called for GET request")
		}
	})

	t.Run("POST_Request_No_Options_Method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil) // Not OPTIONS
		rr := httptest.NewRecorder()

		corsWrappedHandler.ServeHTTP(rr, req) // Use the handler that calls baseHandler

		if status := rr.Code; status != http.StatusOK { // Assuming baseHandler returns OK
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeaders := map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type, Authorization",
		}
		for key, expectedValue := range expectedHeaders {
			if value := rr.Header().Get(key); value != expectedValue {
				t.Errorf("handler returned unexpected header %s: got %v want %v", key, value, expectedValue)
			}
		}
		if rr.Body.String() != "OK" {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), "OK")
		}
	})
}

// TestAdminAuthMiddleware tests the admin authentication middleware.
func TestAdminAuthMiddleware(t *testing.T) {
	// Store original AdminAPIKey and restore it after tests
	originalAdminAPIKey := AdminAPIKey
	defer func() { AdminAPIKey = originalAdminAPIKey }()

	// Mock EconomyService, only adminAuthMiddleware is relevant here
	mockEngine := &economy.EconomicEngine{} // Engine can be minimal
	economyService := NewEconomyService(mockEngine)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Admin access granted"))
	})

	adminProtectedHandler := economyService.adminAuthMiddleware(dummyHandler)

	t.Run("SuccessfulAuthentication", func(t *testing.T) {
		AdminAPIKey = "test-secret-key"
		if AdminAPIKey == "" {
			t.Fatal("Test setup error: AdminAPIKey is empty after setting")
		}

		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-Admin-Key", "test-secret-key")
		rr := httptest.NewRecorder()

		adminProtectedHandler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if body := rr.Body.String(); body != "Admin access granted" {
			t.Errorf("handler returned unexpected body: got %v want %v", body, "Admin access granted")
		}
	})

	t.Run("MissingAdminKeyHeader", func(t *testing.T) {
		AdminAPIKey = "test-secret-key"

		req := httptest.NewRequest("GET", "/admin/test", nil)
		// No X-Admin-Key header
		rr := httptest.NewRecorder()

		adminProtectedHandler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		// Check for specific error message if desired
		// Example: if !strings.Contains(rr.Body.String(), "Unauthorized: Invalid Admin Key") { ... }
	})

	t.Run("IncorrectAdminKeyHeader", func(t *testing.T) {
		AdminAPIKey = "test-secret-key"

		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-Admin-Key", "wrong-key")
		rr := httptest.NewRecorder()

		adminProtectedHandler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("AdminAPIKeyNotConfigured", func(t *testing.T) {
		AdminAPIKey = "" // Simulate not configured

		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-Admin-Key", "any-key-doesnt-matter") // Header present but server key not set
		rr := httptest.NewRecorder()

		adminProtectedHandler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
		// Check for specific error message if desired
		// Example: if !strings.Contains(rr.Body.String(), "Admin API key not configured") { ... }
	})

	t.Run("AdminAPIKeyConfigured_EmptyHeaderValue", func(t *testing.T) {
		AdminAPIKey = "test-secret-key"

		req := httptest.NewRequest("GET", "/admin/test", nil)
		req.Header.Set("X-Admin-Key", "") // Empty header value
		rr := httptest.NewRecorder()

		adminProtectedHandler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}
