package core

import (
	"log"
	"sync"
)

type EventBus struct {
	subscribers map[string][]chan string
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan string),
	}
}

func (eb *EventBus) Subscribe(eventType string) chan string {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan string, 10)
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	return ch
}

func (eb *EventBus) Publish(eventType, message string) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if subscribers, found := eb.subscribers[eventType]; found {
		for _, ch := range subscribers {
			ch <- message
		}
		log.Printf("Event published: %s -> %s", eventType, message)
	}
}
