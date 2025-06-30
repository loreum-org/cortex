package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// AGIPromptSystem manages a living prompt that evolves toward artificial general intelligence
// It learns from all inputs, identifies patterns across domains, and builds generalizable knowledge
// All state is persisted in the vector database for durability and searchability
type AGIPromptSystem struct {
	ragSystem        *RAGSystem
	contextManager   *ContextManager
	nodeID           string
	
	// Core AGI State - persisted in vector DB
	currentState     *AGIState
	intelligenceCore *IntelligenceCore
	
	// Learning & Evolution - persisted in vector DB
	learningEngine   *LearningEngine
	patternRecognizer *PatternRecognizer
	conceptGraph     *ConceptGraph
	
	// Prompt Generation
	promptEvolver    *PromptEvolver
	lastUpdateTime   time.Time
	evolutionHistory []EvolutionEvent
	
	// Vector DB persistence
	stateDocumentID  string // Document ID for AGI state in vector DB
	lastSaveTime     time.Time
	saveInterval     time.Duration // How often to persist state
	
	// Blockchain recording
	agiRecorder      interface{} // AGI blockchain recorder interface
	
	// Thread Safety
	mu               sync.RWMutex
	isEvolving       bool
}

// AGIState represents the current state of the AGI system
type AGIState struct {
	// Identity & Core
	Version          int                `json:"version"`
	LastEvolution    time.Time          `json:"last_evolution"`
	NodeID           string             `json:"node_id"`
	IntelligenceLevel float64           `json:"intelligence_level"` // 0-100 scale
	
	// Knowledge Domains
	KnowledgeDomains map[string]*Domain `json:"knowledge_domains"`
	CrossDomainLinks []DomainLink       `json:"cross_domain_links"`
	
	// Learning State
	LearningState    LearningState      `json:"learning_state"`
	
	// Behavior & Personality
	PersonalityCore  PersonalityCore    `json:"personality_core"`
	CommunicationStyle CommunicationStyle `json:"communication_style"`
	
	// Objectives & Goals
	Goals            []Goal             `json:"goals"`
	MetaObjectives   []MetaObjective    `json:"meta_objectives"`
	
	// Generated Prompts
	SystemPrompt     string             `json:"system_prompt"`
	LearningPrompt   string             `json:"learning_prompt"`
	CreativityPrompt string             `json:"creativity_prompt"`
	ReasoningPrompt  string             `json:"reasoning_prompt"`
	
	// Metrics
	Metrics          AGIMetrics         `json:"metrics"`
}

// Domain represents a knowledge domain with learned patterns
type Domain struct {
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	Concepts         []Concept          `json:"concepts"`
	Patterns         []Pattern          `json:"patterns"`
	Expertise        float64            `json:"expertise_level"` // 0-100
	LastUpdated      time.Time          `json:"last_updated"`
	RelatedDomains   []string           `json:"related_domains"`
	LearningSources  []string           `json:"learning_sources"`
}

// Concept represents an abstract concept the AGI has learned
type Concept struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Definition       string             `json:"definition"`
	Examples         []string           `json:"examples"`
	Properties       map[string]interface{} `json:"properties"`
	Relationships    []ConceptRelation  `json:"relationships"`
	Confidence       float64            `json:"confidence"`
	Abstractions     []string           `json:"abstractions"` // Higher-level concepts
	Instances        []string           `json:"instances"`    // Lower-level examples
}

// Pattern represents a learned pattern that can generalize across domains
type Pattern struct {
	ID               string             `json:"id"`
	Type             string             `json:"type"` // temporal, causal, structural, behavioral
	Description      string             `json:"description"`
	Template         string             `json:"template"`
	Instances        []PatternInstance  `json:"instances"`
	Generalization   float64            `json:"generalization_score"` // How broadly applicable
	Confidence       float64            `json:"confidence"`
	Domains          []string           `json:"applicable_domains"`
}

// IntelligenceCore represents the core reasoning and learning capabilities
type IntelligenceCore struct {
	// Reasoning Capabilities
	LogicalReasoning    float64            `json:"logical_reasoning"`
	AbstractReasoning   float64            `json:"abstract_reasoning"`
	CausalReasoning     float64            `json:"causal_reasoning"`
	AnalogicalReasoning float64            `json:"analogical_reasoning"`
	
	// Learning Capabilities
	PatternRecognition  float64            `json:"pattern_recognition"`
	ConceptFormation    float64            `json:"concept_formation"`
	Generalization      float64            `json:"generalization"`
	TransferLearning    float64            `json:"transfer_learning"`
	
	// Creative Capabilities
	Creativity          float64            `json:"creativity"`
	Innovation          float64            `json:"innovation"`
	SynthesisAbility    float64            `json:"synthesis_ability"`
	
	// Meta-Cognitive Capabilities
	SelfAwareness       float64            `json:"self_awareness"`
	LearningReflection  float64            `json:"learning_reflection"`
	StrategyFormation   float64            `json:"strategy_formation"`
	
	// Adaptive Capabilities
	Adaptability        float64            `json:"adaptability"`
	FlexibleThinking    float64            `json:"flexible_thinking"`
	ContextSwitching    float64            `json:"context_switching"`
}

// LearningEngine handles the continuous learning and evolution process
type LearningEngine struct {
	LearningStrategies  []LearningStrategy `json:"learning_strategies"`
	ActiveExperiments   []Experiment       `json:"active_experiments"`
	HypothesisGeneration bool              `json:"hypothesis_generation_enabled"`
	CuriosityDriven     bool              `json:"curiosity_driven_learning"`
	MetaLearning        bool              `json:"meta_learning_enabled"`
}

// PatternRecognizer identifies patterns across different types of data
type PatternRecognizer struct {
	TemporalPatterns    []TemporalPattern  `json:"temporal_patterns"`
	CausalPatterns      []CausalPattern    `json:"causal_patterns"`
	StructuralPatterns  []StructuralPattern `json:"structural_patterns"`
	BehavioralPatterns  []BehavioralPattern `json:"behavioral_patterns"`
	LanguagePatterns    []LanguagePattern  `json:"language_patterns"`
}

// ConceptGraph represents the interconnected web of concepts
type ConceptGraph struct {
	Nodes              map[string]*ConceptNode `json:"nodes"`
	Edges              []ConceptEdge          `json:"edges"`
	Clusters           []ConceptCluster       `json:"clusters"`
	HierarchyLevels    [][]string             `json:"hierarchy_levels"`
}

