package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/loreum-org/cortex/pkg/types"
)

// ServiceRegistryManager manages the decentralized service registry
type ServiceRegistryManager struct {
	registry         *types.ServiceRegistry
	nodeID           string
	consensusService ConsensusService
	p2pService       P2PService
	mutex            sync.RWMutex

	// Local services offered by this node
	localServices map[string]*types.ServiceOffering

	// Attestation storage
	attestations map[string]*types.ServiceAttestation

	// Query routing
	router *ServiceRouter

	// Callbacks
	onServiceUpdate func(service *types.ServiceOffering)
	onServiceRemove func(serviceID string)
}

// ConsensusService interface for interacting with consensus
type ConsensusService interface {
	SubmitTransaction(ctx context.Context, tx interface{}) error
	IsTransactionFinalized(txID string) bool
}

// P2PService interface for peer-to-peer communication
type P2PService interface {
	Broadcast(topic string, data []byte) error
	Subscribe(topic string, handler func([]byte)) error
	GetConnectedPeers() []string
}

// NewServiceRegistryManager creates a new service registry manager
func NewServiceRegistryManager(nodeID string, consensus ConsensusService, p2p P2PService) *ServiceRegistryManager {
	registry := &types.ServiceRegistry{
		Services:  make(map[string]*types.ServiceOffering),
		NodeIndex: make(map[string][]string),
		TypeIndex: make(map[types.ServiceType][]string),
		UpdatedAt: time.Now(),
	}

	manager := &ServiceRegistryManager{
		registry:         registry,
		nodeID:           nodeID,
		consensusService: consensus,
		p2pService:       p2p,
		localServices:    make(map[string]*types.ServiceOffering),
		attestations:     make(map[string]*types.ServiceAttestation),
		router:           NewServiceRouter(registry),
	}

	// Subscribe to service registry updates
	p2p.Subscribe("service_registry", manager.handleServiceRegistryMessage)

	return manager
}

// RegisterService registers a new service offered by this node
func (srm *ServiceRegistryManager) RegisterService(ctx context.Context, service *types.ServiceOffering) error {
	srm.mutex.Lock()
	defer srm.mutex.Unlock()

	// Set node ID and generate service ID if not provided
	service.NodeID = srm.nodeID
	if service.ID == "" {
		service.ID = generateServiceID(srm.nodeID, service.Type, service.Name)
	}

	// Set timestamps
	now := time.Now()
	service.CreatedAt = now
	service.UpdatedAt = now
	service.LastSeen = now
	service.Status = types.ServiceStatusActive

	// Create registration transaction
	tx := &types.ServiceRegistryTransaction{
		TransactionBase: types.TransactionBase{
			ID:        uuid.New().String(),
			Type:      "service_registry",
			Timestamp: now.Unix(),
			CreatedAt: now,
		},
		Action:    types.ServiceActionRegister,
		ServiceID: service.ID,
		NodeID:    srm.nodeID,
		Service:   service,
	}

	// Sign transaction
	if err := srm.signTransaction(tx); err != nil {
		return fmt.Errorf("failed to sign registration transaction: %w", err)
	}

	// Submit to consensus
	if err := srm.consensusService.SubmitTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to submit registration transaction: %w", err)
	}

	// Store locally
	srm.localServices[service.ID] = service
	srm.addServiceToRegistry(service)

	// Broadcast to network
	srm.broadcastServiceUpdate(service)

	log.Printf("Service registered: %s (%s) on node %s", service.Name, service.Type, srm.nodeID)
	return nil
}

// DeregisterService removes a service from the registry
func (srm *ServiceRegistryManager) DeregisterService(ctx context.Context, serviceID string) error {
	srm.mutex.Lock()
	defer srm.mutex.Unlock()

	// Check if service exists and belongs to this node
	service, exists := srm.localServices[serviceID]
	if !exists {
		return fmt.Errorf("service %s not found on this node", serviceID)
	}

	// Create deregistration transaction
	tx := &types.ServiceRegistryTransaction{
		TransactionBase: types.TransactionBase{
			ID:        uuid.New().String(),
			Type:      "service_registry",
			Timestamp: time.Now().Unix(),
			CreatedAt: time.Now(),
		},
		Action:    types.ServiceActionDeregister,
		ServiceID: serviceID,
		NodeID:    srm.nodeID,
	}

	// Sign transaction
	if err := srm.signTransaction(tx); err != nil {
		return fmt.Errorf("failed to sign deregistration transaction: %w", err)
	}

	// Submit to consensus
	if err := srm.consensusService.SubmitTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to submit deregistration transaction: %w", err)
	}

	// Remove locally
	delete(srm.localServices, serviceID)
	srm.removeServiceFromRegistry(serviceID)

	// Broadcast removal
	srm.broadcastServiceRemoval(serviceID)

	log.Printf("Service deregistered: %s from node %s", service.Name, srm.nodeID)
	return nil
}

