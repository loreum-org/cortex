package sensors

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// MockEthClient is a mock Ethereum client for demonstration purposes
type MockEthClient struct{}

// MockLog represents a log event from the blockchain
type MockLog struct {
	Address     string
	Topics      []string
	Data        []byte
	BlockNumber uint64
	TxHash      string
	TxIndex     uint
	BlockHash   string
	Index       uint
	Removed     bool
}

// BlockchainSensor implements the Sensor interface for blockchain networks
type BlockchainSensor struct {
	client      *MockEthClient
	filters     map[string]*types.Filter
	eventChan   chan types.SensorEvent
	subscribers []chan<- types.SensorEvent
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	sensorID    string
}

// NewBlockchainSensor creates a new blockchain sensor
func NewBlockchainSensor(endpoint string, sensorID string) (*BlockchainSensor, error) {
	// In a real implementation, this would connect to an actual blockchain node
	client := &MockEthClient{}

	ctx, cancel := context.WithCancel(context.Background())

	return &BlockchainSensor{
		client:      client,
		filters:     make(map[string]*types.Filter),
		eventChan:   make(chan types.SensorEvent, 100),
		subscribers: make([]chan<- types.SensorEvent, 0),
		ctx:         ctx,
		cancel:      cancel,
		sensorID:    sensorID,
	}, nil
}

// Start starts the blockchain sensor
func (s *BlockchainSensor) Start(ctx context.Context) error {
	s.mu.RLock()
	filterCount := len(s.filters)
	s.mu.RUnlock()

	if filterCount == 0 {
		return errors.New("no filters set up")
	}

	// Start processing events
	go s.processEvents()

	// Set up event listeners for each filter
	s.mu.RLock()
	for id, filter := range s.filters {
		go s.watchEvents(id, filter)
	}
	s.mu.RUnlock()

	return nil
}

// Stop stops the blockchain sensor
func (s *BlockchainSensor) Stop() error {
	s.cancel()
	return nil
}

// GetData retrieves data based on a query
func (s *BlockchainSensor) GetData(query string) ([]byte, error) {
	// In a real implementation, this would query the blockchain
	return []byte("Blockchain data for query: " + query), nil
}

// Subscribe adds a subscriber to the sensor
func (s *BlockchainSensor) Subscribe(channel chan<- types.SensorEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers = append(s.subscribers, channel)
}

// AddFilter adds a filter to the sensor
func (s *BlockchainSensor) AddFilter(id string, filter *types.Filter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.filters[id] = filter
}

// processEvents processes events from the event channel and distributes them to subscribers
func (s *BlockchainSensor) processEvents() {
	for {
		select {
		case event := <-s.eventChan:
			s.mu.RLock()
			for _, subscriber := range s.subscribers {
				subscriber <- event
			}
			s.mu.RUnlock()
		case <-s.ctx.Done():
			return
		}
	}
}

// watchEvents watches for events matching a filter
func (s *BlockchainSensor) watchEvents(id string, filter *types.Filter) {
	// In a real implementation, this would set up a subscription to blockchain events
	// For demonstration, we'll generate mock events
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Generate a mock event
			event := types.SensorEvent{
				SensorID: s.sensorID,
				Data: MockLog{
					Address:     "0x1234567890abcdef1234567890abcdef12345678",
					Topics:      []string{"0xabcdef"},
					Data:        []byte("Mock log data"),
					BlockNumber: 12345678,
					TxHash:      "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				},
				Time: time.Now(),
			}

			// Send the event to the event channel
			select {
			case s.eventChan <- event:
				// Event sent successfully
			default:
				// Channel is full, log this in a real implementation
			}
		case <-s.ctx.Done():
			return
		}
	}
}