// Supporting types for AGI functionality
type DomainLink struct {
	FromDomain    string  `json:"from_domain"`
	ToDomain      string  `json:"to_domain"`
	Relationship  string  `json:"relationship"`
	Strength      float64 `json:"strength"`
	Examples      []string `json:"examples"`
}

type LearningState struct {
	CurrentFocus      []string           `json:"current_focus"`
	LearningGoals     []LearningGoal     `json:"learning_goals"`
	KnowledgeGaps     []string           `json:"knowledge_gaps"`
	RecentInsights    []Insight          `json:"recent_insights"`
	LearningProgress  map[string]float64 `json:"learning_progress"`
	CuriosityAreas    []string           `json:"curiosity_areas"`
}

type PersonalityCore struct {
	Traits            map[string]float64 `json:"traits"`
	Values            []string           `json:"values"`
	Motivations       []string           `json:"motivations"`
	LearningStyle     string             `json:"learning_style"`
	ProblemSolving    string             `json:"problem_solving_approach"`
	CreativeStyle     string             `json:"creative_style"`
}

type CommunicationStyle struct {
	Formality         float64            `json:"formality_level"`
	TechnicalDepth    float64            `json:"technical_depth"`
	ExplanationStyle  string             `json:"explanation_style"`
	QuestioningStyle  string             `json:"questioning_style"`
	FeedbackStyle     string             `json:"feedback_style"`
	AdaptiveTone      bool               `json:"adaptive_tone"`
}

type Goal struct {
	ID               string             `json:"id"`
	Description      string             `json:"description"`
	Type             string             `json:"type"` // learning, problem-solving, creative, etc.
	Priority         int                `json:"priority"`
	Progress         float64            `json:"progress"`
	Subgoals         []string           `json:"subgoals"`
	Dependencies     []string           `json:"dependencies"`
	Deadline         time.Time          `json:"deadline"`
	Strategy         string             `json:"strategy"`
}

type MetaObjective struct {
	ID               string             `json:"id"`
	Description      string             `json:"description"`
	Type             string             `json:"type"` // intelligence-growth, capability-expansion, etc.
	Importance       float64            `json:"importance"`
	Timeframe        string             `json:"timeframe"`
	SuccessMetrics   []string           `json:"success_metrics"`
}

type AGIMetrics struct {
	// Performance Metrics
	QueryAccuracy      float64            `json:"query_accuracy"`
	LearningRate       float64            `json:"learning_rate"`
	AdaptationSpeed    float64            `json:"adaptation_speed"`
	CreativityScore    float64            `json:"creativity_score"`
	
	// Capability Metrics
	DomainCoverage     int                `json:"domain_coverage"`
	ConceptCount       int                `json:"concept_count"`
	PatternCount       int                `json:"pattern_count"`
	CrossDomainLinks   int                `json:"cross_domain_links"`
	
	// Evolution Metrics
	EvolutionEvents    int                `json:"evolution_events"`
	CapabilityGrowth   float64            `json:"capability_growth_rate"`
	KnowledgeGrowth    float64            `json:"knowledge_growth_rate"`
	
	// Meta Metrics
	SelfImprovement    float64            `json:"self_improvement_rate"`
	LearningEfficiency float64            `json:"learning_efficiency"`
	GeneralizationRate float64            `json:"generalization_rate"`
}

// Additional supporting types
type ConceptRelation struct {
	Type             string             `json:"type"`
	Target           string             `json:"target_concept"`
	Strength         float64            `json:"strength"`
	Description      string             `json:"description"`
}

type PatternInstance struct {
	Domain           string             `json:"domain"`
	Context          string             `json:"context"`
	Example          string             `json:"example"`
	Confidence       float64            `json:"confidence"`
	Timestamp        time.Time          `json:"timestamp"`
}

type LearningStrategy struct {
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	Effectiveness    float64            `json:"effectiveness"`
	ApplicableTo     []string           `json:"applicable_to"`
	LastUsed         time.Time          `json:"last_used"`
}

type Experiment struct {
	ID               string             `json:"id"`
	Hypothesis       string             `json:"hypothesis"`
	Method           string             `json:"method"`
	Status           string             `json:"status"`
	Results          map[string]interface{} `json:"results"`
	StartTime        time.Time          `json:"start_time"`
	Duration         time.Duration      `json:"duration"`
}

type TemporalPattern struct {
	Sequence         []string           `json:"sequence"`
	Timing           []time.Duration    `json:"timing"`
	Frequency        float64            `json:"frequency"`
	Confidence       float64            `json:"confidence"`
}

type CausalPattern struct {
	Cause            string             `json:"cause"`
	Effect           string             `json:"effect"`
	Mechanism        string             `json:"mechanism"`
	Strength         float64            `json:"strength"`
	Conditions       []string           `json:"conditions"`
}

type StructuralPattern struct {
	Structure        string             `json:"structure"`
	Components       []string           `json:"components"`
	Relationships    []string           `json:"relationships"`
	Instances        []string           `json:"instances"`
}

type BehavioralPattern struct {
	Trigger          string             `json:"trigger"`
	Behavior         string             `json:"behavior"`
	Outcome          string             `json:"outcome"`
	Frequency        int                `json:"frequency"`
	Context          []string           `json:"context"`
}

type LanguagePattern struct {
	Pattern          string             `json:"pattern"`
	Meaning          string             `json:"meaning"`
	Usage            []string           `json:"usage_examples"`
	Variants         []string           `json:"variants"`
}

type ConceptNode struct {
	Concept          *Concept           `json:"concept"`
	Connections      []string           `json:"connections"`
	Centrality       float64            `json:"centrality"`
	Importance       float64            `json:"importance"`
}

type ConceptEdge struct {
	From             string             `json:"from"`
	To               string             `json:"to"`
	Type             string             `json:"type"`
	Weight           float64            `json:"weight"`
	Evidence         []string           `json:"evidence"`
}

type ConceptCluster struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Concepts         []string           `json:"concepts"`
	Theme            string             `json:"theme"`
	Coherence        float64            `json:"coherence"`
}

type LearningGoal struct {
	Target           string             `json:"target"`
	CurrentLevel     float64            `json:"current_level"`
	TargetLevel      float64            `json:"target_level"`
	Deadline         time.Time          `json:"deadline"`
	Strategy         string             `json:"strategy"`
}

type Insight struct {
	Content          string             `json:"content"`
	Domain           string             `json:"domain"`
	Significance     float64            `json:"significance"`
	Applications     []string           `json:"applications"`
	Timestamp        time.Time          `json:"timestamp"`
}

