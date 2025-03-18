package types

import (
	"context"
	"time"
)

// SensorEvent represents an event detected by a sensor
type SensorEvent struct {
	SensorID string      `json:"sensor_id"`
	Data     interface{} `json:"data"`
	Time     time.Time   `json:"time"`
}

// EventBus handles the distribution of events to subscribers
type EventBus struct {
	Subscribers map[string][]chan<- SensorEvent `json:"-"` // Map of event type to subscribers
}

// Filter represents a filter for events
type Filter struct {
	Topics    [][]string `json:"topics"`
	Addresses []string   `json:"addresses"`
}

// Sensor interface defines the methods that all sensors must implement
type Sensor interface {
	Start(ctx context.Context) error
	Stop() error
	GetData(query string) ([]byte, error)
	Subscribe(channel chan<- SensorEvent)
}

// DataProcessor processes data from sensors
type DataProcessor struct {
	ProcessingRules map[string]string `json:"processing_rules"`
	Buffer          []SensorEvent     `json:"-"`
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		Subscribers: make(map[string][]chan<- SensorEvent),
	}
}

// Subscribe subscribes to events of a specific type
func (e *EventBus) Subscribe(eventType string, ch chan<- SensorEvent) {
	if _, exists := e.Subscribers[eventType]; !exists {
		e.Subscribers[eventType] = make([]chan<- SensorEvent, 0)
	}
	e.Subscribers[eventType] = append(e.Subscribers[eventType], ch)
}

// Publish publishes an event to all subscribers
func (e *EventBus) Publish(eventType string, event SensorEvent) {
	for _, ch := range e.Subscribers[eventType] {
		ch <- event
	}
}
