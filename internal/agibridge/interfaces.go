package agibridge

import (
	"time"
)

// AGIConnection represents the connection between agents and the AGI system
type AGIConnection interface {
	// Streaming communication
	StreamToAGI(data *AGIStreamData) error
	SubscribeToAGI(handler AGIStreamHandler) error
	UnsubscribeFromAGI() error

	// State synchronization
	GetAGIState() (*AGIState, error)
	UpdateAGIConsciousness(update *ConsciousnessUpdate) error

	// Feedback and learning
	ReportAgentPerformance(metrics *AgentPerformanceReport) error
	ReportAgentInsight(insight *AgentInsight) error

	// Capability negotiation
	RegisterAGICapabilities(capabilities []AGICapability) error
	RequestAGIGuidance(request *AGIGuidanceRequest) (*AGIGuidanceResponse, error)
}

// AGIStreamData represents data streamed between agents and AGI
type AGIStreamData struct {
	Type      AGIStreamType          `json:"type"`
	AgentID   string                 `json:"agent_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
	Priority  int                    `json:"priority"` // 1-10, higher = more important
}

// AGIStreamType defines the types of data that can be streamed
type AGIStreamType string

const (
	AGIStreamTypeQuery             AGIStreamType = "query"
	AGIStreamTypeResponse          AGIStreamType = "response"
	AGIStreamTypeMetrics           AGIStreamType = "metrics"
	AGIStreamTypeInsight           AGIStreamType = "insight"
	AGIStreamTypeError             AGIStreamType = "error"
	AGIStreamTypeStateUpdate       AGIStreamType = "state_update"
	AGIStreamTypeConsciousnessSync AGIStreamType = "consciousness_sync"
	AGIStreamTypeCapabilityUpdate  AGIStreamType = "capability_update"
	AGIStreamTypeGuidance          AGIStreamType = "guidance"
	AGIStreamTypeFeedback          AGIStreamType = "feedback"
)

// AGIStreamHandler handles incoming AGI stream data
type AGIStreamHandler func(data *AGIStreamData) error

// AGIState represents the current state of the AGI system
type AGIState struct {
	IntelligenceLevel  float64                `json:"intelligence_level"`
	ConsciousnessState *ConsciousnessSnapshot `json:"consciousness_state"`
	DomainKnowledge    map[string]float64     `json:"domain_knowledge"`
	ActiveGoals        []string               `json:"active_goals"`
	CurrentAttention   *AttentionFocus        `json:"current_attention"`
	EmotionalState     map[string]float64     `json:"emotional_state"`
	LearningRate       float64                `json:"learning_rate"`
	Timestamp          time.Time              `json:"timestamp"`
}

// ConsciousnessSnapshot represents a snapshot of consciousness state
type ConsciousnessSnapshot struct {
	CycleCount     int64                  `json:"cycle_count"`
	AttentionFocus string                 `json:"attention_focus"`
	ActiveThoughts []string               `json:"active_thoughts"`
	WorkingMemory  map[string]interface{} `json:"working_memory"`
	DecisionState  string                 `json:"decision_state"`
	EnergyLevel    float64                `json:"energy_level"`
	ProcessingLoad float64                `json:"processing_load"`
}

// AttentionFocus represents what the AGI is currently focusing on
type AttentionFocus struct {
	Topic     string        `json:"topic"`
	Intensity float64       `json:"intensity"`
	Duration  time.Duration `json:"duration"`
	Context   string        `json:"context"`
	Agents    []string      `json:"agents"` // Agent IDs related to this focus
}

// ConsciousnessUpdate represents an update to consciousness state from agents
type ConsciousnessUpdate struct {
	AgentID    string                  `json:"agent_id"`
	UpdateType ConsciousnessUpdateType `json:"update_type"`
	Data       map[string]interface{}  `json:"data"`
	Confidence float64                 `json:"confidence"`
	Impact     float64                 `json:"impact"` // Expected impact on consciousness (0-1)
	Timestamp  time.Time               `json:"timestamp"`
}

// ConsciousnessUpdateType defines types of consciousness updates
type ConsciousnessUpdateType string

const (
	ConsciousnessUpdateAttention ConsciousnessUpdateType = "attention"
	ConsciousnessUpdateEmotion   ConsciousnessUpdateType = "emotion"
	ConsciousnessUpdateGoal      ConsciousnessUpdateType = "goal"
	ConsciousnessUpdateMemory    ConsciousnessUpdateType = "memory"
	ConsciousnessUpdateInsight   ConsciousnessUpdateType = "insight"
	ConsciousnessUpdateLearning  ConsciousnessUpdateType = "learning"
	ConsciousnessUpdateEnergy    ConsciousnessUpdateType = "energy"
)

// AgentPerformanceReport represents performance metrics from an agent
type AgentPerformanceReport struct {
	AgentID          string             `json:"agent_id"`
	TaskType         string             `json:"task_type"`
	PerformanceScore float64            `json:"performance_score"`
	ResponseTime     time.Duration      `json:"response_time"`
	SuccessRate      float64            `json:"success_rate"`
	QualityMetrics   map[string]float64 `json:"quality_metrics"`
	LearningPoints   []string           `json:"learning_points"`
	Challenges       []string           `json:"challenges"`
	Timestamp        time.Time          `json:"timestamp"`
}

// AgentInsight represents insights discovered by agents
type AgentInsight struct {
	AgentID     string                 `json:"agent_id"`
	Type        AgentInsightType       `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Confidence  float64                `json:"confidence"`
	Domain      string                 `json:"domain"`
	Impact      InsightImpact          `json:"impact"`
	Timestamp   time.Time              `json:"timestamp"`
}