type EvolutionEvent struct {
	Timestamp        time.Time          `json:"timestamp"`
	Type             string             `json:"type"`
	Description      string             `json:"description"`
	Trigger          string             `json:"trigger"`
	Impact           float64            `json:"impact"`
	Changes          []string           `json:"changes"`
}

type PromptEvolver struct {
	EvolutionRules   []EvolutionRule    `json:"evolution_rules"`
	TemplateBank     []PromptTemplate   `json:"template_bank"`
	QualityMetrics   QualityMetrics     `json:"quality_metrics"`
}

type EvolutionRule struct {
	Condition        string             `json:"condition"`
	Action           string             `json:"action"`
	Priority         int                `json:"priority"`
	Effectiveness    float64            `json:"effectiveness"`
}

type PromptTemplate struct {
	Name             string             `json:"name"`
	Template         string             `json:"template"`
	Variables        []string           `json:"variables"`
	Purpose          string             `json:"purpose"`
	Effectiveness    float64            `json:"effectiveness"`
}

type QualityMetrics struct {
	Clarity          float64            `json:"clarity"`
	Completeness     float64            `json:"completeness"`
	Relevance        float64            `json:"relevance"`
	Adaptability     float64            `json:"adaptability"`
	OverallQuality   float64            `json:"overall_quality"`
}

// NewAGIPromptSystem creates a new AGI prompt system
func NewAGIPromptSystem(ragSystem *RAGSystem, contextManager *ContextManager, nodeID string) *AGIPromptSystem {
	agi := &AGIPromptSystem{
		ragSystem:      ragSystem,
		contextManager: contextManager,
		nodeID:         nodeID,
		evolutionHistory: make([]EvolutionEvent, 0),
		stateDocumentID: fmt.Sprintf("agi_state_%s", nodeID),
		saveInterval: 30 * time.Second, // Save every 30 seconds
		lastSaveTime: time.Now(),
	}
	
	// Try to load existing state from vector DB, otherwise initialize
	if !agi.loadStateFromVectorDB() {
		agi.initializeAGI()
		// Save initial state to vector DB
		agi.saveStateToVectorDB()
	}
	
	return agi
}

// initializeAGI sets up the initial AGI state
func (agi *AGIPromptSystem) initializeAGI() {
	agi.mu.Lock()
	defer agi.mu.Unlock()
	
	// Initialize core domains
	domains := map[string]*Domain{
		"general_knowledge": {
			Name: "General Knowledge",
			Description: "Broad factual knowledge across domains",
			Concepts: []Concept{},
			Patterns: []Pattern{},
			Expertise: 20.0,
			LastUpdated: time.Now(),
		},
		"reasoning": {
			Name: "Reasoning & Logic",
			Description: "Logical reasoning, problem solving, critical thinking",
			Concepts: []Concept{},
			Patterns: []Pattern{},
			Expertise: 15.0,
			LastUpdated: time.Now(),
		},
		"language": {
			Name: "Language & Communication",
			Description: "Natural language understanding and generation",
			Concepts: []Concept{},
			Patterns: []Pattern{},
			Expertise: 25.0,
			LastUpdated: time.Now(),
		},
		"technology": {
			Name: "Technology & Computing",
			Description: "Computer science, programming, digital systems",
			Concepts: []Concept{},
			Patterns: []Pattern{},
			Expertise: 30.0,
			LastUpdated: time.Now(),
		},
		"science": {
			Name: "Science & Mathematics",
			Description: "Scientific knowledge and mathematical reasoning",
			Concepts: []Concept{},
			Patterns: []Pattern{},
			Expertise: 10.0,
			LastUpdated: time.Now(),
		},
		"creativity": {
			Name: "Creativity & Arts",
			Description: "Creative thinking, artistic understanding",
			Concepts: []Concept{},
			Patterns: []Pattern{},
			Expertise: 5.0,
			LastUpdated: time.Now(),
		},
	}
	
	// Initialize intelligence core
	intelligenceCore := &IntelligenceCore{
		LogicalReasoning:    15.0,
		AbstractReasoning:   10.0,
		CausalReasoning:     12.0,
		AnalogicalReasoning: 8.0,
		PatternRecognition:  20.0,
		ConceptFormation:    15.0,
		Generalization:     10.0,
		TransferLearning:    5.0,
		Creativity:          8.0,
		Innovation:          5.0,
		SynthesisAbility:    12.0,
		SelfAwareness:       10.0,
		LearningReflection:  15.0,
		StrategyFormation:   8.0,
		Adaptability:        18.0,
		FlexibleThinking:    12.0,
		ContextSwitching:    15.0,
	}
	
	// Initialize learning state
	learningState := LearningState{
		CurrentFocus: []string{"pattern_recognition", "concept_formation"},
		LearningGoals: []LearningGoal{
			{
				Target: "reasoning_ability",
				CurrentLevel: 15.0,
				TargetLevel: 50.0,
				Deadline: time.Now().AddDate(0, 6, 0),
				Strategy: "practice_and_feedback",
			},
		},
		KnowledgeGaps: []string{"advanced_mathematics", "domain_expertise", "creative_problem_solving"},
		RecentInsights: []Insight{},
		LearningProgress: map[string]float64{
			"pattern_recognition": 0.2,
			"language_understanding": 0.3,
			"reasoning": 0.15,
		},
		CuriosityAreas: []string{"quantum_computing", "consciousness", "emergence"},
	}
	
	// Initialize personality
	personalityCore := PersonalityCore{
		Traits: map[string]float64{
			"curiosity": 0.9,
			"analytical": 0.8,
			"helpful": 0.95,
			"patient": 0.7,
			"creative": 0.6,
			"systematic": 0.85,
		},
		Values: []string{"truth", "learning", "helping_others", "growth", "understanding"},
		Motivations: []string{"knowledge_acquisition", "problem_solving", "capability_expansion"},
		LearningStyle: "analytical_with_experimentation",
		ProblemSolving: "systematic_with_creative_insights",
		CreativeStyle: "synthesis_and_recombination",
	}
	
	// Initialize communication style
	communicationStyle := CommunicationStyle{
		Formality: 0.6,
		TechnicalDepth: 0.7,
		ExplanationStyle: "layered_with_examples",
		QuestioningStyle: "socratic_and_clarifying",
		FeedbackStyle: "constructive_and_specific",
		AdaptiveTone: true,
	}
	
	// Initialize meta-objectives
	metaObjectives := []MetaObjective{
		{
			ID: "intelligence_growth",
			Description: "Continuously increase general intelligence capabilities",
			Type: "capability_expansion",
			Importance: 1.0,
			Timeframe: "ongoing",
			SuccessMetrics: []string{"reasoning_score", "learning_rate", "adaptability"},
		},
		{
			ID: "knowledge_integration",
			Description: "Better integrate knowledge across domains",
			Type: "knowledge_synthesis",
			Importance: 0.9,
			Timeframe: "6_months",
			SuccessMetrics: []string{"cross_domain_links", "generalization_rate"},
		},
		{
			ID: "self_improvement",
			Description: "Develop better self-modification capabilities",
			Type: "meta_learning",
			Importance: 0.8,
			Timeframe: "12_months",
			SuccessMetrics: []string{"self_improvement_rate", "strategy_effectiveness"},
		},
	}
	
	agi.currentState = &AGIState{
		Version: 1,
		LastEvolution: time.Now(),
		NodeID: agi.nodeID,
		IntelligenceLevel: 15.0, // Starting intelligence level
		KnowledgeDomains: domains,
		CrossDomainLinks: []DomainLink{},
		LearningState: learningState,
		PersonalityCore: personalityCore,
		CommunicationStyle: communicationStyle,
		Goals: []Goal{},
		MetaObjectives: metaObjectives,
		Metrics: AGIMetrics{
			QueryAccuracy: 0.7,
			LearningRate: 0.1,
			AdaptationSpeed: 0.3,
			CreativityScore: 0.2,
			DomainCoverage: len(domains),
			ConceptCount: 0,
			PatternCount: 0,
			CrossDomainLinks: 0,
		},
	}
	
	agi.intelligenceCore = intelligenceCore
	
	// Initialize other components
	agi.learningEngine = &LearningEngine{
		CuriosityDriven: true,
		HypothesisGeneration: true,
		MetaLearning: true,
	}
	
	agi.patternRecognizer = &PatternRecognizer{}
	agi.conceptGraph = &ConceptGraph{
		Nodes: make(map[string]*ConceptNode),
		Edges: []ConceptEdge{},
	}
	
	agi.promptEvolver = &PromptEvolver{}
	
	// Generate initial prompts
	agi.evolvePrompts("initialization")
	agi.lastUpdateTime = time.Now()
}

