package main

import (
	"time"

	cortex "github.com/loreum-org/cortex/core"
)

func main() {
	c := cortex.NewCortex()

	// Create Sensors
	s1 := cortex.NewSensor("PriceFeed", 2*time.Second)
	s2 := cortex.NewSensor("WebScraper", 3*time.Second)
	c.Sensors = append(c.Sensors, s1, s2)

	// Create Agents
	a1 := cortex.NewAgent("TradingBot")
	a2 := cortex.NewAgent("SentimentAnalyzer")
	c.Agents = append(c.Agents, a1, a2)

	// Run Sensors
	for _, sensor := range c.Sensors {
		go sensor.Start(c.Ctx)
	}

	// Subscribe Agents to Sensor Data
	for _, agent := range c.Agents {
		go func(agent *cortex.Agent) {
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
