package main

import (
	"context"
	"time"

	"github.com/loreum-org/cortex/core"
)

func main() {
	c := core.NewCortex()

	// Create Sensors
	s1 := core.NewSensor("PriceFeed", 2*time.Second)
	s2 := core.NewSensor("WebScraper", 3*time.Second)
	c.Sensors = append(c.Sensors, s1, s2)

	// Create Agents
	a1 := core.NewAgent("TradingBot")
	a2 := core.NewAgent("SentimentAnalyzer")
	c.Agents = append(c.Agents, a1, a2)

	ctx := context.Background()

	// Run Sensors
	for _, sensor := range c.Sensors {
		go sensor.Start(ctx)
	}

	// Subscribe Agents to Sensor Data
	for _, agent := range c.Agents {
		go func(agent *core.Agent) {
			for data := range c.EventBus.Subscribe("sensor-data") {
				agent.ProcessData(data)
			}
		}(agent)
	}

	// Simulate Event Emissions
	go func() {
		for {
			time.Sleep(2 * time.Second)
			c.EventBus.Publish("sensor-data", "New market data received")
		}
	}()

	// Keep Running
	select {}
}