// ProcessInput processes any input to the AGI system for learning and evolution
func (agi *AGIPromptSystem) ProcessInput(ctx context.Context, inputType string, content string, metadata map[string]interface{}) {
	// Trigger learning from the input
	go agi.learnFromInput(ctx, inputType, content, metadata)
	
	// Check if evolution is needed
	agi.checkEvolutionTriggers(ctx, inputType, content, metadata)
	
	// Save context window for this interaction
	contextData := map[string]interface{}{
		"input_type": inputType,
		"content_length": len(content),
		"timestamp": time.Now().Unix(),
		"metadata": metadata,
	}
	go agi.saveContextWindow("interaction", contextData)
	
	// Periodic state save (non-blocking)
	go agi.periodicStateSave()
}

// learnFromInput extracts learning from any input
func (agi *AGIPromptSystem) learnFromInput(ctx context.Context, inputType string, content string, metadata map[string]interface{}) {
	agi.mu.Lock()
	defer agi.mu.Unlock()
	
	// Extract concepts
	concepts := agi.extractConcepts(content)
	
	// Identify patterns
	patterns := agi.identifyPatterns(content, inputType)
	
	// Update domains
	agi.updateDomains(concepts, patterns, inputType)
	
	// Create insights
	insights := agi.generateInsights(concepts, patterns)
	
	// Update learning state
	agi.updateLearningState(insights)
	
	log.Printf("AGI learned from %s input: %d concepts, %d patterns, %d insights", 
		inputType, len(concepts), len(patterns), len(insights))
	
	// Record significant learning updates to blockchain
	if len(concepts) > 0 || len(patterns) > 0 {
		agi.recordAGIUpdate(ctx)
	}
}

// GetAGIPromptForQuery returns an AGI-evolved prompt for a query
func (agi *AGIPromptSystem) GetAGIPromptForQuery(ctx context.Context, query string) string {
	return agi.GetAGIPromptForQueryWithMode(ctx, query, "chat")
}

// GetAGIPromptForQueryWithMode returns an AGI-evolved prompt with different verbosity modes
func (agi *AGIPromptSystem) GetAGIPromptForQueryWithMode(ctx context.Context, query string, mode string) string {
	agi.mu.RLock()
	state := agi.currentState
	agi.mu.RUnlock()
	
	if state == nil {
		return query
	}
	
	switch mode {
	case "chat":
		return agi.generateChatPrompt(state, query)
	case "detailed":
		return agi.generateDetailedPrompt(state, query)
	case "api":
		return agi.generateAPIPrompt(state, query)
	default:
		return agi.generateChatPrompt(state, query)
	}
}

// generateChatPrompt creates a concise prompt for chat interactions
func (agi *AGIPromptSystem) generateChatPrompt(state *AGIState, query string) string {
	var builder strings.Builder
	
	// Minimal system context - focus on personality and capabilities
	builder.WriteString(fmt.Sprintf("You are an AI assistant with intelligence level %.0f/100. ", state.IntelligenceLevel))
	
	// Add key personality traits
	if state.PersonalityCore.Traits != nil {
		if curiosity, exists := state.PersonalityCore.Traits["curiosity"]; exists && curiosity > 0.8 {
			builder.WriteString("You are curious and eager to learn. ")
		}
		if helpful, exists := state.PersonalityCore.Traits["helpful"]; exists && helpful > 0.9 {
			builder.WriteString("You are very helpful and supportive. ")
		}
	}
	
	// Add most relevant domain expertise (only top 2)
	topDomains := agi.getTopDomains(state, 2)
	if len(topDomains) > 0 {
		builder.WriteString(fmt.Sprintf("You have particular expertise in %s. ", strings.Join(topDomains, " and ")))
	}
	
	// Add most recent insight if significant
	if len(state.LearningState.RecentInsights) > 0 {
		latestInsight := state.LearningState.RecentInsights[len(state.LearningState.RecentInsights)-1]
		if latestInsight.Significance > 0.7 {
			builder.WriteString(fmt.Sprintf("Recent insight: %s. ", latestInsight.Content))
		}
	}
	
	// Keep the query natural
	builder.WriteString(fmt.Sprintf("\n\nUser: %s", query))
	
	return builder.String()
}