// FindServices finds services matching the given requirements
func (srm *ServiceRegistryManager) FindServices(requirements []types.ServiceRequirement) []*types.ServiceOffering {
	srm.mutex.RLock()
	defer srm.mutex.RUnlock()

	var matchingServices []*types.ServiceOffering

	for _, service := range srm.registry.Services {
		if srm.serviceMatchesRequirements(service, requirements) {
			matchingServices = append(matchingServices, service)
		}
	}

	return matchingServices
}

// RouteQuery routes a query to the best available node
func (srm *ServiceRegistryManager) RouteQuery(query *types.ServiceQuery) (string, error) {
	return srm.router.RouteQuery(query)
}

// CreateAttestation creates an attestation for a completed service execution
func (srm *ServiceRegistryManager) CreateAttestation(
	queryID, serviceID, userID string,
	startTime, endTime time.Time,
	inputData, outputData []byte,
	success bool,
	errorMsg string,
	tokensUsed int64,
	cost string,
) (*types.ServiceAttestation, error) {

	attestation := &types.ServiceAttestation{
		ID:          uuid.New().String(),
		QueryID:     queryID,
		NodeID:      srm.nodeID,
		ServiceID:   serviceID,
		UserID:      userID,
		StartTime:   startTime,
		EndTime:     endTime,
		InputHash:   hashData(inputData),
		OutputHash:  hashData(outputData),
		Success:     success,
		ErrorMsg:    errorMsg,
		TokensUsed:  tokensUsed,
		ComputeTime: endTime.Sub(startTime).Milliseconds(),
		Cost:        cost,
		CreatedAt:   time.Now(),
	}

	// Sign attestation
	if err := srm.signAttestation(attestation); err != nil {
		return nil, fmt.Errorf("failed to sign attestation: %w", err)
	}

	// Store attestation
	srm.mutex.Lock()
	srm.attestations[attestation.ID] = attestation
	srm.mutex.Unlock()

	return attestation, nil
}

// GetLocalServices returns services offered by this node
func (srm *ServiceRegistryManager) GetLocalServices() map[string]*types.ServiceOffering {
	srm.mutex.RLock()
	defer srm.mutex.RUnlock()

	services := make(map[string]*types.ServiceOffering)
	for id, service := range srm.localServices {
		services[id] = service
	}
	return services
}

// GetNetworkServices returns all services in the network
func (srm *ServiceRegistryManager) GetNetworkServices() map[string]*types.ServiceOffering {
	srm.mutex.RLock()
	defer srm.mutex.RUnlock()

	services := make(map[string]*types.ServiceOffering)
	for id, service := range srm.registry.Services {
		services[id] = service
	}
	return services
}

// GetServicesByType returns services of a specific type
func (srm *ServiceRegistryManager) GetServicesByType(serviceType types.ServiceType) []*types.ServiceOffering {
	srm.mutex.RLock()
	defer srm.mutex.RUnlock()

	var services []*types.ServiceOffering
	for _, serviceID := range srm.registry.TypeIndex[serviceType] {
		if service, exists := srm.registry.Services[serviceID]; exists {
			services = append(services, service)
		}
	}
	return services
}

// GetServicesByNode returns services offered by a specific node
func (srm *ServiceRegistryManager) GetServicesByNode(nodeID string) []*types.ServiceOffering {
	srm.mutex.RLock()
	defer srm.mutex.RUnlock()

	var services []*types.ServiceOffering
	for _, serviceID := range srm.registry.NodeIndex[nodeID] {
		if service, exists := srm.registry.Services[serviceID]; exists {
			services = append(services, service)
		}
	}
	return services
}

// SendHeartbeat updates the last seen time for all local services
func (srm *ServiceRegistryManager) SendHeartbeat() {
	srm.mutex.Lock()
	defer srm.mutex.Unlock()

	now := time.Now()
	for _, service := range srm.localServices {
		service.LastSeen = now
		service.UpdatedAt = now

		// Update in registry
		if registryService, exists := srm.registry.Services[service.ID]; exists {
			registryService.LastSeen = now
			registryService.UpdatedAt = now
		}
	}

	// Broadcast heartbeat
	srm.broadcastHeartbeat()
}

