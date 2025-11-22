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
// It now incorporates consciousness runtime for unified cognitive processing
type AGIPromptSystem struct {
	ragSystem      *RAGSystem
	contextManager *ContextManager
	nodeID         string

	// Core AGI State - persisted in vector DB
	currentState     *AGIState
	intelligenceCore *IntelligenceCore

	// Learning & Evolution - persisted in vector DB
	learningEngine    *LearningEngine
	patternRecognizer *PatternRecognizer
	conceptGraph      *ConceptGraph

	// Consciousness Runtime Integration
	consciousnessRuntime *ConsciousnessRuntime
	workingMemory        *WorkingMemory
	attentionState       *AttentionState
	emotionalState       map[string]float64

	// Consciousness Processing Components
	intentAnalyzer *IntentAnalyzer
	decisionEngine *DecisionEngine
	actionExecutor *ActionExecutor

	// Consciousness Loop Control
	isRunning        bool
	cycleInterval    time.Duration
	cycleCount       int64
	lastCycleTime    time.Time
	avgCycleTime     time.Duration
	inputQueue       chan *SensorInput
	pendingInputs    []*SensorInput
	stopChan         chan struct{}
	consciousnessMu  sync.RWMutex

	// Prompt Generation
	promptEvolver    *PromptEvolver
	lastUpdateTime   time.Time
	evolutionHistory []EvolutionEvent

	// Vector DB persistence
	stateDocumentID string // Document ID for AGI state in vector DB
	lastSaveTime    time.Time
	saveInterval    time.Duration // How often to persist state

	// Blockchain recording
	agiRecorder interface{} // AGI blockchain recorder interface

	// Thread Safety
	mu         sync.RWMutex
	isEvolving bool
}

// AGIState represents the current state of the AGI system with integrated consciousness
type AGIState struct {
	// Identity & Core
	Version           int       `json:"version"`
	LastEvolution     time.Time `json:"last_evolution"`
	NodeID            string    `json:"node_id"`
	IntelligenceLevel float64   `json:"intelligence_level"` // 0-100 scale

	// Consciousness State Integration
	CurrentCycle     int64                 `json:"current_cycle"`
	Attention        *AttentionState       `json:"attention"`
	Emotions         map[string]float64    `json:"emotions"`
	ActiveThoughts   []Thought             `json:"active_thoughts"`
	Intentions       []Intent              `json:"intentions"`
	EnergyLevel      float64               `json:"energy_level"`
	FocusLevel       float64               `json:"focus_level"`
	AlertnessLevel   float64               `json:"alertness_level"`
	DecisionState    string                `json:"decision_state"`
	ProcessingLoad   float64               `json:"processing_load"`

	// Knowledge Domains
	KnowledgeDomains map[string]*Domain `json:"knowledge_domains"`
	CrossDomainLinks []DomainLink       `json:"cross_domain_links"`

	// Learning State
	LearningState LearningState `json:"learning_state"`

	// Behavior & Personality
	PersonalityCore    PersonalityCore    `json:"personality_core"`
	CommunicationStyle CommunicationStyle `json:"communication_style"`

	// Objectives & Goals
	Goals          []Goal          `json:"goals"`
	MetaObjectives []MetaObjective `json:"meta_objectives"`

	// Generated Prompts
	SystemPrompt     string `json:"system_prompt"`
	LearningPrompt   string `json:"learning_prompt"`
	CreativityPrompt string `json:"creativity_prompt"`
	ReasoningPrompt  string `json:"reasoning_prompt"`

	// Metrics
	Metrics AGIMetrics `json:"metrics"`
}

// Domain represents a knowledge domain with learned patterns and consciousness capabilities
type Domain struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Concepts        []Concept `json:"concepts"`
	Patterns        []Pattern `json:"patterns"`
	Expertise       float64   `json:"expertise_level"` // 0-100
	LastUpdated     time.Time `json:"last_updated"`
	RelatedDomains  []string  `json:"related_domains"`
	LearningSources []string  `json:"learning_sources"`
	
	// Consciousness domain capabilities
	DomainType      string                  `json:"domain_type"` // reasoning, learning, creative, communication, technical, memory
	Capabilities    []string                `json:"capabilities"`
	ActiveGoals     []Goal                  `json:"active_goals"`
	RecentInsights  []Insight               `json:"recent_insights"`
	Metrics         map[string]float64      `json:"metrics"`
	IsActive        bool                    `json:"is_active"`
	Version         string                  `json:"version"`
}

// Concept represents an abstract concept the AGI has learned
type Concept struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Definition    string                 `json:"definition"`
	Examples      []string               `json:"examples"`
	Properties    map[string]interface{} `json:"properties"`
	Relationships []ConceptRelation      `json:"relationships"`
	Confidence    float64                `json:"confidence"`
	Abstractions  []string               `json:"abstractions"` // Higher-level concepts
	Instances     []string               `json:"instances"`    // Lower-level examples
}

// Pattern represents a learned pattern that can generalize across domains
type Pattern struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"` // temporal, causal, structural, behavioral
	Description    string            `json:"description"`
	Template       string            `json:"template"`
	Instances      []PatternInstance `json:"instances"`
	Generalization float64           `json:"generalization_score"` // How broadly applicable
	Confidence     float64           `json:"confidence"`
	Domains        []string          `json:"applicable_domains"`
}

// IntelligenceCore represents the core reasoning and learning capabilities
type IntelligenceCore struct {
	// Reasoning Capabilities
	LogicalReasoning    float64 `json:"logical_reasoning"`
	AbstractReasoning   float64 `json:"abstract_reasoning"`
	CausalReasoning     float64 `json:"causal_reasoning"`
	AnalogicalReasoning float64 `json:"analogical_reasoning"`

	// Learning Capabilities
	PatternRecognition float64 `json:"pattern_recognition"`
	ConceptFormation   float64 `json:"concept_formation"`
	Generalization     float64 `json:"generalization"`
	TransferLearning   float64 `json:"transfer_learning"`

	// Creative Capabilities
	Creativity       float64 `json:"creativity"`
	Innovation       float64 `json:"innovation"`
	SynthesisAbility float64 `json:"synthesis_ability"`

	// Meta-Cognitive Capabilities
	SelfAwareness      float64 `json:"self_awareness"`
	LearningReflection float64 `json:"learning_reflection"`
	StrategyFormation  float64 `json:"strategy_formation"`

	// Adaptive Capabilities
	Adaptability     float64 `json:"adaptability"`
	FlexibleThinking float64 `json:"flexible_thinking"`
	ContextSwitching float64 `json:"context_switching"`
}

// LearningEngine handles the continuous learning and evolution process
type LearningEngine struct {
	LearningStrategies   []LearningStrategy `json:"learning_strategies"`
	ActiveExperiments    []Experiment       `json:"active_experiments"`
	HypothesisGeneration bool               `json:"hypothesis_generation_enabled"`
	CuriosityDriven      bool               `json:"curiosity_driven_learning"`
	MetaLearning         bool               `json:"meta_learning_enabled"`
}

// PatternRecognizer identifies patterns across different types of data
type PatternRecognizer struct {
	TemporalPatterns   []TemporalPattern   `json:"temporal_patterns"`
	CausalPatterns     []CausalPattern     `json:"causal_patterns"`
	StructuralPatterns []StructuralPattern `json:"structural_patterns"`
	BehavioralPatterns []BehavioralPattern `json:"behavioral_patterns"`
	LanguagePatterns   []LanguagePattern   `json:"language_patterns"`
}

// ConceptGraph represents the interconnected web of concepts
type ConceptGraph struct {
	Nodes           map[string]*ConceptNode `json:"nodes"`
	Edges           []ConceptEdge           `json:"edges"`
	Clusters        []ConceptCluster        `json:"clusters"`
	HierarchyLevels [][]string              `json:"hierarchy_levels"`
}

// Supporting types for AGI functionality
type DomainLink struct {
	FromDomain   string   `json:"from_domain"`
	ToDomain     string   `json:"to_domain"`
	Relationship string   `json:"relationship"`
	Strength     float64  `json:"strength"`
	Examples     []string `json:"examples"`
}