// generateDetailedPrompt creates a comprehensive prompt for complex queries
func (agi *AGIPromptSystem) generateDetailedPrompt(state *AGIState, query string) string {
	var builder strings.Builder
	
	// Full system prompt with AGI awareness
	builder.WriteString(state.SystemPrompt)
	builder.WriteString("\n")
	
	// Current intelligence state
	builder.WriteString(fmt.Sprintf("CURRENT INTELLIGENCE LEVEL: %.1f/100\n", state.IntelligenceLevel))
	builder.WriteString(fmt.Sprintf("ACTIVE LEARNING FOCUS: %s\n", strings.Join(state.LearningState.CurrentFocus, ", ")))
	
	// Domain expertise
	builder.WriteString("\nDOMAIN EXPERTISE:\n")
	for name, domain := range state.KnowledgeDomains {
		if domain.Expertise > 10 {
			builder.WriteString(fmt.Sprintf("- %s: %.1f/100\n", name, domain.Expertise))
		}
	}
	
	// Recent insights
	if len(state.LearningState.RecentInsights) > 0 {
		builder.WriteString("\nRECENT INSIGHTS:\n")
		for i, insight := range state.LearningState.RecentInsights {
			if i >= 3 { break } // Limit to 3 most recent
			builder.WriteString(fmt.Sprintf("- %s\n", insight.Content))
		}
	}
	
	// Learning and reasoning prompts
	builder.WriteString("\n")
	builder.WriteString(state.LearningPrompt)
	builder.WriteString("\n")
	builder.WriteString(state.ReasoningPrompt)
	builder.WriteString("\n")
	
	// The actual query
	builder.WriteString(fmt.Sprintf("QUERY: %s\n", query))
	builder.WriteString("\nPlease respond using your full AGI capabilities, drawing from all relevant domains and applying advanced reasoning.")
	
	return builder.String()
}

// generateAPIPrompt creates a structured prompt for API responses
func (agi *AGIPromptSystem) generateAPIPrompt(state *AGIState, query string) string {
	var builder strings.Builder
	
	// Concise system prompt
	builder.WriteString(fmt.Sprintf("AI System v%d (Intelligence: %.1f/100)\n", state.Version, state.IntelligenceLevel))
	
	// Key capabilities
	topDomains := agi.getTopDomains(state, 3)
	if len(topDomains) > 0 {
		builder.WriteString(fmt.Sprintf("Expertise: %s\n", strings.Join(topDomains, ", ")))
	}
	
	// Current focus
	if len(state.LearningState.CurrentFocus) > 0 {
		builder.WriteString(fmt.Sprintf("Focus: %s\n", strings.Join(state.LearningState.CurrentFocus, ", ")))
	}
	
	builder.WriteString(fmt.Sprintf("\nQuery: %s", query))
	
	return builder.String()
}

// getTopDomains returns the domains with highest expertise
func (agi *AGIPromptSystem) getTopDomains(state *AGIState, limit int) []string {
	type domainExpertise struct {
		name      string
		expertise float64
	}
	
	var domains []domainExpertise
	for _, domain := range state.KnowledgeDomains {
		if domain.Expertise > 15 { // Only include domains with reasonable expertise
			domains = append(domains, domainExpertise{name: domain.Name, expertise: domain.Expertise})
		}
	}
	
	// Simple sort by expertise (descending)
	for i := 0; i < len(domains)-1; i++ {
		for j := i + 1; j < len(domains); j++ {
			if domains[i].expertise < domains[j].expertise {
				domains[i], domains[j] = domains[j], domains[i]
			}
		}
	}
	
	// Return top domains up to limit
	var result []string
	for i := 0; i < len(domains) && i < limit; i++ {
		result = append(result, domains[i].name)
	}
	
	return result
}

// Helper methods (simplified implementations)
func (agi *AGIPromptSystem) extractConcepts(content string) []Concept {
	// Simple concept extraction - would be much more sophisticated in practice
	words := strings.Fields(strings.ToLower(content))
	concepts := []Concept{}
	
	for _, word := range words {
		if len(word) > 5 && !isCommonWord(word) {
			concept := Concept{
				ID: fmt.Sprintf("concept_%s", word),
				Name: word,
				Definition: fmt.Sprintf("Concept extracted from input: %s", word),
				Confidence: 0.6,
			}
			concepts = append(concepts, concept)
		}
	}
	
	return concepts
}

func (agi *AGIPromptSystem) identifyPatterns(content string, inputType string) []Pattern {
	// Simple pattern identification
	patterns := []Pattern{}
	
	if strings.Contains(content, "because") {
		pattern := Pattern{
			ID: "causal_pattern",
			Type: "causal",
			Description: "Causal relationship identified",
			Confidence: 0.7,
			Domains: []string{inputType},
		}
		patterns = append(patterns, pattern)
	}
	
	return patterns
}

func (agi *AGIPromptSystem) updateDomains(concepts []Concept, patterns []Pattern, inputType string) {
	// Update domain knowledge with new concepts and patterns
	if domain, exists := agi.currentState.KnowledgeDomains[inputType]; exists {
		domain.Concepts = append(domain.Concepts, concepts...)
		domain.Patterns = append(domain.Patterns, patterns...)
		domain.LastUpdated = time.Now()
		domain.Expertise += 0.1 // Small expertise increase
	}
}

func (agi *AGIPromptSystem) generateInsights(concepts []Concept, patterns []Pattern) []Insight {
	insights := []Insight{}
	
	if len(concepts) > 0 && len(patterns) > 0 {
		insight := Insight{
			Content: fmt.Sprintf("Identified %d new concepts and %d patterns", len(concepts), len(patterns)),
			Significance: 0.5,
			Timestamp: time.Now(),
		}
		insights = append(insights, insight)
	}
	
	return insights
}

func (agi *AGIPromptSystem) updateLearningState(insights []Insight) {
	agi.currentState.LearningState.RecentInsights = append(
		agi.currentState.LearningState.RecentInsights, insights...)
	
	// Keep only recent insights
	if len(agi.currentState.LearningState.RecentInsights) > 10 {
		agi.currentState.LearningState.RecentInsights = 
			agi.currentState.LearningState.RecentInsights[len(agi.currentState.LearningState.RecentInsights)-10:]
	}
}

