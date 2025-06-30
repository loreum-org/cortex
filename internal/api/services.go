package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/loreum-org/cortex/internal/services"
	"github.com/loreum-org/cortex/pkg/types"
)

// ServiceAPI handles HTTP requests for service management
type ServiceAPI struct {
	registry *services.ServiceRegistryManager
}

// NewServiceAPI creates a new service API handler
func NewServiceAPI(registry *services.ServiceRegistryManager) *ServiceAPI {
	return &ServiceAPI{
		registry: registry,
	}
}

// RegisterRoutes registers service management routes
func (sa *ServiceAPI) RegisterRoutes(router *mux.Router) {
	// Service registration and management
	router.HandleFunc("/services", sa.handleListServices).Methods("GET")
	router.HandleFunc("/services", sa.handleRegisterService).Methods("POST")
	router.HandleFunc("/services/{serviceId}", sa.handleGetService).Methods("GET")
	router.HandleFunc("/services/{serviceId}", sa.handleUpdateService).Methods("PUT")
	router.HandleFunc("/services/{serviceId}", sa.handleDeregisterService).Methods("DELETE")
	
	// Service discovery
	router.HandleFunc("/services/discover", sa.handleDiscoverServices).Methods("POST")
	router.HandleFunc("/services/types/{serviceType}", sa.handleGetServicesByType).Methods("GET")
	router.HandleFunc("/services/nodes/{nodeId}", sa.handleGetServicesByNode).Methods("GET")
	
	// Query routing
	router.HandleFunc("/queries/route", sa.handleRouteQuery).Methods("POST")
	router.HandleFunc("/queries/{queryId}/attest", sa.handleCreateAttestation).Methods("POST")
	
	// Local node services
	router.HandleFunc("/node/services", sa.handleGetLocalServices).Methods("GET")
	router.HandleFunc("/node/services/heartbeat", sa.handleSendHeartbeat).Methods("POST")
	
	// Network overview
	router.HandleFunc("/network/services", sa.handleGetNetworkServices).Methods("GET")
	router.HandleFunc("/network/services/stats", sa.handleGetServiceStats).Methods("GET")
}

// handleListServices returns all services in the network
func (sa *ServiceAPI) handleListServices(w http.ResponseWriter, r *http.Request) {
	services := sa.registry.GetNetworkServices()
	
	response := map[string]interface{}{
		"services":   services,
		"count":      len(services),
		"timestamp":  time.Now(),
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

// handleRegisterService registers a new service
func (sa *ServiceAPI) handleRegisterService(w http.ResponseWriter, r *http.Request) {
	var req RegisterServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := sa.validateRegisterServiceRequest(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid service data", err)
		return
	}

	// Create service offering
	service := &types.ServiceOffering{
		Type:         req.Type,
		Name:         req.Name,
		Description:  req.Description,
		Capabilities: req.Capabilities,
		Pricing:      req.Pricing,
		Metadata:     req.Metadata,
		Status:       types.ServiceStatusActive,
	}

	// Register service
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := sa.registry.RegisterService(ctx, service); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to register service", err)
		return
	}

	response := map[string]interface{}{
		"service_id": service.ID,
		"message":    "Service registered successfully",
		"service":    service,
	}

	writeJSONResponse(w, http.StatusCreated, response)
}

// handleGetService returns details for a specific service
func (sa *ServiceAPI) handleGetService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["serviceId"]

	services := sa.registry.GetNetworkServices()
	service, exists := services[serviceID]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "Service not found", fmt.Errorf("service %s not found", serviceID))
		return
	}

	writeJSONResponse(w, http.StatusOK, service)
}

// handleUpdateService updates a service (only for services owned by this node)
func (sa *ServiceAPI) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["serviceId"]

	var req UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Check if service belongs to this node
	localServices := sa.registry.GetLocalServices()
	service, exists := localServices[serviceID]
	if !exists {
		writeErrorResponse(w, http.StatusForbidden, "Service not owned by this node", fmt.Errorf("service %s not owned by this node", serviceID))
		return
	}

	// Update service fields
	if req.Description != nil {
		service.Description = *req.Description
	}
	if req.Status != nil {
		service.Status = *req.Status
	}
	if req.Pricing != nil {
		service.Pricing = req.Pricing
	}
	if req.Metadata != nil {
		service.Metadata = req.Metadata
	}

	service.UpdatedAt = time.Now()

	writeJSONResponse(w, http.StatusOK, service)
}

