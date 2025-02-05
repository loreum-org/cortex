package core

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Cortex struct {
	ID       string
	Sensors  []*Sensor
	Agents   []*Agent
	VectorDB *VectorDatabase
	EventBus *EventBus
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewCortex() *Cortex {
	ctx, cancel := context.WithCancel(context.Background())

	return &Cortex{
		ID:       uuid.NewString(),
		Sensors:  []*Sensor{},
		Agents:   []*Agent{},
		VectorDB: NewVectorDatabase(),
		EventBus: NewEventBus(),
		ctx:      ctx,
		cancel:   cancel,
	}
}