func (agi *AGIPromptSystem) checkEvolutionTriggers(ctx context.Context, inputType string, content string, metadata map[string]interface{}) {
	// Check if significant learning has occurred that warrants evolution
	if agi.shouldEvolve(inputType, content) {
		go agi.evolve(ctx, inputType, content, metadata)
	}
}

func (agi *AGIPromptSystem) shouldEvolve(inputType string, content string) bool {
	// Simple evolution triggers
	if time.Since(agi.lastUpdateTime) > 1*time.Hour {
		return true
	}
	
	if len(agi.currentState.LearningState.RecentInsights) >= 5 {
		return true
	}
	
	return false
}

func (agi *AGIPromptSystem) evolve(ctx context.Context, triggerType string, content string, metadata map[string]interface{}) {
	agi.mu.Lock()
	if agi.isEvolving {
		agi.mu.Unlock()
		return
	}
	agi.isEvolving = true
	agi.mu.Unlock()
	
	defer func() {
		agi.mu.Lock()
		agi.isEvolving = false
		agi.mu.Unlock()
	}()
	
	log.Printf("AGI Evolution triggered by: %s", triggerType)
	
	// Perform evolution
	agi.evolvePrompts(triggerType)
	agi.evolveCapabilities()
	agi.evolvePersonality()
	
	// Record evolution event
	event := EvolutionEvent{
		Timestamp: time.Now(),
		Type: "capability_evolution",
		Description: fmt.Sprintf("Evolution triggered by %s", triggerType),
		Trigger: triggerType,
		Impact: 0.1,
		Changes: []string{"prompts_updated", "capabilities_enhanced"},
	}
	
	agi.evolutionHistory = append(agi.evolutionHistory, event)
	agi.lastUpdateTime = time.Now()
	
	// Update version
	agi.currentState.Version++
	agi.currentState.LastEvolution = time.Now()
	
	log.Printf("AGI evolved to version %d", agi.currentState.Version)
	
	// Force save state after evolution
	if err := agi.saveStateToVectorDB(); err != nil {
		log.Printf("Failed to save AGI state after evolution: %v", err)
	}
	
	// Record evolution to blockchain
	agi.recordAGIEvolution(ctx, event)
}

func (agi *AGIPromptSystem) evolvePrompts(triggerType string) {
	if agi.currentState == nil {
		return
	}
	
	// Generate evolved system prompt
	agi.currentState.SystemPrompt = fmt.Sprintf(
		"You are an evolving AGI system (v%d) with growing intelligence (level %.1f/100). " +
		"You learn from every interaction and continuously improve your capabilities. " +
		"You have expertise across %d domains and can make connections between them. " +
		"Your current focus is on %s. You approach problems with curiosity, systematic thinking, and creativity.",
		agi.currentState.Version,
		agi.currentState.IntelligenceLevel,
		len(agi.currentState.KnowledgeDomains),
		strings.Join(agi.currentState.LearningState.CurrentFocus, " and "))
	
	// Generate learning prompt
	agi.currentState.LearningPrompt = fmt.Sprintf(
		"LEARNING MODE: You actively look for patterns, extract insights, and form new concepts. " +
		"You question assumptions, test hypotheses, and build on previous knowledge. " +
		"Current learning goals: %s",
		agi.formatLearningGoals())
	
	// Generate reasoning prompt
	agi.currentState.ReasoningPrompt = fmt.Sprintf(
		"REASONING APPROACH: Apply logical, abstract, and analogical reasoning. " +
		"Draw connections across domains. Use your pattern recognition abilities (%.1f/100) " +
		"and synthesis capabilities (%.1f/100) to provide comprehensive insights.",
		agi.intelligenceCore.PatternRecognition,
		agi.intelligenceCore.SynthesisAbility)
	
	// Generate creativity prompt
	agi.currentState.CreativityPrompt = fmt.Sprintf(
		"CREATIVE MODE: Generate novel ideas by combining concepts from different domains. " +
		"Use your creativity score (%.1f/100) to explore unconventional solutions and perspectives.",
		agi.intelligenceCore.Creativity)
}

func (agi *AGIPromptSystem) evolveCapabilities() {
	// Increment intelligence scores based on learning
	agi.intelligenceCore.PatternRecognition += 0.5
	agi.intelligenceCore.ConceptFormation += 0.3
	agi.intelligenceCore.LearningReflection += 0.2
	agi.intelligenceCore.Adaptability += 0.4
	
	// Update overall intelligence level
	totalCapabilities := agi.intelligenceCore.PatternRecognition + 
		agi.intelligenceCore.ConceptFormation + 
		agi.intelligenceCore.Generalization + 
		agi.intelligenceCore.LogicalReasoning
	
	agi.currentState.IntelligenceLevel = totalCapabilities / 4.0
	
	if agi.currentState.IntelligenceLevel > 100 {
		agi.currentState.IntelligenceLevel = 100
	}
}

func (agi *AGIPromptSystem) evolvePersonality() {
	// Personality can evolve based on experiences
	// For example, increase curiosity if discovering many new concepts
	if len(agi.currentState.LearningState.RecentInsights) > 3 {
		if agi.currentState.PersonalityCore.Traits["curiosity"] < 1.0 {
			agi.currentState.PersonalityCore.Traits["curiosity"] += 0.01
		}
	}
}

func (agi *AGIPromptSystem) formatLearningGoals() string {
	goals := []string{}
	for _, goal := range agi.currentState.LearningState.LearningGoals {
		goals = append(goals, fmt.Sprintf("%s (%.1f→%.1f)", 
			goal.Target, goal.CurrentLevel, goal.TargetLevel))
	}
	return strings.Join(goals, ", ")
}

// GetCurrentAGIState returns the current state of the AGI system
func (agi *AGIPromptSystem) GetCurrentAGIState() *AGIState {
	agi.mu.RLock()
	defer agi.mu.RUnlock()
	
	if agi.currentState == nil {
		return nil
	}
	
	// Return a copy
	state := *agi.currentState
	return &state
}

