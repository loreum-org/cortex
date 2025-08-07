package types

import (
	"context"
	"fmt"
	"time"

	"github.com/loreum-org/cortex/internal/events"
)

// NewDomainCoordinator creates a new domain coordinator
func NewDomainCoordinator(eventBus *events.EventBus) *DomainCoordinator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DomainCoordinator{
		domains:      make(map[DomainType]Domain),
		eventBus:     eventBus,
		ctx:          ctx,
		cancel:       cancel,
		messageQueue: make(chan *InterDomainMessage, 1000),
	}
}

// RegisterDomain registers a new consciousness domain
func (dc *DomainCoordinator) RegisterDomain(domain Domain) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	domainType := domain.GetType()
	if _, exists := dc.domains[domainType]; exists {
		return fmt.Errorf("domain %s already registered", domainType)
	}

	dc.domains[domainType] = domain
	
	// Start the domain if coordinator is running
	if dc.isRunning {
		if err := domain.Start(dc.ctx); err != nil {
			delete(dc.domains, domainType)
			return fmt.Errorf("failed to start domain %s: %w", domainType, err)
		}
	}

	return nil
}

// GetDomain retrieves a registered domain
func (dc *DomainCoordinator) GetDomain(domainType DomainType) (Domain, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	domain, exists := dc.domains[domainType]
	if !exists {
		return nil, fmt.Errorf("domain %s not found", domainType)
	}

	return domain, nil
}

// SendMessage sends a message between domains
func (dc *DomainCoordinator) SendMessage(message *InterDomainMessage) error {
	select {
	case dc.messageQueue <- message:
		return nil
	case <-dc.ctx.Done():
		return fmt.Errorf("coordinator is shutting down")
	default:
		return fmt.Errorf("message queue is full")
	}
}

// Start starts the domain coordinator and all registered domains
func (dc *DomainCoordinator) Start() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.isRunning {
		return fmt.Errorf("coordinator already running")
	}

	// Start all domains
	for domainType, domain := range dc.domains {
		if err := domain.Start(dc.ctx); err != nil {
			return fmt.Errorf("failed to start domain %s: %w", domainType, err)
		}
	}

	// Start message processing
	go dc.processMessages()

	dc.isRunning = true
	return nil
}

// Stop stops the domain coordinator and all domains
func (dc *DomainCoordinator) Stop() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if !dc.isRunning {
		return nil
	}

	// Cancel context to stop message processing
	dc.cancel()

	// Stop all domains
	for domainType, domain := range dc.domains {
		if err := domain.Stop(dc.ctx); err != nil {
			// Log error but continue stopping other domains
			fmt.Printf("Error stopping domain %s: %v\n", domainType, err)
		}
	}

	dc.isRunning = false
	return nil
}

// processMessages processes inter-domain messages
func (dc *DomainCoordinator) processMessages() {
	for {
		select {
		case message := <-dc.messageQueue:
			dc.handleMessage(message)
		case <-dc.ctx.Done():
			return
		}
	}
}

// handleMessage handles an inter-domain message
func (dc *DomainCoordinator) handleMessage(message *InterDomainMessage) {
	dc.mu.RLock()
	targetDomain, exists := dc.domains[message.ToDomain]
	dc.mu.RUnlock()

	if !exists {
		// Log error - target domain not found
		return
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	if err := targetDomain.ReceiveMessage(ctx, message); err != nil {
		// Log error - failed to deliver message
	}
}

// GetOverallState returns the combined state of all domains
func (dc *DomainCoordinator) GetOverallState() map[DomainType]*DomainState {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	states := make(map[DomainType]*DomainState)
	for domainType, domain := range dc.domains {
		states[domainType] = domain.GetState()
	}

	return states
}

// GetOverallMetrics returns combined metrics from all domains
func (dc *DomainCoordinator) GetOverallMetrics() map[DomainType]DomainMetrics {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	metrics := make(map[DomainType]DomainMetrics)
	for domainType, domain := range dc.domains {
		metrics[domainType] = domain.GetMetrics()
	}

	return metrics
}