// Helper methods

func (srm *ServiceRegistryManager) addServiceToRegistry(service *types.ServiceOffering) {
	srm.registry.Services[service.ID] = service

	// Update node index
	srm.registry.NodeIndex[service.NodeID] = append(srm.registry.NodeIndex[service.NodeID], service.ID)

	// Update type index
	srm.registry.TypeIndex[service.Type] = append(srm.registry.TypeIndex[service.Type], service.ID)

	srm.registry.UpdatedAt = time.Now()

	if srm.onServiceUpdate != nil {
		srm.onServiceUpdate(service)
	}
}

func (srm *ServiceRegistryManager) removeServiceFromRegistry(serviceID string) {
	service, exists := srm.registry.Services[serviceID]
	if !exists {
		return
	}

	// Remove from main registry
	delete(srm.registry.Services, serviceID)

	// Remove from node index
	nodeServices := srm.registry.NodeIndex[service.NodeID]
	for i, id := range nodeServices {
		if id == serviceID {
			srm.registry.NodeIndex[service.NodeID] = append(nodeServices[:i], nodeServices[i+1:]...)
			break
		}
	}

	// Remove from type index
	typeServices := srm.registry.TypeIndex[service.Type]
	for i, id := range typeServices {
		if id == serviceID {
			srm.registry.TypeIndex[service.Type] = append(typeServices[:i], typeServices[i+1:]...)
			break
		}
	}

	srm.registry.UpdatedAt = time.Now()

	if srm.onServiceRemove != nil {
		srm.onServiceRemove(serviceID)
	}
}

func (srm *ServiceRegistryManager) serviceMatchesRequirements(service *types.ServiceOffering, requirements []types.ServiceRequirement) bool {
	for _, req := range requirements {
		if service.Type != req.Type {
			continue
		}

		// Check capabilities
		if len(req.Capabilities) > 0 {
			hasRequiredCapabilities := true
			for _, reqCap := range req.Capabilities {
				found := false
				for _, serviceCap := range service.Capabilities {
					if serviceCap.Name == reqCap {
						found = true
						break
					}
				}
				if !found {
					hasRequiredCapabilities = false
					break
				}
			}
			if !hasRequiredCapabilities {
				continue
			}
		}

		return true
	}
	return false
}

func (srm *ServiceRegistryManager) handleServiceRegistryMessage(data []byte) {
	// Handle incoming service registry updates from other nodes
	// This would parse and apply updates from the network
	log.Printf("Received service registry update: %d bytes", len(data))
}

func (srm *ServiceRegistryManager) broadcastServiceUpdate(service *types.ServiceOffering) {
	data, err := json.Marshal(service)
	if err != nil {
		log.Printf("Failed to marshal service update: %v", err)
		return
	}

	if err := srm.p2pService.Broadcast("service_registry", data); err != nil {
		log.Printf("Failed to broadcast service update: %v", err)
	}
}

func (srm *ServiceRegistryManager) broadcastServiceRemoval(serviceID string) {
	message := map[string]interface{}{
		"action":     "remove",
		"service_id": serviceID,
		"node_id":    srm.nodeID,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal service removal: %v", err)
		return
	}

	if err := srm.p2pService.Broadcast("service_registry", data); err != nil {
		log.Printf("Failed to broadcast service removal: %v", err)
	}
}

func (srm *ServiceRegistryManager) broadcastHeartbeat() {
	message := map[string]interface{}{
		"action":  "heartbeat",
		"node_id": srm.nodeID,
		"time":    time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal heartbeat: %v", err)
		return
	}

	if err := srm.p2pService.Broadcast("service_heartbeat", data); err != nil {
		log.Printf("Failed to broadcast heartbeat: %v", err)
	}
}

func (srm *ServiceRegistryManager) signTransaction(tx *types.ServiceRegistryTransaction) error {
	// Create hash of transaction data
	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)
	tx.Hash = hex.EncodeToString(hash[:])

	// In a real implementation, this would use the node's private key
	tx.Signature = "placeholder_signature_" + tx.Hash[:16]

	return nil
}

func (srm *ServiceRegistryManager) signAttestation(attestation *types.ServiceAttestation) error {
	// Create signature for attestation
	data, err := json.Marshal(attestation)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)
	// In a real implementation, this would use the node's private key
	attestation.Signature = "placeholder_signature_" + hex.EncodeToString(hash[:])[:16]

	return nil
}

func generateServiceID(nodeID string, serviceType types.ServiceType, serviceName string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", nodeID, serviceType, serviceName, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

func hashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