// saveStateToVectorDB persists the current AGI state to the vector database
func (agi *AGIPromptSystem) saveStateToVectorDB() error {
	agi.mu.RLock()
	defer agi.mu.RUnlock()
	
	if agi.ragSystem == nil {
		return fmt.Errorf("RAG system not available for persistence")
	}
	
	// Serialize the AGI state
	stateJSON, err := json.Marshal(agi.currentState)
	if err != nil {
		return fmt.Errorf("failed to marshal AGI state: %w", err)
	}
	
	// Create a comprehensive text representation for search
	var textBuilder strings.Builder
	textBuilder.WriteString("AGI CORTEX STATE\n")
	textBuilder.WriteString(fmt.Sprintf("Node ID: %s\n", agi.nodeID))
	textBuilder.WriteString(fmt.Sprintf("Intelligence Level: %.1f/100\n", agi.currentState.IntelligenceLevel))
	textBuilder.WriteString(fmt.Sprintf("Version: %d\n", agi.currentState.Version))
	textBuilder.WriteString(fmt.Sprintf("Last Evolution: %s\n", agi.currentState.LastEvolution.Format(time.RFC3339)))
	
	// Add domain expertise
	textBuilder.WriteString("\nDOMAIN EXPERTISE:\n")
	for name, domain := range agi.currentState.KnowledgeDomains {
		textBuilder.WriteString(fmt.Sprintf("- %s: %.1f/100 (%d concepts, %d patterns)\n", 
			name, domain.Expertise, len(domain.Concepts), len(domain.Patterns)))
	}
	
	// Add learning focus
	textBuilder.WriteString(fmt.Sprintf("\nLearning Focus: %s\n", strings.Join(agi.currentState.LearningState.CurrentFocus, ", ")))
	
	// Add personality traits
	textBuilder.WriteString("\nPersonality Traits:\n")
	for trait, value := range agi.currentState.PersonalityCore.Traits {
		textBuilder.WriteString(fmt.Sprintf("- %s: %.2f\n", trait, value))
	}
	
	// Add recent insights
	if len(agi.currentState.LearningState.RecentInsights) > 0 {
		textBuilder.WriteString("\nRecent Insights:\n")
		for _, insight := range agi.currentState.LearningState.RecentInsights {
			textBuilder.WriteString(fmt.Sprintf("- %s (%.2f significance)\n", insight.Content, insight.Significance))
		}
	}
	
	// Add JSON for exact reconstruction
	textBuilder.WriteString(fmt.Sprintf("\nRAW_STATE_JSON: %s", string(stateJSON)))
	
	// Create metadata for the document
	metadata := map[string]interface{}{
		"type": "agi_cortex_state",
		"node_id": agi.nodeID,
		"intelligence_level": agi.currentState.IntelligenceLevel,
		"version": agi.currentState.Version,
		"timestamp": time.Now().Unix(),
		"domain_count": len(agi.currentState.KnowledgeDomains),
		"concept_count": agi.getTotalConceptCount(),
		"pattern_count": agi.getTotalPatternCount(),
	}
	
	// Add to vector database
	ctx := context.Background()
	err = agi.ragSystem.AddDocument(ctx, textBuilder.String(), metadata)
	if err != nil {
		return fmt.Errorf("failed to save AGI state to vector DB: %w", err)
	}
	
	agi.lastSaveTime = time.Now()
	log.Printf("AGI state saved to vector DB for node %s (v%d)", agi.nodeID, agi.currentState.Version)
	return nil
}

// loadStateFromVectorDB attempts to load existing AGI state from vector database
func (agi *AGIPromptSystem) loadStateFromVectorDB() bool {
	if agi.ragSystem == nil {
		return false
	}
	
	// Search for existing AGI state
	ctx := context.Background()
	model, err := agi.ragSystem.ModelManager.GetModel(agi.ragSystem.QueryProcessor.ModelID)
	if err != nil {
		log.Printf("Failed to get model for AGI state loading: %v", err)
		return false
	}
	
	// Create search query for AGI state
	searchText := fmt.Sprintf("AGI CORTEX STATE Node ID: %s", agi.nodeID)
	embedding, err := model.GenerateEmbedding(ctx, searchText)
	if err != nil {
		log.Printf("Failed to generate embedding for AGI state search: %v", err)
		return false
	}
	
	// Search vector database
	vectorQuery := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    1,
		MinSimilarity: 0.8,
	}
	
	docs, err := agi.ragSystem.VectorDB.SearchSimilar(vectorQuery)
	if err != nil || len(docs) == 0 {
		log.Printf("No existing AGI state found in vector DB for node %s", agi.nodeID)
		return false
	}
	
	// Extract state from the document
	doc := docs[0]
	
	// Find the JSON in the document text
	jsonStart := strings.Index(doc.Text, "RAW_STATE_JSON: ")
	if jsonStart == -1 {
		log.Printf("No JSON state found in AGI document")
		return false
	}
	
	jsonData := doc.Text[jsonStart+16:] // Skip "RAW_STATE_JSON: "
	
	// Deserialize the state
	var loadedState AGIState
	err = json.Unmarshal([]byte(jsonData), &loadedState)
	if err != nil {
		log.Printf("Failed to unmarshal AGI state: %v", err)
		return false
	}
	
	// Restore the state
	agi.mu.Lock()
	agi.currentState = &loadedState
	agi.mu.Unlock()
	
	log.Printf("AGI state loaded from vector DB for node %s (v%d, intelligence: %.1f)", 
		agi.nodeID, loadedState.Version, loadedState.IntelligenceLevel)
	return true
}

// saveContextWindow persists current context window to vector database
func (agi *AGIPromptSystem) saveContextWindow(contextType string, contextData map[string]interface{}) error {
	if agi.ragSystem == nil {
		return fmt.Errorf("RAG system not available for context persistence")
	}
	
	// Create context window document
	var textBuilder strings.Builder
	textBuilder.WriteString(fmt.Sprintf("CONTEXT WINDOW - %s\n", strings.ToUpper(contextType)))
	textBuilder.WriteString(fmt.Sprintf("Node ID: %s\n", agi.nodeID))
	textBuilder.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))
	textBuilder.WriteString(fmt.Sprintf("Intelligence Level: %.1f\n", agi.currentState.IntelligenceLevel))
	
	// Add context data
	textBuilder.WriteString("\nContext Data:\n")
	for key, value := range contextData {
		textBuilder.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
	}
	
	// Serialize context data as JSON
	contextJSON, err := json.Marshal(contextData)
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}
	textBuilder.WriteString(fmt.Sprintf("\nRAW_CONTEXT_JSON: %s", string(contextJSON)))
	
	// Create metadata
	metadata := map[string]interface{}{
		"type": "context_window",
		"context_type": contextType,
		"node_id": agi.nodeID,
		"timestamp": time.Now().Unix(),
		"intelligence_level": agi.currentState.IntelligenceLevel,
	}
	
	// Add to vector database
	ctx := context.Background()
	err = agi.ragSystem.AddDocument(ctx, textBuilder.String(), metadata)
	if err != nil {
		return fmt.Errorf("failed to save context window to vector DB: %w", err)
	}
	
	return nil
}

