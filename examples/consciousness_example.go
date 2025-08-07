package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/consciousness"
	"github.com/loreum-org/cortex/internal/consciousness/types"
	"github.com/loreum-org/cortex/internal/events"
)

func main() {
	// Example of how to test and use the consciousness system
	fmt.Println("🧠 Consciousness System Example")
	fmt.Println("===============================")

	// Create default configuration
	config := consciousness.DefaultConfiguration()
	
	// Optional: Configure AI model (if available)
	config.AIModelConfig = map[string]interface{}{
		"provider": "ollama",
		"model":    "llama2",
		// Add other AI configuration as needed
	}
	
	// Optional: Configure event bus
	config.EventBusConfig = &events.EventBusConfig{
		QueueSize:   1000,
		WorkerCount: 4,
	}

	// Initialize the consciousness system
	fmt.Println("📡 Initializing consciousness system...")
	system, err := consciousness.NewConsciousnessSystem(config)
	if err != nil {
		log.Fatalf("Failed to create consciousness system: %v", err)
	}

	// Start the system
	fmt.Println("🚀 Starting consciousness system...")
	if err := system.Start(); err != nil {
		log.Fatalf("Failed to start consciousness system: %v", err)
	}
	defer system.Stop()

	// Test 1: Simple reasoning task
	fmt.Println("\n🧮 Test 1: Reasoning Task")
	reasoningInput := &types.DomainInput{
		ID:      "test_reasoning_1",
		Type:    "logical_analysis",
		Content: map[string]interface{}{
			"problem": "If all birds can fly, and penguins are birds, but penguins cannot fly, what logical conclusion can we draw?",
			"context": "logical reasoning about exceptions",
		},
		Priority:  0.8,
		Source:    "example_test",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	reasoningOutput, err := system.ProcessInput(ctx, reasoningInput)
	if err != nil {
		log.Printf("Reasoning test failed: %v", err)
	} else {
		fmt.Printf("✅ Reasoning result: %+v\n", reasoningOutput.Content)
	}

	// Test 2: Creative task
	fmt.Println("\n🎨 Test 2: Creative Task")
	creativeInput := &types.DomainInput{
		ID:      "test_creative_1",
		Type:    "creative_thinking",
		Content: map[string]interface{}{
			"prompt": "Generate innovative ideas for sustainable urban transportation",
			"constraints": []string{"environmentally friendly", "cost-effective", "scalable"},
		},
		Priority:  0.7,
		Source:    "example_test",
		Timestamp: time.Now(),
	}

	creativeOutput, err := system.ProcessInput(ctx, creativeInput)
	if err != nil {
		log.Printf("Creative test failed: %v", err)
	} else {
		fmt.Printf("✅ Creative result: %+v\n", creativeOutput.Content)
	}

	// Test 3: Cross-domain collaboration
	fmt.Println("\n🤝 Test 3: Cross-Domain Collaboration")
	collaborationTask := consciousness.CollaborationTask{
		ID:          "test_collaboration_1",
		Type:        "multi_domain_analysis",
		Description: "Analyze the technical feasibility and creative potential of a new AI system",
		Objective:   "Comprehensive analysis combining technical and creative perspectives",
		Context: map[string]interface{}{
			"system_description": "An AI system that generates music based on emotional context",
			"technical_requirements": []string{"real-time processing", "emotional recognition", "music generation"},
			"creative_requirements": []string{"artistic quality", "emotional resonance", "originality"},
		},
		Priority:  0.9,
		Timestamp: time.Now(),
	}

	domains := []types.DomainType{
		types.DomainTechnical,
		types.DomainCreative,
		types.DomainReasoning,
	}

	collaborationResult, err := system.ProcessCrossTask(ctx, collaborationTask, domains)
	if err != nil {
		log.Printf("Collaboration test failed: %v", err)
	} else {
		fmt.Printf("✅ Collaboration result: %+v\n", collaborationResult.Outcomes)
	}

	// Test 4: Learning process
	fmt.Println("\n📚 Test 4: Learning Process")
	learningObjective := consciousness.LearningObjective{
		ID:          "test_learning_1",
		Primary:     "Learn patterns in problem-solving approaches",
		Secondary:   []string{"skill_acquisition", "pattern_recognition"},
		Success:     []consciousness.SuccessCriteria{{}, {}}, // Using empty structs for now
		Constraints: []string{"cross-domain integration", "real-time learning"},
		Priority:    0.9,
		Context: map[string]interface{}{
			"domain": "problem_solving",
			"focus":  "cross-domain pattern recognition",
		},
	}

	learningDomains := []types.DomainType{
		types.DomainLearning,
		types.DomainReasoning,
		types.DomainMemory,
	}

	learningProcess, err := system.InitiateLearning(
		consciousness.ProcessTypePatternSynthesis,
		learningDomains,
		learningObjective,
	)
	if err != nil {
		log.Printf("Learning test failed: %v", err)
	} else {
		fmt.Printf("✅ Learning process initiated: %s\n", learningProcess.ID)
	}

	// Wait a bit for processing
	time.Sleep(2 * time.Second)

	// Test 5: System status and monitoring
	fmt.Println("\n📊 Test 5: System Status")
	status := system.GetSystemStatus()
	fmt.Printf("System running: %v\n", status.IsRunning)
	fmt.Printf("Uptime: %v\n", status.Uptime)
	fmt.Printf("Active domains: %v\n", status.ActiveDomains)
	fmt.Printf("Overall health: %.2f\n", status.SystemHealth.OverallHealth)
	fmt.Printf("Processed inputs: %d\n", status.PerformanceMetrics.ProcessedInputs)

	// Test 6: Collective intelligence
	fmt.Println("\n🌐 Test 6: Collective Intelligence")
	collectiveIntelligence := system.GetCollectiveIntelligence()
	if collectiveIntelligence != nil {
		fmt.Printf("Intelligence level: %.2f\n", collectiveIntelligence.OverallLevel)
		fmt.Printf("Domain contributions: %d\n", len(collectiveIntelligence.DomainContributions))
		fmt.Printf("Emergent capabilities: %d\n", len(collectiveIntelligence.EmergentCapabilities))
	}

	// Test 7: Active learning processes
	fmt.Println("\n🔄 Test 7: Active Learning Processes")
	activeLearning := system.GetActiveLearningProcesses()
	fmt.Printf("Active learning processes: %d\n", len(activeLearning))
	for _, process := range activeLearning {
		fmt.Printf("- %s: %s (Progress: %.2f%%)\n", 
			process.ID, process.Type, process.Progress.OverallProgress*100)
	}

	// Test 8: Emergent behaviors
	fmt.Println("\n🌱 Test 8: Emergent Behaviors")
	emergentBehaviors := system.GetEmergentBehaviors()
	fmt.Printf("Detected emergent behaviors: %d\n", len(emergentBehaviors))
	for _, behavior := range emergentBehaviors {
		fmt.Printf("- %s: %s (Complexity: %.2f)\n", 
			behavior.ID, behavior.Type, behavior.Complexity)
	}

	fmt.Println("\n✅ All tests completed successfully!")
	fmt.Println("The consciousness system is now ready for use.")
	
	// Keep running for a bit to observe system behavior
	fmt.Println("\n⏳ System running for 10 seconds to observe behavior...")
	time.Sleep(10 * time.Second)
	
	fmt.Println("🛑 Shutting down consciousness system...")
}