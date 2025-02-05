package core

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

type Sensor struct {
	ID       string
	Name     string
	DataChan chan string
	Interval time.Duration
}

func NewSensor(name string, interval time.Duration) *Sensor {
	return &Sensor{
		ID:       uuid.NewString(),
		Name:     name,
		DataChan: make(chan string, 10),
		Interval: interval,
	}
}

func (s *Sensor) Start(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := "SensorData-" + s.ID
			s.DataChan <- data
			log.Printf("Sensor [%s] emitted data: %s", s.Name, data)
		case <-ctx.Done():
			log.Printf("Sensor [%s] stopping...", s.Name)
			return
		}
	}
}