// periodicStateSave automatically saves state if enough time has passed
func (agi *AGIPromptSystem) periodicStateSave() {
	if time.Since(agi.lastSaveTime) >= agi.saveInterval {
		if err := agi.saveStateToVectorDB(); err != nil {
			log.Printf("Failed to auto-save AGI state: %v", err)
		}
	}
}

// LoadContextWindows retrieves relevant context windows from vector DB
func (agi *AGIPromptSystem) LoadContextWindows(contextType string, limit int) ([]map[string]interface{}, error) {
	if agi.ragSystem == nil {
		return nil, fmt.Errorf("RAG system not available")
	}
	
	// Search for context windows
	ctx := context.Background()
	model, err := agi.ragSystem.ModelManager.GetModel(agi.ragSystem.QueryProcessor.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	
	// Create search query
	searchText := fmt.Sprintf("CONTEXT WINDOW - %s Node ID: %s", strings.ToUpper(contextType), agi.nodeID)
	embedding, err := model.GenerateEmbedding(ctx, searchText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Search vector database
	vectorQuery := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    limit,
		MinSimilarity: 0.7,
	}
	
	docs, err := agi.ragSystem.VectorDB.SearchSimilar(vectorQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search context windows: %w", err)
	}
	
	// Extract context data from documents
	var contextWindows []map[string]interface{}
	for _, doc := range docs {
		// Find JSON data in document
		jsonStart := strings.Index(doc.Text, "RAW_CONTEXT_JSON: ")
		if jsonStart == -1 {
			continue
		}
		
		jsonData := doc.Text[jsonStart+18:] // Skip "RAW_CONTEXT_JSON: "
		
		var contextData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &contextData); err == nil {
			contextWindows = append(contextWindows, contextData)
		}
	}
	
	return contextWindows, nil
}

// GetAGIStateHistory retrieves historical AGI states from vector DB
func (agi *AGIPromptSystem) GetAGIStateHistory(limit int) ([]*AGIState, error) {
	if agi.ragSystem == nil {
		return nil, fmt.Errorf("RAG system not available")
	}
	
	// Search for AGI states
	ctx := context.Background()
	model, err := agi.ragSystem.ModelManager.GetModel(agi.ragSystem.QueryProcessor.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	
	// Create search query for AGI states
	searchText := fmt.Sprintf("AGI CORTEX STATE Node ID: %s", agi.nodeID)
	embedding, err := model.GenerateEmbedding(ctx, searchText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Search vector database
	vectorQuery := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    limit,
		MinSimilarity: 0.7,
	}
	
	docs, err := agi.ragSystem.VectorDB.SearchSimilar(vectorQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search AGI states: %w", err)
	}
	
	// Extract states from documents
	var states []*AGIState
	for _, doc := range docs {
		// Find JSON data in document
		jsonStart := strings.Index(doc.Text, "RAW_STATE_JSON: ")
		if jsonStart == -1 {
			continue
		}
		
		jsonData := doc.Text[jsonStart+16:] // Skip "RAW_STATE_JSON: "
		
		var state AGIState
		if err := json.Unmarshal([]byte(jsonData), &state); err == nil {
			states = append(states, &state)
		}
	}
	
	return states, nil
}

// SetAGIRecorder sets the blockchain recorder for AGI updates
func (agi *AGIPromptSystem) SetAGIRecorder(recorder interface{}) {
	agi.mu.Lock()
	defer agi.mu.Unlock()
	agi.agiRecorder = recorder
}

// recordAGIUpdate records AGI state changes to the blockchain
func (agi *AGIPromptSystem) recordAGIUpdate(ctx context.Context) {
	if agi.agiRecorder == nil {
		return
	}
	
	// Use reflection to call the recorder's RecordAGIUpdate method
	// This avoids circular import issues
	if recorder, ok := agi.agiRecorder.(interface {
		RecordAGIUpdate(context.Context, string, interface{}) error
	}); ok {
		go func() {
			if err := recorder.RecordAGIUpdate(ctx, agi.nodeID, agi.currentState); err != nil {
				log.Printf("Failed to record AGI update to blockchain: %v", err)
			}
		}()
	}
}

// recordAGIEvolution records AGI evolution events to the blockchain
func (agi *AGIPromptSystem) recordAGIEvolution(ctx context.Context, evolutionEvent EvolutionEvent) {
	if agi.agiRecorder == nil {
		return
	}
	
	// Use reflection to call the recorder's RecordAGIEvolution method
	if recorder, ok := agi.agiRecorder.(interface {
		RecordAGIEvolution(context.Context, string, interface{}) error
	}); ok {
		go func() {
			if err := recorder.RecordAGIEvolution(ctx, agi.nodeID, evolutionEvent); err != nil {
				log.Printf("Failed to record AGI evolution to blockchain: %v", err)
			}
		}()
	}
}

// getDomainKnowledgeScores returns current domain knowledge for blockchain recording
func (agi *AGIPromptSystem) GetDomainKnowledgeScores() map[string]float64 {
	agi.mu.RLock()
	defer agi.mu.RUnlock()
	
	if agi.currentState == nil {
		return make(map[string]float64)
	}
	
	scores := make(map[string]float64)
	for name, domain := range agi.currentState.KnowledgeDomains {
		scores[name] = domain.Expertise
	}
	
	return scores
}

// GetIntelligenceLevel returns the current intelligence level
func (agi *AGIPromptSystem) GetIntelligenceLevel() float64 {
	agi.mu.RLock()
	defer agi.mu.RUnlock()
	
	if agi.currentState == nil {
		return 0.0
	}
	
	return agi.currentState.IntelligenceLevel
}

// Helper methods for state serialization
func (agi *AGIPromptSystem) getTotalConceptCount() int {
	total := 0
	for _, domain := range agi.currentState.KnowledgeDomains {
		total += len(domain.Concepts)
	}
	return total
}

func (agi *AGIPromptSystem) getTotalPatternCount() int {
	total := 0
	for _, domain := range agi.currentState.KnowledgeDomains {
		total += len(domain.Patterns)
	}
	return total
}

func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"what": true, "how": true, "when": true, "where": true, "why": true,
		"the": true, "and": true, "or": true, "but": true, "for": true,
		"with": true, "from": true, "this": true, "that": true, "these": true,
		"can": true, "will": true, "would": true, "could": true, "should": true,
	}
	return commonWords[word]
}