// handleDeregisterService removes a service
func (sa *ServiceAPI) handleDeregisterService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["serviceId"]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := sa.registry.DeregisterService(ctx, serviceID); err != nil {
		if err.Error() == fmt.Sprintf("service %s not found on this node", serviceID) {
			writeErrorResponse(w, http.StatusForbidden, "Service not owned by this node", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to deregister service", err)
		}
		return
	}

	response := map[string]interface{}{
		"message":    "Service deregistered successfully",
		"service_id": serviceID,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleDiscoverServices discovers services matching requirements
func (sa *ServiceAPI) handleDiscoverServices(w http.ResponseWriter, r *http.Request) {
	var req DiscoverServicesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	matchingServices := sa.registry.FindServices(req.Requirements)

	response := map[string]interface{}{
		"services":  matchingServices,
		"count":     len(matchingServices),
		"query":     req,
		"timestamp": time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleGetServicesByType returns services of a specific type
func (sa *ServiceAPI) handleGetServicesByType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceType := types.ServiceType(vars["serviceType"])

	services := sa.registry.GetServicesByType(serviceType)

	response := map[string]interface{}{
		"services":     services,
		"service_type": serviceType,
		"count":        len(services),
		"timestamp":    time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleGetServicesByNode returns services offered by a specific node
func (sa *ServiceAPI) handleGetServicesByNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	services := sa.registry.GetServicesByNode(nodeID)

	response := map[string]interface{}{
		"services":  services,
		"node_id":   nodeID,
		"count":     len(services),
		"timestamp": time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleRouteQuery routes a query to the best available node
func (sa *ServiceAPI) handleRouteQuery(w http.ResponseWriter, r *http.Request) {
	var req RouteQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Create service query
	query := &types.ServiceQuery{
		ID:               generateQueryID(),
		UserID:           req.UserID,
		RequiredServices: req.RequiredServices,
		Content:          req.Content,
		Metadata:         req.Metadata,
		MaxPrice:         req.MaxPrice,
		Timeout:          time.Duration(req.TimeoutSeconds) * time.Second,
		CreatedAt:        time.Now(),
	}

	// Route query
	selectedNode, err := sa.registry.RouteQuery(query)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "No suitable nodes available", err)
		return
	}

	response := map[string]interface{}{
		"query_id":        query.ID,
		"selected_node":   selectedNode,
		"estimated_cost":  "1000000000000000000", // Placeholder
		"routing_time":    time.Since(query.CreatedAt),
		"message":         "Query routed successfully",
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleCreateAttestation creates an attestation for a completed service execution
func (sa *ServiceAPI) handleCreateAttestation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queryID := vars["queryId"]

	var req CreateAttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	attestation, err := sa.registry.CreateAttestation(
		queryID,
		req.ServiceID,
		req.UserID,
		time.Unix(req.StartTime, 0),
		time.Unix(req.EndTime, 0),
		[]byte(req.InputData),
		[]byte(req.OutputData),
		req.Success,
		req.ErrorMessage,
		req.TokensUsed,
		req.Cost,
	)

	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create attestation", err)
		return
	}

	response := map[string]interface{}{
		"attestation_id": attestation.ID,
		"message":        "Attestation created successfully",
		"attestation":    attestation,
	}

	writeJSONResponse(w, http.StatusCreated, response)
}

// handleGetLocalServices returns services offered by this node
func (sa *ServiceAPI) handleGetLocalServices(w http.ResponseWriter, r *http.Request) {
	services := sa.registry.GetLocalServices()

	response := map[string]interface{}{
		"services":  services,
		"count":     len(services),
		"timestamp": time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleSendHeartbeat sends heartbeat for all local services
func (sa *ServiceAPI) handleSendHeartbeat(w http.ResponseWriter, r *http.Request) {
	sa.registry.SendHeartbeat()

	response := map[string]interface{}{
		"message":   "Heartbeat sent successfully",
		"timestamp": time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleGetNetworkServices returns overview of all network services
func (sa *ServiceAPI) handleGetNetworkServices(w http.ResponseWriter, r *http.Request) {
	services := sa.registry.GetNetworkServices()

	// Group by type
	byType := make(map[types.ServiceType][]*types.ServiceOffering)
	for _, service := range services {
		byType[service.Type] = append(byType[service.Type], service)
	}

	response := map[string]interface{}{
		"services":   services,
		"by_type":    byType,
		"total":      len(services),
		"timestamp":  time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// handleGetServiceStats returns statistics about services in the network
func (sa *ServiceAPI) handleGetServiceStats(w http.ResponseWriter, r *http.Request) {
	services := sa.registry.GetNetworkServices()

	stats := map[string]interface{}{
		"total_services": len(services),
		"by_type":        make(map[types.ServiceType]int),
		"by_status":      make(map[types.ServiceStatus]int),
		"active_nodes":   make(map[string]bool),
		"timestamp":      time.Now(),
	}

	for _, service := range services {
		// Count by type
		stats["by_type"].(map[types.ServiceType]int)[service.Type]++
		
		// Count by status
		stats["by_status"].(map[types.ServiceStatus]int)[service.Status]++
		
		// Track unique nodes
		if service.Status == types.ServiceStatusActive {
			stats["active_nodes"].(map[string]bool)[service.NodeID] = true
		}
	}

	// Convert active nodes to count
	stats["active_node_count"] = len(stats["active_nodes"].(map[string]bool))
	delete(stats, "active_nodes")

	writeJSONResponse(w, http.StatusOK, stats)
}

// Request/Response types

type RegisterServiceRequest struct {
	Type         types.ServiceType          `json:"type"`
	Name         string                     `json:"name"`
	Description  string                     `json:"description"`
	Capabilities []types.ServiceCapability  `json:"capabilities"`
	Pricing      *types.ServicePricing      `json:"pricing"`
	Metadata     map[string]interface{}     `json:"metadata"`
}

type UpdateServiceRequest struct {
	Description *string                    `json:"description,omitempty"`
	Status      *types.ServiceStatus       `json:"status,omitempty"`
	Pricing     *types.ServicePricing      `json:"pricing,omitempty"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`
}

type DiscoverServicesRequest struct {
	Requirements []types.ServiceRequirement `json:"requirements"`
	MaxResults   int                        `json:"max_results,omitempty"`
}

type RouteQueryRequest struct {
	UserID           string                     `json:"user_id"`
	RequiredServices []types.ServiceRequirement `json:"required_services"`
	Content          string                     `json:"content"`
	Metadata         map[string]interface{}     `json:"metadata,omitempty"`
	MaxPrice         string                     `json:"max_price,omitempty"`
	TimeoutSeconds   int                        `json:"timeout_seconds,omitempty"`
}

type CreateAttestationRequest struct {
	ServiceID    string `json:"service_id"`
	UserID       string `json:"user_id"`
	StartTime    int64  `json:"start_time"`
	EndTime      int64  `json:"end_time"`
	InputData    string `json:"input_data"`
	OutputData   string `json:"output_data"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
	TokensUsed   int64  `json:"tokens_used,omitempty"`
	Cost         string `json:"cost"`
}

// Helper functions

func (sa *ServiceAPI) validateRegisterServiceRequest(req *RegisterServiceRequest) error {
	if req.Type == "" {
		return fmt.Errorf("service type is required")
	}
	if req.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if req.Type != types.ServiceTypeAgent && req.Type != types.ServiceTypeModel && 
	   req.Type != types.ServiceTypeSensor && req.Type != types.ServiceTypeTool {
		return fmt.Errorf("invalid service type: %s", req.Type)
	}
	return nil
}

func generateQueryID() string {
	return fmt.Sprintf("query_%d", time.Now().UnixNano())
}