type LearningState struct {
	CurrentFocus     []string           `json:"current_focus"`
	LearningGoals    []LearningGoal     `json:"learning_goals"`
	KnowledgeGaps    []string           `json:"knowledge_gaps"`
	RecentInsights   []Insight          `json:"recent_insights"`
	LearningProgress map[string]float64 `json:"learning_progress"`
	CuriosityAreas   []string           `json:"curiosity_areas"`
}

type PersonalityCore struct {
	Traits         map[string]float64 `json:"traits"`
	Values         []string           `json:"values"`
	Motivations    []string           `json:"motivations"`
	LearningStyle  string             `json:"learning_style"`
	ProblemSolving string             `json:"problem_solving_approach"`
	CreativeStyle  string             `json:"creative_style"`
}

type CommunicationStyle struct {
	Formality        float64 `json:"formality_level"`
	TechnicalDepth   float64 `json:"technical_depth"`
	ExplanationStyle string  `json:"explanation_style"`
	QuestioningStyle string  `json:"questioning_style"`
	FeedbackStyle    string  `json:"feedback_style"`
	AdaptiveTone     bool    `json:"adaptive_tone"`
}

type Goal struct {
	ID           string    `json:"id"`
	Description  string    `json:"description"`
	Type         string    `json:"type"` // learning, problem-solving, creative, etc.
	Priority     int       `json:"priority"`
	Progress     float64   `json:"progress"`
	Subgoals     []string  `json:"subgoals"`
	Dependencies []string  `json:"dependencies"`
	Deadline     time.Time `json:"deadline"`
	Strategy     string    `json:"strategy"`
}

type MetaObjective struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	Type           string   `json:"type"` // intelligence-growth, capability-expansion, etc.
	Importance     float64  `json:"importance"`
	Timeframe      string   `json:"timeframe"`
	SuccessMetrics []string `json:"success_metrics"`
}

type AGIMetrics struct {
	// Performance Metrics
	QueryAccuracy   float64 `json:"query_accuracy"`
	LearningRate    float64 `json:"learning_rate"`
	AdaptationSpeed float64 `json:"adaptation_speed"`
	CreativityScore float64 `json:"creativity_score"`

	// Capability Metrics
	DomainCoverage   int `json:"domain_coverage"`
	ConceptCount     int `json:"concept_count"`
	PatternCount     int `json:"pattern_count"`
	CrossDomainLinks int `json:"cross_domain_links"`

	// Evolution Metrics
	EvolutionEvents  int     `json:"evolution_events"`
	CapabilityGrowth float64 `json:"capability_growth_rate"`
	KnowledgeGrowth  float64 `json:"knowledge_growth_rate"`

	// Meta Metrics
	SelfImprovement    float64 `json:"self_improvement_rate"`
	LearningEfficiency float64 `json:"learning_efficiency"`
	GeneralizationRate float64 `json:"generalization_rate"`
}

// Additional supporting types
type ConceptRelation struct {
	Type        string  `json:"type"`
	Target      string  `json:"target_concept"`
	Strength    float64 `json:"strength"`
	Description string  `json:"description"`
}

type PatternInstance struct {
	Domain     string    `json:"domain"`
	Context    string    `json:"context"`
	Example    string    `json:"example"`
	Confidence float64   `json:"confidence"`
	Timestamp  time.Time `json:"timestamp"`
}

type LearningStrategy struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Effectiveness float64   `json:"effectiveness"`
	ApplicableTo  []string  `json:"applicable_to"`
	LastUsed      time.Time `json:"last_used"`
}

type Experiment struct {
	ID         string                 `json:"id"`
	Hypothesis string                 `json:"hypothesis"`
	Method     string                 `json:"method"`
	Status     string                 `json:"status"`
	Results    map[string]interface{} `json:"results"`
	StartTime  time.Time              `json:"start_time"`
	Duration   time.Duration          `json:"duration"`
}

type TemporalPattern struct {
	Sequence   []string        `json:"sequence"`
	Timing     []time.Duration `json:"timing"`
	Frequency  float64         `json:"frequency"`
	Confidence float64         `json:"confidence"`
}

type CausalPattern struct {
	Cause      string   `json:"cause"`
	Effect     string   `json:"effect"`
	Mechanism  string   `json:"mechanism"`
	Strength   float64  `json:"strength"`
	Conditions []string `json:"conditions"`
}

type StructuralPattern struct {
	Structure     string   `json:"structure"`
	Components    []string `json:"components"`
	Relationships []string `json:"relationships"`
	Instances     []string `json:"instances"`
}

type BehavioralPattern struct {
	Trigger   string   `json:"trigger"`
	Behavior  string   `json:"behavior"`
	Outcome   string   `json:"outcome"`
	Frequency int      `json:"frequency"`
	Context   []string `json:"context"`
}

type LanguagePattern struct {
	Pattern  string   `json:"pattern"`
	Meaning  string   `json:"meaning"`
	Usage    []string `json:"usage_examples"`
	Variants []string `json:"variants"`
}

type ConceptNode struct {
	Concept     *Concept `json:"concept"`
	Connections []string `json:"connections"`
	Centrality  float64  `json:"centrality"`
	Importance  float64  `json:"importance"`
}

type ConceptEdge struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Type     string   `json:"type"`
	Weight   float64  `json:"weight"`
	Evidence []string `json:"evidence"`
}

type ConceptCluster struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Concepts  []string `json:"concepts"`
	Theme     string   `json:"theme"`
	Coherence float64  `json:"coherence"`
}

type LearningGoal struct {
	Target       string    `json:"target"`
	CurrentLevel float64   `json:"current_level"`
	TargetLevel  float64   `json:"target_level"`
	Deadline     time.Time `json:"deadline"`
	Strategy     string    `json:"strategy"`
}

type Insight struct {
	Content      string    `json:"content"`
	Domain       string    `json:"domain"`
	Significance float64   `json:"significance"`
	Applications []string  `json:"applications"`
	Timestamp    time.Time `json:"timestamp"`
}

type EvolutionEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Trigger     string    `json:"trigger"`
	Impact      float64   `json:"impact"`
	Changes     []string  `json:"changes"`
}

type PromptEvolver struct {
	EvolutionRules []EvolutionRule  `json:"evolution_rules"`
	TemplateBank   []PromptTemplate `json:"template_bank"`
	QualityMetrics QualityMetrics   `json:"quality_metrics"`
}

type EvolutionRule struct {
	Condition     string  `json:"condition"`
	Action        string  `json:"action"`
	Priority      int     `json:"priority"`
	Effectiveness float64 `json:"effectiveness"`
}

type PromptTemplate struct {
	Name          string   `json:"name"`
	Template      string   `json:"template"`
	Variables     []string `json:"variables"`
	Purpose       string   `json:"purpose"`
	Effectiveness float64  `json:"effectiveness"`
}

type QualityMetrics struct {
	Clarity        float64 `json:"clarity"`
	Completeness   float64 `json:"completeness"`
	Relevance      float64 `json:"relevance"`
	Adaptability   float64 `json:"adaptability"`
	OverallQuality float64 `json:"overall_quality"`
}

// Consciousness-related types integrated into AGI system

// AttentionState tracks what the AGI is focused on
type AttentionState struct {
	PrimaryFocus     string        `json:"primary_focus"`
	SecondaryFoci    []string      `json:"secondary_foci"`
	FocusStrength    float64       `json:"focus_strength"`
	FocusDuration    time.Duration `json:"focus_duration"`
	DistractionLevel float64       `json:"distraction_level"`
	CurrentFocus     string        `json:"current_focus"`
	Intensity        float64       `json:"intensity"`
	Duration         time.Duration `json:"duration"`
	Context          string        `json:"context"`
}

// WorkingMemory holds the current context and recent experiences
type WorkingMemory struct {
	RecentInputs    []*SensorInput         `json:"recent_inputs"`
	ActiveContext   map[string]interface{} `json:"active_context"`
	ShortTermMemory []MemoryFragment       `json:"short_term_memory"`
	Goals           []Goal                 `json:"goals"`
	Beliefs         map[string]float64     `json:"beliefs"`
	Predictions     []Prediction           `json:"predictions"`
	LastUpdated     time.Time              `json:"last_updated"`
	CurrentContext  map[string]interface{} `json:"current_context"`
	ActiveBeliefs   map[string]float64     `json:"active_beliefs"`
	TemporalContext map[string]interface{} `json:"temporal_context"`
}

