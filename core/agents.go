package core

import (
	"log"

	"github.com/google/uuid"
)

type Agent struct {
	ID   string
	Name string
}

func NewAgent(name string) *Agent {
	return &Agent{
		ID:   uuid.NewString(),
		Name: name,
	}
}

func (a *Agent) ProcessData(data string) {
	log.Printf("Agent [%s] processing data: %s", a.Name, data)
}
