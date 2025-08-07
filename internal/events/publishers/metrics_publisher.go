package publishers

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/loreum-org/cortex/internal/events"
)

// MetricsPublisher publishes system metrics events
type MetricsPublisher struct {
	eventBus   *events.EventBus
	server     ServerInterface
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
	interval   time.Duration
}

// ServerInterface defines the interface for getting server metrics
type ServerInterface interface {
	GetConnectionCount() int
	GetUptime() time.Duration
	GetQueryMetrics() map[string]interface{}
	GetRAGMetrics() map[string]interface{}
	GetNetworkMetrics() map[string]interface{}
}

// NewMetricsPublisher creates a new metrics publisher
func NewMetricsPublisher(eventBus *events.EventBus, server ServerInterface, interval time.Duration) *MetricsPublisher {
	if interval == 0 {
		interval = 5 * time.Second
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MetricsPublisher{
		eventBus: eventBus,
		server:   server,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins publishing metrics events
func (mp *MetricsPublisher) Start() {
	mp.ticker = time.NewTicker(mp.interval)
	
	go func() {
		defer mp.ticker.Stop()
		
		log.Printf("📊 Metrics Publisher: Started (interval: %v)", mp.interval)
		
		for {
			select {
			case <-mp.ticker.C:
				mp.publishMetrics()
				
			case <-mp.ctx.Done():
				log.Printf("📊 Metrics Publisher: Stopped")
				return
			}
		}
	}()
}

// Stop stops the metrics publisher
func (mp *MetricsPublisher) Stop() {
	if mp.cancel != nil {
		mp.cancel()
	}
}

// publishMetrics collects and publishes system metrics
func (mp *MetricsPublisher) publishMetrics() {
	// Collect system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	systemMetrics := map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc":         memStats.Alloc,
			"total_alloc":   memStats.TotalAlloc,
			"sys":           memStats.Sys,
			"heap_alloc":    memStats.HeapAlloc,
			"heap_sys":      memStats.HeapSys,
			"heap_in_use":   memStats.HeapInuse,
			"heap_released": memStats.HeapReleased,
			"stack_in_use":  memStats.StackInuse,
			"stack_sys":     memStats.StackSys,
		},
		"gc": map[string]interface{}{
			"num_gc":       memStats.NumGC,
			"pause_total":  memStats.PauseTotalNs,
			"last_gc":      time.Unix(0, int64(memStats.LastGC)),
		},
		"goroutines": runtime.NumGoroutine(),
		"cpu_cores":  runtime.NumCPU(),
		"timestamp":  time.Now(),
	}
	
	// Add server-specific metrics if available
	if mp.server != nil {
		systemMetrics["connections"] = mp.server.GetConnectionCount()
		systemMetrics["uptime"] = mp.server.GetUptime().Seconds()
		
		// Add query metrics
		if queryMetrics := mp.server.GetQueryMetrics(); queryMetrics != nil {
			systemMetrics["queries"] = queryMetrics
		}
		
		// Add RAG metrics
		if ragMetrics := mp.server.GetRAGMetrics(); ragMetrics != nil {
			systemMetrics["rag"] = ragMetrics
		}
		
		// Add network metrics
		if networkMetrics := mp.server.GetNetworkMetrics(); networkMetrics != nil {
			systemMetrics["network"] = networkMetrics
		}
	}
	
	// Create and publish metrics event
	metricsEvent := events.NewEvent(
		events.EventTypeMetricsUpdated,
		"metrics_publisher",
		events.MetricsData{
			Type:      "system_metrics",
			Values:    systemMetrics,
			Source:    "metrics_publisher",
			Timestamp: time.Now(),
		},
	)
	
	// Publish event
	if err := mp.eventBus.Publish(metricsEvent); err != nil {
		log.Printf("❌ Metrics Publisher: Failed to publish metrics: %v", err)
	}
}

// PublishCustomMetrics allows publishing custom metrics
func (mp *MetricsPublisher) PublishCustomMetrics(metricType string, values map[string]interface{}) error {
	metricsEvent := events.NewEvent(
		events.EventTypeMetricsUpdated,
		"metrics_publisher",
		events.MetricsData{
			Type:      metricType,
			Values:    values,
			Source:    "custom",
			Timestamp: time.Now(),
		},
	)
	
	return mp.eventBus.Publish(metricsEvent)
}

// GetInterval returns the current publishing interval
func (mp *MetricsPublisher) GetInterval() time.Duration {
	return mp.interval
}

// SetInterval updates the publishing interval
func (mp *MetricsPublisher) SetInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	
	mp.interval = interval
	
	// Restart ticker with new interval if running
	if mp.ticker != nil {
		mp.ticker.Stop()
		mp.ticker = time.NewTicker(interval)
		log.Printf("📊 Metrics Publisher: Interval updated to %v", interval)
	}
}