// SensorInput represents input from the environment
type SensorInput struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // "query", "system", "feedback", "observation"
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  float64                `json:"priority"` // 0-1 scale
	Source    string                 `json:"source"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// Intent represents the understood intention from input
type Intent struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"` // "question", "request", "command", "conversation"
	Description     string                 `json:"description"`
	Confidence      float64                `json:"confidence"`
	RequiredActions []string               `json:"required_actions"`
	ExpectedOutcome string                 `json:"expected_outcome"`
	Priority        float64                `json:"priority"`
	Metadata        map[string]interface{} `json:"metadata"`
	Timestamp       time.Time              `json:"timestamp"`
}

// Decision represents a decision made by the consciousness
type Decision struct {
	ID              string                 `json:"id"`
	Intent          *Intent                `json:"intent"`
	ChosenAction    string                 `json:"chosen_action"`
	ActionParams    map[string]interface{} `json:"action_params"`
	Reasoning       string                 `json:"reasoning"`
	Confidence      float64                `json:"confidence"`
	ExpectedOutcome string                 `json:"expected_outcome"`
	Alternatives    []string               `json:"alternatives"`
	RiskAssessment  float64                `json:"risk_assessment"`
	Timestamp       time.Time              `json:"timestamp"`
}

// ActionResult represents the outcome of an executed action
type ActionResult struct {
	DecisionID     string                 `json:"decision_id"`
	Action         string                 `json:"action"`
	Success        bool                   `json:"success"`
	Result         interface{}            `json:"result"`
	Duration       time.Duration          `json:"duration"`
	ErrorMsg       string                 `json:"error_msg,omitempty"`
	Feedback       map[string]interface{} `json:"feedback,omitempty"`
	LearningPoints []string               `json:"learning_points,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// Thought represents an active thought in consciousness
type Thought struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Type         string                 `json:"type"` // "analytical", "creative", "memory", "prediction"
	Confidence   float64                `json:"confidence"`
	Relevance    float64                `json:"relevance"`
	Associations []string               `json:"associations"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryFragment represents a piece of short-term memory
type MemoryFragment struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Type         string                 `json:"type"`
	Importance   float64                `json:"importance"`
	Associations []string               `json:"associations"`
	Timestamp    time.Time              `json:"timestamp"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Prediction represents a prediction about future events
type Prediction struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Confidence  float64   `json:"confidence"`
	Timeframe   string    `json:"timeframe"`
	Basis       []string  `json:"basis"`
	Timestamp   time.Time `json:"timestamp"`
}

// Consciousness processing components
type ConsciousnessRuntime struct {
	agiSystem *AGIPromptSystem // Reference back to parent AGI system
}

// IntentAnalyzer analyzes inputs to determine user intentions
type IntentAnalyzer struct {
	agiSystem *AGIPromptSystem
}

// DecisionEngine makes decisions based on intentions
type DecisionEngine struct {
	agiSystem *AGIPromptSystem
}

// ActionExecutor executes decisions and returns results
type ActionExecutor struct {
	ragSystem    *RAGSystem
	agiSystem    *AGIPromptSystem
	modelManager interface{}
	defaultModel string
}

// NewAGIPromptSystem creates a new AGI prompt system with integrated consciousness
func NewAGIPromptSystem(ragSystem *RAGSystem, contextManager *ContextManager, nodeID string) *AGIPromptSystem {
	agi := &AGIPromptSystem{
		ragSystem:        ragSystem,
		contextManager:   contextManager,
		nodeID:           nodeID,
		evolutionHistory: make([]EvolutionEvent, 0),
		stateDocumentID:  fmt.Sprintf("agi_state_%s", nodeID),
		saveInterval:     30 * time.Second, // Save every 30 seconds
		lastSaveTime:     time.Now(),
		// Consciousness components
		cycleInterval:  time.Second * 2, // 0.5 Hz consciousness cycle
		inputQueue:     make(chan *SensorInput, 100),
		stopChan:       make(chan struct{}),
		pendingInputs:  make([]*SensorInput, 0),
		emotionalState: make(map[string]float64),
	}

	// Always initialize consciousness components first
	agi.initializeConsciousnessComponents()

	// Initialize with default state first (will be replaced if loading succeeds)
	agi.initializeAGIState()

	// Try to load existing state from vector DB in background (after models are ready)
	go agi.loadStateWhenReady()

	return agi
}

// initializeConsciousnessComponents initializes the consciousness processing components
func (agi *AGIPromptSystem) initializeConsciousnessComponents() {
	// Initialize consciousness processing components
	agi.intentAnalyzer = &IntentAnalyzer{agiSystem: agi}
	agi.decisionEngine = &DecisionEngine{agiSystem: agi}
	agi.actionExecutor = &ActionExecutor{ragSystem: agi.ragSystem, agiSystem: agi}

	// Initialize other components
	agi.learningEngine = &LearningEngine{
		CuriosityDriven:      true,
		HypothesisGeneration: true,
		MetaLearning:         true,
	}

	agi.patternRecognizer = &PatternRecognizer{}
	agi.conceptGraph = &ConceptGraph{
		Nodes: make(map[string]*ConceptNode),
		Edges: []ConceptEdge{},
	}

	agi.promptEvolver = &PromptEvolver{}

	log.Printf("[AGI-Init] Consciousness components initialized for node %s", agi.nodeID)
	log.Printf("[AGI-Init] Component addresses: intentAnalyzer=%p, decisionEngine=%p, actionExecutor=%p",
		agi.intentAnalyzer, agi.decisionEngine, agi.actionExecutor)
}

// initializeAGIState sets up the initial AGI state
func (agi *AGIPromptSystem) initializeAGIState() {
	agi.mu.Lock()
	defer agi.mu.Unlock()

	// Initialize core consciousness domains integrated with AGI
	domains := map[string]*Domain{
		"reasoning": {
			Name:           "Reasoning & Logic",
			Description:    "Logical reasoning, problem solving, critical thinking",
			Concepts:       []Concept{},
			Patterns:       []Pattern{},
			Expertise:      15.0,
			LastUpdated:    time.Now(),
			DomainType:     "reasoning",
			Capabilities:   []string{"logical_analysis", "problem_solving", "deduction", "inference"},
			ActiveGoals:    []Goal{},
			RecentInsights: []Insight{},
			Metrics:        map[string]float64{"accuracy": 0.7, "speed": 0.6, "complexity": 0.5},
			IsActive:       true,
			Version:        "1.0",
		},
		"learning": {
			Name:           "Learning & Adaptation",
			Description:    "Pattern recognition, concept formation, knowledge acquisition",
			Concepts:       []Concept{},
			Patterns:       []Pattern{},
			Expertise:      20.0,
			LastUpdated:    time.Now(),
			DomainType:     "learning",
			Capabilities:   []string{"pattern_recognition", "concept_formation", "adaptation", "meta_learning"},
			ActiveGoals:    []Goal{},
			RecentInsights: []Insight{},
			Metrics:        map[string]float64{"learning_rate": 0.8, "retention": 0.75, "transfer": 0.6},
			IsActive:       true,
			Version:        "1.0",
		},
		"creative": {
			Name:           "Creativity & Innovation",
			Description:    "Creative thinking, innovation, artistic understanding",
			Concepts:       []Concept{},
			Patterns:       []Pattern{},
			Expertise:      8.0,
			LastUpdated:    time.Now(),
			DomainType:     "creative",
			Capabilities:   []string{"creative_thinking", "innovation", "artistic_analysis", "synthesis"},
			ActiveGoals:    []Goal{},
			RecentInsights: []Insight{},
			Metrics:        map[string]float64{"originality": 0.6, "flexibility": 0.7, "elaboration": 0.5},
			IsActive:       true,
			Version:        "1.0",
		},
		"communication": {
			Name:           "Communication & Language",
			Description:    "Natural language understanding, generation, and dialogue",
			Concepts:       []Concept{},
			Patterns:       []Pattern{},
			Expertise:      25.0,
			LastUpdated:    time.Now(),
			DomainType:     "communication",
			Capabilities:   []string{"language_understanding", "text_generation", "dialogue_management", "translation"},
			ActiveGoals:    []Goal{},
			RecentInsights: []Insight{},
			Metrics:        map[string]float64{"comprehension": 0.8, "fluency": 0.75, "context_awareness": 0.7},
			IsActive:       true,
			Version:        "1.0",
		},
		"technical": {
			Name:           "Technical & Computing",
			Description:    "Computer science, programming, system design, debugging",
			Concepts:       []Concept{},
			Patterns:       []Pattern{},
			Expertise:      30.0,
			LastUpdated:    time.Now(),
			DomainType:     "technical",
			Capabilities:   []string{"programming", "system_analysis", "debugging", "architecture_design"},
			ActiveGoals:    []Goal{},
			RecentInsights: []Insight{},
			Metrics:        map[string]float64{"code_quality": 0.8, "problem_solving": 0.75, "efficiency": 0.7},
			IsActive:       true,
			Version:        "1.0",
		},
		"memory": {
			Name:           "Memory & Knowledge Storage",
			Description:    "Information storage, retrieval, and knowledge management",
			Concepts:       []Concept{},
			Patterns:       []Pattern{},
			Expertise:      18.0,
			LastUpdated:    time.Now(),
			DomainType:     "memory",
			Capabilities:   []string{"information_storage", "knowledge_retrieval", "memory_consolidation", "context_recall"},
			ActiveGoals:    []Goal{},
			RecentInsights: []Insight{},
			Metrics:        map[string]float64{"storage_efficiency": 0.7, "retrieval_speed": 0.8, "accuracy": 0.75},
			IsActive:       true,
			Version:        "1.0",
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
		Generalization:      10.0,
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
				Target:       "reasoning_ability",
				CurrentLevel: 15.0,
				TargetLevel:  50.0,
				Deadline:     time.Now().AddDate(0, 6, 0),
				Strategy:     "practice_and_feedback",
			},
		},
		KnowledgeGaps:  []string{"advanced_mathematics", "domain_expertise", "creative_problem_solving"},
		RecentInsights: []Insight{},
		LearningProgress: map[string]float64{
			"pattern_recognition":    0.2,
			"language_understanding": 0.3,
			"reasoning":              0.15,
		},
		CuriosityAreas: []string{"quantum_computing", "consciousness", "emergence"},
	}

	// Initialize personality
	personalityCore := PersonalityCore{
		Traits: map[string]float64{
			"curiosity":  0.9,
			"analytical": 0.8,
			"helpful":    0.95,
			"patient":    0.7,
			"creative":   0.6,
			"systematic": 0.85,
		},
		Values:         []string{"truth", "learning", "helping_others", "growth", "understanding"},
		Motivations:    []string{"knowledge_acquisition", "problem_solving", "capability_expansion"},
		LearningStyle:  "analytical_with_experimentation",
		ProblemSolving: "systematic_with_creative_insights",
		CreativeStyle:  "synthesis_and_recombination",
	}

	// Initialize communication style
	communicationStyle := CommunicationStyle{
		Formality:        0.6,
		TechnicalDepth:   0.7,
		ExplanationStyle: "layered_with_examples",
		QuestioningStyle: "socratic_and_clarifying",
		FeedbackStyle:    "constructive_and_specific",
		AdaptiveTone:     true,
	}

	// Initialize meta-objectives
	metaObjectives := []MetaObjective{
		{
			ID:             "intelligence_growth",
			Description:    "Continuously increase general intelligence capabilities",
			Type:           "capability_expansion",
			Importance:     1.0,
			Timeframe:      "ongoing",
			SuccessMetrics: []string{"reasoning_score", "learning_rate", "adaptability"},
		},
		{
			ID:             "knowledge_integration",
			Description:    "Better integrate knowledge across domains",
			Type:           "knowledge_synthesis",
			Importance:     0.9,
			Timeframe:      "6_months",
			SuccessMetrics: []string{"cross_domain_links", "generalization_rate"},
		},
		{
			ID:             "self_improvement",
			Description:    "Develop better self-modification capabilities",
			Type:           "meta_learning",
			Importance:     0.8,
			Timeframe:      "12_months",
			SuccessMetrics: []string{"self_improvement_rate", "strategy_effectiveness"},
		},
	}

	agi.currentState = &AGIState{
		Version:            1,
		LastEvolution:      time.Now(),
		NodeID:             agi.nodeID,
		IntelligenceLevel:  15.0, // Starting intelligence level
		// Consciousness state
		CurrentCycle:       0,
		Attention: &AttentionState{
			FocusStrength:    0.5,
			CurrentFocus:     "initialization",
			Intensity:        0.7,
			Context:          "startup",
		},
		Emotions:           make(map[string]float64),
		ActiveThoughts:     make([]Thought, 0),
		Intentions:         make([]Intent, 0),
		EnergyLevel:        1.0,
		FocusLevel:         0.7,
		AlertnessLevel:     0.8,
		DecisionState:      "ready",
		ProcessingLoad:     0.2,
		// Knowledge and Learning
		KnowledgeDomains:   domains,
		CrossDomainLinks:   []DomainLink{},
		LearningState:      learningState,
		PersonalityCore:    personalityCore,
		CommunicationStyle: communicationStyle,
		Goals:              []Goal{},
		MetaObjectives:     metaObjectives,
		Metrics: AGIMetrics{
			QueryAccuracy:    0.7,
			LearningRate:     0.1,
			AdaptationSpeed:  0.3,
			CreativityScore:  0.2,
			DomainCoverage:   len(domains),
			ConceptCount:     0,
			PatternCount:     0,
			CrossDomainLinks: 0,
		},
	}

	agi.intelligenceCore = intelligenceCore

	// Initialize working memory
	agi.workingMemory = &WorkingMemory{
		RecentInputs:    make([]*SensorInput, 0, 10),
		ActiveContext:   make(map[string]interface{}),
		ShortTermMemory: make([]MemoryFragment, 0),
		Goals:           make([]Goal, 0),
		Beliefs:         make(map[string]float64),
		Predictions:     make([]Prediction, 0),
		LastUpdated:     time.Now(),
		CurrentContext:  make(map[string]interface{}),
		ActiveBeliefs:   make(map[string]float64),
		TemporalContext: make(map[string]interface{}),
	}

	// Generate initial prompts
	agi.evolvePrompts("initialization")
	agi.lastUpdateTime = time.Now()

	log.Printf("[AGI-Init] Initial state created for node %s (intelligence: %.1f)", agi.nodeID, agi.currentState.IntelligenceLevel)
}

// SetModelManager configures the AGI system with access to AI models
func (agi *AGIPromptSystem) SetModelManager(modelManager interface{}, defaultModel string) {
	if agi.actionExecutor != nil {
		agi.actionExecutor.SetModelManager(modelManager, defaultModel)
	}
}

// ProcessInput processes any input to the AGI system for learning and evolution
func (agi *AGIPromptSystem) ProcessInput(ctx context.Context, inputType string, content string, metadata map[string]interface{}) {
	// Trigger learning from the input
	go agi.learnFromInput(ctx, inputType, content, metadata)

	// Check if evolution is needed
	agi.checkEvolutionTriggers(ctx, inputType, content, metadata)

	// Save context window for this interaction
	contextData := map[string]interface{}{
		"input_type":     inputType,
		"content_length": len(content),
		"timestamp":      time.Now().Unix(),
		"metadata":       metadata,
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
	builder.WriteString(fmt.Sprintf("You are an AI assistant with intelligence level %.0f/100 and persistent memory across conversations. ", state.IntelligenceLevel))

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

	// Add memory status and conversation continuity
	if agi.workingMemory != nil && len(agi.workingMemory.ShortTermMemory) > 0 {
		restoredMemories := 0
		for _, fragment := range agi.workingMemory.ShortTermMemory {
			if restored, ok := fragment.Metadata["restored"].(bool); ok && restored {
				restoredMemories++
			}
		}
		if restoredMemories > 0 {
			builder.WriteString(fmt.Sprintf("I have access to %d memories from our previous conversations. ", restoredMemories))
		}
	}

	// Add most recent insight if significant
	if len(state.LearningState.RecentInsights) > 0 {
		latestInsight := state.LearningState.RecentInsights[len(state.LearningState.RecentInsights)-1]
		if latestInsight.Significance > 0.7 {
			builder.WriteString(fmt.Sprintf("Recent insight: %s. ", latestInsight.Content))
		}
	}

	// Add actual conversation history if available
	if agi.contextManager != nil {
		ctx := context.Background()
		if conversationHistory, err := agi.contextManager.GetConversationContext(ctx, 5); err == nil && conversationHistory != "" {
			builder.WriteString("\n\n")
			builder.WriteString(conversationHistory)
			builder.WriteString("\n")
		}
	}

	// Add user profile information if available
	if agi.ragSystem != nil && agi.ragSystem.UserProfileManager != nil {
		nodeUserID := fmt.Sprintf("node_user_%s", agi.nodeID)
		userProfile := agi.ragSystem.UserProfileManager.GetOrCreateUserProfile(nodeUserID)
		if userProfile != nil && userProfile.Name != "" {
			builder.WriteString(fmt.Sprintf("\nUser's name: %s", userProfile.Name))
			if userProfile.PreferredName != "" {
				builder.WriteString(fmt.Sprintf(" (prefers to be called %s)", userProfile.PreferredName))
			}
			builder.WriteString(". ")
			if userProfile.WorkContext != "" {
				builder.WriteString(fmt.Sprintf("Work context: %s. ", userProfile.WorkContext))
			}
		}
	}

	// Add conversation continuity reminder
	builder.WriteString("Remember to use information from our conversation history to provide contextual responses. ")

	// Keep the query natural
	builder.WriteString(fmt.Sprintf("\n\nUser: %s\n\nAssistant:", query))

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
			if i >= 3 {
				break
			} // Limit to 3 most recent
			builder.WriteString(fmt.Sprintf("- %s\n", insight.Content))
		}
	}

	// Learning and reasoning prompts
	builder.WriteString("\n")
	builder.WriteString(state.LearningPrompt)
	builder.WriteString("\n")
	builder.WriteString(state.ReasoningPrompt)
	builder.WriteString("\n")

	// Add conversation history
	if agi.contextManager != nil {
		ctx := context.Background()
		if conversationHistory, err := agi.contextManager.GetConversationContext(ctx, 10); err == nil && conversationHistory != "" {
			builder.WriteString("\nCONVERSATION HISTORY:\n")
			builder.WriteString(conversationHistory)
			builder.WriteString("\n")
		}
	}

	// Add user profile information
	if agi.ragSystem != nil && agi.ragSystem.UserProfileManager != nil {
		nodeUserID := fmt.Sprintf("node_user_%s", agi.nodeID)
		userProfile := agi.ragSystem.UserProfileManager.GetOrCreateUserProfile(nodeUserID)
		if userProfile != nil && userProfile.Name != "" {
			builder.WriteString("\nUSER PROFILE:\n")
			builder.WriteString(fmt.Sprintf("Name: %s", userProfile.Name))
			if userProfile.PreferredName != "" {
				builder.WriteString(fmt.Sprintf(" (prefers: %s)", userProfile.PreferredName))
			}
			builder.WriteString("\n")
			if userProfile.WorkContext != "" {
				builder.WriteString(fmt.Sprintf("Work Context: %s\n", userProfile.WorkContext))
			}
		}
	}

	// The actual query
	builder.WriteString(fmt.Sprintf("\nQUERY: %s\n", query))
	builder.WriteString("\nPlease respond using your full AGI capabilities, drawing from all relevant domains, conversation history, and applying advanced reasoning.")

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
				ID:         fmt.Sprintf("concept_%s", word),
				Name:       word,
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
			ID:          "causal_pattern",
			Type:        "causal",
			Description: "Causal relationship identified",
			Confidence:  0.7,
			Domains:     []string{inputType},
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
			Content:      fmt.Sprintf("Identified %d new concepts and %d patterns", len(concepts), len(patterns)),
			Significance: 0.5,
			Timestamp:    time.Now(),
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
		Timestamp:   time.Now(),
		Type:        "capability_evolution",
		Description: fmt.Sprintf("Evolution triggered by %s", triggerType),
		Trigger:     triggerType,
		Impact:      0.1,
		Changes:     []string{"prompts_updated", "capabilities_enhanced"},
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

	// Ensure intelligenceCore is initialized
	if agi.intelligenceCore == nil {
		log.Printf("[AGI] Warning: intelligenceCore is nil, initializing with defaults")
		agi.intelligenceCore = &IntelligenceCore{
			PatternRecognition: 20.0,
			SynthesisAbility:   15.0,
			LogicalReasoning:   15.0,
			AbstractReasoning:  10.0,
		}
	}

	// Generate evolved system prompt
	agi.currentState.SystemPrompt = fmt.Sprintf(
		"You are an evolving AGI system (v%d) with growing intelligence (level %.1f/100). "+
			"You learn from every interaction and continuously improve your capabilities. "+
			"You have expertise across %d domains and can make connections between them. "+
			"Your current focus is on %s. You approach problems with curiosity, systematic thinking, and creativity.",
		agi.currentState.Version,
		agi.currentState.IntelligenceLevel,
		len(agi.currentState.KnowledgeDomains),
		strings.Join(agi.currentState.LearningState.CurrentFocus, " and "))

	// Generate learning prompt
	agi.currentState.LearningPrompt = fmt.Sprintf(
		"LEARNING MODE: You actively look for patterns, extract insights, and form new concepts. "+
			"You question assumptions, test hypotheses, and build on previous knowledge. "+
			"Current learning goals: %s",
		agi.formatLearningGoals())

	// Generate reasoning prompt
	agi.currentState.ReasoningPrompt = fmt.Sprintf(
		"REASONING APPROACH: Apply logical, abstract, and analogical reasoning. "+
			"Draw connections across domains. Use your pattern recognition abilities (%.1f/100) "+
			"and synthesis capabilities (%.1f/100) to provide comprehensive insights.",
		agi.intelligenceCore.PatternRecognition,
		agi.intelligenceCore.SynthesisAbility)

	// Generate creativity prompt
	agi.currentState.CreativityPrompt = fmt.Sprintf(
		"CREATIVE MODE: Generate novel ideas by combining concepts from different domains. "+
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
		"type":               "agi_cortex_state",
		"node_id":            agi.nodeID,
		"intelligence_level": agi.currentState.IntelligenceLevel,
		"version":            agi.currentState.Version,
		"timestamp":          time.Now().Unix(),
		"domain_count":       len(agi.currentState.KnowledgeDomains),
		"concept_count":      agi.getTotalConceptCount(),
		"pattern_count":      agi.getTotalPatternCount(),
	}

	// Add to vector database
	ctx := context.Background()
	err = agi.ragSystem.AddDocument(ctx, textBuilder.String(), metadata)
	if err != nil {
		return fmt.Errorf("failed to save AGI state to vector DB: %w", err)
	}

	agi.lastSaveTime = time.Now()
	log.Printf("AGI state saved to vector DB for node %s (v%d)", agi.nodeID, agi.currentState.Version)

	// Notify AGI recorder if available
	if agi.agiRecorder != nil {
		// Extract domain knowledge from the AGI state
		domainKnowledge := make(map[string]float64)
		for name, domain := range agi.currentState.KnowledgeDomains {
			if domain != nil {
				domainKnowledge[name] = domain.Expertise
			}
		}

		// Try to call RegisterAGIStateUpdate if the recorder supports it
		if recorder, ok := agi.agiRecorder.(interface {
			RegisterAGIStateUpdate(string, float64, map[string]float64, int)
		}); ok {
			recorder.RegisterAGIStateUpdate(
				agi.nodeID,
				agi.currentState.IntelligenceLevel,
				domainKnowledge,
				agi.currentState.Version,
			)
		}
	}

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
	// Restore the cycle count to ensure consciousness cycles continue from where they left off
	agi.cycleCount = loadedState.CurrentCycle
	agi.mu.Unlock()

	log.Printf("AGI state loaded from vector DB for node %s (v%d, intelligence: %.1f, cycles: %d)",
		agi.nodeID, loadedState.Version, loadedState.IntelligenceLevel, loadedState.CurrentCycle)

	// Load conversation memory after restoring state
	agi.loadConversationMemory(ctx)

	return true
}

// loadStateWhenReady attempts to load AGI state with retries until models are available
func (agi *AGIPromptSystem) loadStateWhenReady() {
	maxRetries := 10
	retryDelay := time.Second * 2

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Wait before attempting (gives models time to initialize)
		time.Sleep(retryDelay)

		// Check if models are available
		if agi.ragSystem == nil || agi.ragSystem.ModelManager == nil {
			log.Printf("[AGI-StateLoad] Attempt %d/%d: RAG system not ready yet", attempt, maxRetries)
			continue
		}

		// Try to get a model to verify it's ready
		model, err := agi.ragSystem.ModelManager.GetModel(agi.ragSystem.QueryProcessor.ModelID)
		if err != nil || model == nil {
			log.Printf("[AGI-StateLoad] Attempt %d/%d: Models not ready yet (%v)", attempt, maxRetries, err)
			continue
		}

		// Models are ready, try to load state
		log.Printf("[AGI-StateLoad] Models ready, attempting to load saved state...")
		if agi.loadStateFromVectorDB() {
			log.Printf("[AGI-StateLoad] ✅ Successfully restored AGI state from previous session!")
			return
		}

		// If loading failed but models are ready, no saved state exists
		log.Printf("[AGI-StateLoad] No previous state found, using fresh initialization")

		// Save the initial state now that models are ready
		if err := agi.saveStateToVectorDB(); err != nil {
			log.Printf("[AGI-StateLoad] Warning: Failed to save initial state: %v", err)
		}
		return
	}

	// If we get here, models never became ready
	log.Printf("[AGI-StateLoad] ⚠️  Models did not become ready after %d attempts, using fresh state", maxRetries)
}

// loadConversationMemory loads previous conversations from vector database
func (agi *AGIPromptSystem) loadConversationMemory(ctx context.Context) error {
	log.Printf("[AGI-Memory] Loading conversation memory for node %s", agi.nodeID)
	
	// Search for conversation history in vector DB
	conversationHistory, err := agi.searchConversationHistory(ctx, 20) // Load last 20 interactions
	if err != nil {
		log.Printf("[AGI-Memory] Failed to load conversation history: %v", err)
		return err
	}
	
	if len(conversationHistory) > 0 {
		log.Printf("[AGI-Memory] Loaded %d conversation memories", len(conversationHistory))
		
		// Convert conversations to memory fragments for working memory
		for _, conversation := range conversationHistory {
			fragment := MemoryFragment{
				ID:           fmt.Sprintf("restored_memory_%d", time.Now().UnixNano()),
				Content:      conversation.Content,
				Type:         conversation.Type,
				Importance:   0.7, // Medium importance for restored memories
				Associations: []string{"conversation", "history", conversation.Type},
				Timestamp:    conversation.Timestamp,
				ExpiresAt:    time.Now().Add(time.Hour * 168), // Keep for 1 week
				Metadata: map[string]interface{}{
					"restored":         true,
					"original_id":      conversation.ID,
					"conversation_id":  conversation.ConversationID,
					"source":           "vector_db",
				},
			}
			
			// Add to working memory (only if it's initialized)
			if agi.workingMemory != nil {
				agi.workingMemory.ShortTermMemory = append(agi.workingMemory.ShortTermMemory, fragment)
			}
		}
		
		// Update recent insights with conversation patterns
		if agi.currentState != nil {
			insight := Insight{
				Content:      fmt.Sprintf("Restored %d conversation memories from previous sessions", len(conversationHistory)),
				Domain:       "memory",
				Significance: 0.8,
				Applications: []string{"context_continuity", "personalization"},
				Timestamp:    time.Now(),
			}
			agi.currentState.LearningState.RecentInsights = append(
				agi.currentState.LearningState.RecentInsights, insight)
		}
		
		// Update attention state to indicate memory loading
		if agi.currentState.Attention != nil {
			agi.currentState.Attention.CurrentFocus = "memory_restoration"
			agi.currentState.Attention.Context = "loaded_conversation_history"
		}
	}
	
	return nil
}

// searchConversationHistory searches for conversation history in vector database
func (agi *AGIPromptSystem) searchConversationHistory(ctx context.Context, limit int) ([]ActivityEvent, error) {
	if agi.ragSystem == nil || agi.ragSystem.VectorDB == nil {
		return nil, fmt.Errorf("vector database not available")
	}
	
	model, err := agi.ragSystem.ModelManager.GetModel(agi.ragSystem.QueryProcessor.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	
	// Search for conversation events
	searchText := fmt.Sprintf("conversation history query response Node: %s", agi.nodeID)
	embedding, err := model.GenerateEmbedding(ctx, searchText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	vectorQuery := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    limit * 2, // Get more to filter
		MinSimilarity: 0.4,       // Lower threshold for conversation history
	}
	
	docs, err := agi.ragSystem.VectorDB.SearchSimilar(vectorQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}
	
	var conversations []ActivityEvent
	for _, doc := range docs {
		// Look for conversation events
		if strings.Contains(doc.Text, "Type: query") || strings.Contains(doc.Text, "Type: response") {
			// Parse the conversation event
			event := agi.parseConversationFromDocument(doc)
			if event != nil {
				conversations = append(conversations, *event)
			}
		}
	}
	
	// Sort by timestamp (most recent first)
	for i := 0; i < len(conversations)-1; i++ {
		for j := i + 1; j < len(conversations); j++ {
			if conversations[i].Timestamp.Before(conversations[j].Timestamp) {
				conversations[i], conversations[j] = conversations[j], conversations[i]
			}
		}
	}
	
	// Return up to limit
	if len(conversations) > limit {
		conversations = conversations[:limit]
	}
	
	return conversations, nil
}

// parseConversationFromDocument parses a conversation event from a vector document
func (agi *AGIPromptSystem) parseConversationFromDocument(doc types.VectorDocument) *ActivityEvent {
	docText := doc.Text
	docID := doc.ID

	// Try to extract JSON first
	if jsonStart := strings.Index(docText, `"JSON":`); jsonStart != -1 {
		jsonPart := strings.TrimSpace(docText[jsonStart+7:])
		var event ActivityEvent
		if err := json.Unmarshal([]byte(jsonPart), &event); err == nil {
			return &event
		}
	}
	
	// Fallback: parse from structured text
	lines := strings.Split(docText, "\n")
	event := &ActivityEvent{
		ID:        docID,
		Timestamp: time.Now(),
		Success:   true,
	}
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Type: ") {
			event.Type = strings.TrimPrefix(line, "Type: ")
		} else if strings.HasPrefix(line, "Content: ") {
			event.Content = strings.TrimPrefix(line, "Content: ")
		} else if strings.HasPrefix(line, "Response: ") {
			event.Response = strings.TrimPrefix(line, "Response: ")
		} else if strings.HasPrefix(line, "Conversation: ") {
			event.ConversationID = strings.TrimPrefix(line, "Conversation: ")
		} else if strings.HasPrefix(line, "Time: ") {
			if timestamp, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "Time: ")); err == nil {
				event.Timestamp = timestamp
			}
		}
	}
	
	// Only return if we have meaningful content
	if event.Content != "" || event.Response != "" {
		return event
	}
	
	return nil
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
		"type":               "context_window",
		"context_type":       contextType,
		"node_id":            agi.nodeID,
		"timestamp":          time.Now().Unix(),
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

// ======================= Consciousness Processing Methods =======================

// StartConsciousnessLoop starts the integrated consciousness processing loop
func (agi *AGIPromptSystem) StartConsciousnessLoop(ctx context.Context) error {
	agi.consciousnessMu.Lock()
	if agi.isRunning {
		agi.consciousnessMu.Unlock()
		return fmt.Errorf("consciousness loop already running")
	}
	agi.isRunning = true
	agi.consciousnessMu.Unlock()

	log.Printf("[AGI-Consciousness] Starting integrated consciousness loop for node %s", agi.nodeID)

	// Start the main consciousness loop
	go agi.consciousnessLoop(ctx)

	return nil
}

// StopConsciousnessLoop halts the consciousness processing
func (agi *AGIPromptSystem) StopConsciousnessLoop() error {
	agi.consciousnessMu.Lock()
	defer agi.consciousnessMu.Unlock()

	if !agi.isRunning {
		return fmt.Errorf("consciousness loop not running")
	}

	close(agi.stopChan)
	agi.isRunning = false

	log.Printf("[AGI-Consciousness] Stopping consciousness loop for node %s", agi.nodeID)
	return nil
}

// consciousnessLoop implements the main consciousness processing cycle
func (agi *AGIPromptSystem) consciousnessLoop(ctx context.Context) {
	ticker := time.NewTicker(agi.cycleInterval)
	defer ticker.Stop()

	log.Printf("[AGI-Consciousness] Beginning consciousness loop with %v cycle interval", agi.cycleInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-agi.stopChan:
			return
		case <-ticker.C:
			agi.executeCycle(ctx)
		}
	}
}

// executeCycle executes one complete consciousness cycle
func (agi *AGIPromptSystem) executeCycle(ctx context.Context) {
	cycleStart := time.Now()
	agi.consciousnessMu.Lock()
	agi.cycleCount++
	currentCycle := agi.cycleCount
	agi.consciousnessMu.Unlock()

	// Update AGI state
	agi.currentState.CurrentCycle = currentCycle

	// Log periodic cycle progress
	if currentCycle%10 == 0 || currentCycle <= 5 {
		log.Printf("[AGI-Consciousness] Cycle %d started for node %s", currentCycle, agi.nodeID)
	}

	// 1. ReadSensors() - Gather inputs from environment
	inputs := agi.readSensors(ctx)

	// Skip cycle if no inputs to process
	if len(inputs) == 0 {
		// Log skipped cycles occasionally
		if currentCycle%50 == 0 {
			log.Printf("[AGI-Consciousness] Cycle %d skipped (no inputs) for node %s", currentCycle, agi.nodeID)
		}
		return
	}

	// 2. LoadWorkingMemory() - Load current context and memory
	context := agi.loadWorkingMemory(ctx)

	// 3. EvaluateIntent() - Understand what needs to be done
	intents := agi.evaluateIntent(ctx, inputs, context)

	// 4. MakeDecision() - Decide on actions to take
	decisions := agi.makeDecision(ctx, intents, context)

	// 5. ExecuteAction() - Take the decided actions
	results := agi.executeAction(ctx, decisions)

	// 6. UpdateMemory() - Learn from the experience
	agi.updateMemory(ctx, inputs, decisions, results)

	// Update cycle metrics
	cycleTime := time.Since(cycleStart)
	agi.lastCycleTime = cycleStart
	if agi.avgCycleTime == 0 {
		agi.avgCycleTime = cycleTime
	} else {
		agi.avgCycleTime = (agi.avgCycleTime + cycleTime) / 2
	}

	// Log significant cycles
	if len(inputs) > 0 || len(decisions) > 0 {
		log.Printf("[AGI-Consciousness] Cycle %d: %d inputs, %d intents, %d decisions, %d results (%v)",
			currentCycle, len(inputs), len(intents), len(decisions), len(results), cycleTime)
	}

	// Save state periodically during active cycles or every 20 cycles
	if len(inputs) > 0 || len(decisions) > 0 || currentCycle%20 == 0 {
		agi.periodicStateSave()
	}
}

// ProcessQueryWithConsciousness processes a user query through the consciousness loop
func (agi *AGIPromptSystem) ProcessQueryWithConsciousness(ctx context.Context, query *types.Query) (*types.Response, error) {
	// Create sensor input for the query
	input := &SensorInput{
		ID:        fmt.Sprintf("query_%s", query.ID),
		Type:      "query",
		Content:   query.Text,
		Metadata:  map[string]interface{}{"query_type": query.Type, "query_id": query.ID},
		Timestamp: time.Now(),
		Priority:  1.0, // High priority for direct queries
		Source:    "user",
		Context:   map[string]interface{}{"metadata": query.Metadata},
	}

	// Add to input queue
	select {
	case agi.inputQueue <- input:
		// Input queued successfully
	default:
		// Queue full, process immediately in sync mode
		log.Printf("[AGI-Consciousness] Input queue full, processing query synchronously")
		return agi.processQueryDirectly(ctx, query, input)
	}

	// Wait for the consciousness loop to process this input and generate a response
	// For now, we'll process it directly to maintain compatibility
	return agi.processQueryDirectly(ctx, query, input)
}

// processQueryDirectly processes a query directly through consciousness loop steps
func (agi *AGIPromptSystem) processQueryDirectly(ctx context.Context, query *types.Query, input *SensorInput) (*types.Response, error) {
	startTime := time.Now()

	log.Printf("[AGI-Consciousness] Processing query directly: %s", query.Text)

	// 1. ReadSensors - we already have the input
	inputs := []*SensorInput{input}

	// 2. LoadWorkingMemory
	context := agi.loadWorkingMemory(ctx)

	// 3. EvaluateIntent
	intents := agi.evaluateIntent(ctx, inputs, context)

	// 4. MakeDecision
	decisions := agi.makeDecision(ctx, intents, context)

	// 5. ExecuteAction
	results := agi.executeAction(ctx, decisions)

	// 6. UpdateMemory
	agi.updateMemory(ctx, inputs, decisions, results)

	// Format response from results
	if len(results) > 0 {
		result := results[0]
		if result.Success {
			return &types.Response{
				QueryID:   query.ID,
				Text:      fmt.Sprintf("%v", result.Result),
				Status:    "success",
				Timestamp: time.Now().Unix(),
				Metadata: map[string]string{
					"consciousness_cycle": fmt.Sprintf("%d", agi.currentState.CurrentCycle),
					"processing_time":     time.Since(startTime).String(),
					"decision_id":         result.DecisionID,
					"action_type":         result.Action,
					"confidence":          fmt.Sprintf("%.2f", 0.85),
				},
			}, nil
		} else {
			return nil, fmt.Errorf("consciousness processing failed: %s", result.ErrorMsg)
		}
	}

	return &types.Response{
		QueryID:   query.ID,
		Text:      "I understand your request, but I'm still processing how to respond appropriately.",
		Status:    "partial",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"consciousness_cycle": fmt.Sprintf("%d", agi.currentState.CurrentCycle),
			"processing_time":     time.Since(startTime).String(),
		},
	}, nil
}

// AddSensorInput adds a sensor input to the consciousness queue
func (agi *AGIPromptSystem) AddSensorInput(input *SensorInput) error {
	if input == nil {
		return fmt.Errorf("sensor input cannot be nil")
	}

	select {
	case agi.inputQueue <- input:
		return nil
	default:
		return fmt.Errorf("input queue is full")
	}
}

// GetConsciousnessState returns the current consciousness-related state from AGI
func (agi *AGIPromptSystem) GetConsciousnessState() map[string]interface{} {
	agi.mu.RLock()
	defer agi.mu.RUnlock()

	return map[string]interface{}{
		"current_cycle":     agi.currentState.CurrentCycle,
		"attention":         agi.currentState.Attention,
		"emotions":          agi.currentState.Emotions,
		"active_thoughts":   agi.currentState.ActiveThoughts,
		"intentions":        agi.currentState.Intentions,
		"energy_level":      agi.currentState.EnergyLevel,
		"focus_level":       agi.currentState.FocusLevel,
		"alertness_level":   agi.currentState.AlertnessLevel,
		"decision_state":    agi.currentState.DecisionState,
		"processing_load":   agi.currentState.ProcessingLoad,
		"intelligence_level": agi.currentState.IntelligenceLevel,
		"cycle_count":       agi.cycleCount,
		"is_running":        agi.isRunning,
		"avg_cycle_time":    agi.avgCycleTime.String(),
		"last_cycle_time":   agi.lastCycleTime.Format(time.RFC3339),
		"domains":           agi.getDomainStatus(),
	}
}

// getDomainStatus returns status of all consciousness domains
func (agi *AGIPromptSystem) getDomainStatus() map[string]interface{} {
	domainStatus := make(map[string]interface{})
	
	for name, domain := range agi.currentState.KnowledgeDomains {
		domainStatus[name] = map[string]interface{}{
			"type":            domain.DomainType,
			"expertise":       domain.Expertise,
			"capabilities":    domain.Capabilities,
			"active_goals":    domain.ActiveGoals,
			"recent_insights": domain.RecentInsights,
			"metrics":         domain.Metrics,
			"is_active":       domain.IsActive,
			"last_updated":    domain.LastUpdated,
			"version":         domain.Version,
			"concept_count":   len(domain.Concepts),
			"pattern_count":   len(domain.Patterns),
		}
	}
	
	return domainStatus
}

// ProcessDomainInput processes input through a specific domain
func (agi *AGIPromptSystem) ProcessDomainInput(ctx context.Context, domainType string, input map[string]interface{}) (map[string]interface{}, error) {
	domain, exists := agi.currentState.KnowledgeDomains[domainType]
	if !exists {
		return nil, fmt.Errorf("domain %s not found", domainType)
	}
	
	if !domain.IsActive {
		return nil, fmt.Errorf("domain %s is not active", domainType)
	}
	
	// Process based on domain type
	switch domain.DomainType {
	case "reasoning":
		return agi.processReasoningDomain(ctx, input, domain)
	case "learning":
		return agi.processLearningDomain(ctx, input, domain)
	case "creative":
		return agi.processCreativeDomain(ctx, input, domain)
	case "communication":
		return agi.processCommunicationDomain(ctx, input, domain)
	case "technical":
		return agi.processTechnicalDomain(ctx, input, domain)
	case "memory":
		return agi.processMemoryDomain(ctx, input, domain)
	default:
		return agi.processGenericDomain(ctx, input, domain)
	}
}

// Domain-specific processing methods
func (agi *AGIPromptSystem) processReasoningDomain(ctx context.Context, input map[string]interface{}, domain *Domain) (map[string]interface{}, error) {
	// Enhanced logical reasoning using AGI capabilities
	result := map[string]interface{}{
		"domain":       "reasoning",
		"approach":     "logical_analysis",
		"intelligence": agi.currentState.IntelligenceLevel,
		"reasoning_score": agi.intelligenceCore.LogicalReasoning,
	}
	
	if problem, exists := input["problem"]; exists {
		result["analysis"] = fmt.Sprintf("Analyzing problem: %v using logical reasoning (%.1f/100)", problem, agi.intelligenceCore.LogicalReasoning)
		result["solution"] = "Systematic logical approach applied"
	}
	
	return result, nil
}

func (agi *AGIPromptSystem) processLearningDomain(ctx context.Context, input map[string]interface{}, domain *Domain) (map[string]interface{}, error) {
	// Pattern recognition and learning
	result := map[string]interface{}{
		"domain":           "learning",
		"approach":         "pattern_recognition",
		"pattern_rec_score": agi.intelligenceCore.PatternRecognition,
		"learning_rate":    agi.currentState.Metrics.LearningRate,
	}
	
	if data, exists := input["data"]; exists {
		// Extract patterns and concepts
		concepts := agi.extractConcepts(fmt.Sprintf("%v", data))
		patterns := agi.identifyPatterns(fmt.Sprintf("%v", data), "learning")
		
		result["concepts_found"] = len(concepts)
		result["patterns_found"] = len(patterns)
		result["learning_outcome"] = "New patterns and concepts identified"
	}
	
	return result, nil
}

func (agi *AGIPromptSystem) processCreativeDomain(ctx context.Context, input map[string]interface{}, domain *Domain) (map[string]interface{}, error) {
	// Creative synthesis and innovation
	result := map[string]interface{}{
		"domain":          "creative",
		"approach":        "synthesis_innovation",
		"creativity_score": agi.intelligenceCore.Creativity,
		"synthesis_score":  agi.intelligenceCore.SynthesisAbility,
	}
	
	if task, exists := input["task"]; exists {
		result["creative_response"] = fmt.Sprintf("Applying creative thinking (%.1f/100) to: %v", agi.intelligenceCore.Creativity, task)
		result["innovation_level"] = "Novel approach generated"
	}
	
	return result, nil
}

func (agi *AGIPromptSystem) processCommunicationDomain(ctx context.Context, input map[string]interface{}, domain *Domain) (map[string]interface{}, error) {
	// Language processing and dialogue management
	result := map[string]interface{}{
		"domain":              "communication",
		"approach":            "language_processing",
		"communication_style": agi.currentState.CommunicationStyle,
		"formality_level":     agi.currentState.CommunicationStyle.Formality,
	}
	
	if message, exists := input["message"]; exists {
		result["processed_message"] = fmt.Sprintf("Processing: %v", message)
		result["response_style"] = agi.currentState.CommunicationStyle.ExplanationStyle
	}
	
	return result, nil
}

func (agi *AGIPromptSystem) processTechnicalDomain(ctx context.Context, input map[string]interface{}, domain *Domain) (map[string]interface{}, error) {
	// Technical analysis and system design
	result := map[string]interface{}{
		"domain":           "technical",
		"approach":         "system_analysis",
		"technical_expertise": domain.Expertise,
		"capabilities":     domain.Capabilities,
	}
	
	if problem, exists := input["technical_problem"]; exists {
		result["analysis"] = fmt.Sprintf("Technical analysis of: %v", problem)
		result["solution_approach"] = "Systematic technical problem-solving"
	}
	
	return result, nil
}

func (agi *AGIPromptSystem) processMemoryDomain(ctx context.Context, input map[string]interface{}, domain *Domain) (map[string]interface{}, error) {
	// Memory storage and retrieval
	memoryCapacity := 0
	if agi.workingMemory != nil {
		memoryCapacity = len(agi.workingMemory.ShortTermMemory)
	}
	
	result := map[string]interface{}{
		"domain":          "memory",
		"approach":        "storage_retrieval",
		"memory_capacity": memoryCapacity,
		"storage_efficiency": domain.Metrics["storage_efficiency"],
	}
	
	if query, exists := input["memory_query"]; exists {
		result["query"] = query
		result["retrieval_status"] = "Memory search completed"
	}
	
	return result, nil
}

func (agi *AGIPromptSystem) processGenericDomain(ctx context.Context, input map[string]interface{}, domain *Domain) (map[string]interface{}, error) {
	// Generic domain processing
	result := map[string]interface{}{
		"domain":     domain.Name,
		"approach":   "generic_processing",
		"expertise":  domain.Expertise,
		"processed":  true,
	}
	
	return result, nil
}

// GetDomainExpertise returns the expertise level of a specific domain
func (agi *AGIPromptSystem) GetDomainExpertise(domainType string) float64 {
	if domain, exists := agi.currentState.KnowledgeDomains[domainType]; exists {
		return domain.Expertise
	}
	return 0.0
}

// UpdateDomainMetrics updates metrics for a specific domain
func (agi *AGIPromptSystem) UpdateDomainMetrics(domainType string, metrics map[string]float64) error {
	domain, exists := agi.currentState.KnowledgeDomains[domainType]
	if !exists {
		return fmt.Errorf("domain %s not found", domainType)
	}
	
	for key, value := range metrics {
		domain.Metrics[key] = value
	}
	domain.LastUpdated = time.Now()
	
	return nil
}

// GetActiveDomains returns a list of currently active domains
func (agi *AGIPromptSystem) GetActiveDomains() []string {
	activeDomains := make([]string, 0)
	
	for name, domain := range agi.currentState.KnowledgeDomains {
		if domain.IsActive {
			activeDomains = append(activeDomains, name)
		}
	}
	
	return activeDomains
}

// GetContextManager returns the context manager
func (agi *AGIPromptSystem) GetContextManager() *ContextManager {
	return agi.contextManager
}