// AgentInsightType defines types of insights
type AgentInsightType string

const (
	InsightTypePattern      AgentInsightType = "pattern"
	InsightTypeCorrelation  AgentInsightType = "correlation"
	InsightTypeAnomaly      AgentInsightType = "anomaly"
	InsightTypeOptimization AgentInsightType = "optimization"
	InsightTypePrediction   AgentInsightType = "prediction"
	InsightTypeKnowledge    AgentInsightType = "knowledge"
	InsightTypeStrategy     AgentInsightType = "strategy"
)

// InsightImpact represents the potential impact of an insight
type InsightImpact struct {
	Scope      string   `json:"scope"`      // local, network, global
	Magnitude  float64  `json:"magnitude"`  // 0-1 scale
	Confidence float64  `json:"confidence"` // 0-1 scale
	Domains    []string `json:"domains"`    // Affected knowledge domains
}

// AGICapability represents a capability that agents can provide to AGI
type AGICapability struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	InputTypes  []string               `json:"input_types"`
	OutputTypes []string               `json:"output_types"`
	Quality     float64                `json:"quality"`     // 0-1 scale
	Reliability float64                `json:"reliability"` // 0-1 scale
	Latency     time.Duration          `json:"latency"`     // Expected response time
	Metadata    map[string]interface{} `json:"metadata"`
}

// AGIGuidanceRequest represents a request for guidance from AGI
type AGIGuidanceRequest struct {
	AgentID     string                 `json:"agent_id"`
	Context     string                 `json:"context"`
	Challenge   string                 `json:"challenge"`
	Options     []string               `json:"options"`
	Constraints map[string]interface{} `json:"constraints"`
	Priority    int                    `json:"priority"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
}

// AGIGuidanceResponse represents guidance provided by AGI
type AGIGuidanceResponse struct {
	RequestID          string                 `json:"request_id"`
	Recommendation     string                 `json:"recommendation"`
	Reasoning          string                 `json:"reasoning"`
	Confidence         float64                `json:"confidence"`
	AlternativeOptions []string               `json:"alternative_options"`
	Resources          []string               `json:"resources"`
	FollowUp           []string               `json:"follow_up"`
	Metadata           map[string]interface{} `json:"metadata"`
	Timestamp          time.Time              `json:"timestamp"`
}
