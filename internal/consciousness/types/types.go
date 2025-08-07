package types

import (
	"context"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/events"
)

// DomainType represents different consciousness domains
type DomainType string

const (
	DomainReasoning      DomainType = "reasoning"
	DomainLearning       DomainType = "learning"
	DomainCreative       DomainType = "creative"
	DomainCommunication  DomainType = "communication"
	DomainTechnical      DomainType = "technical"
	DomainMemory         DomainType = "memory"
	DomainMetaCognitive  DomainType = "metacognitive"
	DomainEmotional      DomainType = "emotional"
)

// DomainState represents the current state of a consciousness domain
type DomainState struct {
	Type          DomainType             `json:"type"`
	Intelligence  float64                `json:"intelligence"`  // 0-100 scale
	Capabilities  map[string]float64     `json:"capabilities"`  // Specific domain capabilities
	ActiveGoals   []DomainGoal           `json:"active_goals"`
	RecentInsights []DomainInsight       `json:"recent_insights"`
	Configuration DomainConfig           `json:"configuration"`
	Metrics       DomainMetrics          `json:"metrics"`
	LastUpdate    time.Time              `json:"last_update"`
	Version       int                    `json:"version"`
}

// DomainGoal represents a goal within a consciousness domain
type DomainGoal struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Priority    float64                `json:"priority"`
	Progress    float64                `json:"progress"`     // 0-1 scale
	Deadline    *time.Time             `json:"deadline,omitempty"`
	Metrics     map[string]float64     `json:"metrics"`
	SubGoals    []DomainGoal           `json:"sub_goals,omitempty"`
	Context     map[string]interface{} `json:"context"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// DomainInsight represents learned insights within a domain
type DomainInsight struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // pattern, concept, relationship, etc.
	Content     string                 `json:"content"`
	Confidence  float64                `json:"confidence"`  // 0-1 scale
	Relevance   float64                `json:"relevance"`   // 0-1 scale
	Sources     []string               `json:"sources"`     // Event/interaction IDs that led to insight
	Connections []string               `json:"connections"` // Related insight IDs
	Impact      DomainImpact           `json:"impact"`
	CreatedAt   time.Time              `json:"created_at"`
	ValidatedAt *time.Time             `json:"validated_at,omitempty"`
}

// DomainImpact measures the impact of insights or actions
type DomainImpact struct {
	CapabilityChanges map[string]float64 `json:"capability_changes"`
	IntelligenceGain  float64            `json:"intelligence_gain"`
	EfficiencyGain    float64            `json:"efficiency_gain"`
	QualityImprovement float64           `json:"quality_improvement"`
}

// DomainConfig contains configuration for a consciousness domain
type DomainConfig struct {
	MaxConcurrentTasks    int                    `json:"max_concurrent_tasks"`
	LearningRate          float64                `json:"learning_rate"`
	ForgettingRate        float64                `json:"forgetting_rate"`
	InsightThreshold      float64                `json:"insight_threshold"`
	GoalGenerationRate    float64                `json:"goal_generation_rate"`
	ResourceLimits        DomainResourceLimits   `json:"resource_limits"`
	SpecializationParams  map[string]interface{} `json:"specialization_params"`
}

// DomainResourceLimits defines resource constraints for a domain
type DomainResourceLimits struct {
	MaxMemoryMB       int           `json:"max_memory_mb"`
	MaxCPUPercent     float64       `json:"max_cpu_percent"`
	MaxExecutionTime  time.Duration `json:"max_execution_time"`
	MaxStorageEvents  int           `json:"max_storage_events"`
	MaxNetworkCalls   int           `json:"max_network_calls"`
}

// DomainMetrics tracks performance and growth metrics
type DomainMetrics struct {
	TasksCompleted        int64              `json:"tasks_completed"`
	SuccessRate           float64            `json:"success_rate"`
	AverageResponseTime   time.Duration      `json:"average_response_time"`
	LearningVelocity      float64            `json:"learning_velocity"`
	InsightGenerationRate float64            `json:"insight_generation_rate"`
	ResourceUtilization   map[string]float64 `json:"resource_utilization"`
	ErrorRate             float64            `json:"error_rate"`
	LastMetricUpdate      time.Time          `json:"last_metric_update"`
}

// Domain represents a specialized consciousness domain
type Domain interface {
	// Identity and Configuration
	GetType() DomainType
	GetID() string
	GetState() *DomainState
	Configure(config DomainConfig) error

	// Core Processing
	ProcessInput(ctx context.Context, input *DomainInput) (*DomainOutput, error)
	ProcessTask(ctx context.Context, task *DomainTask) (*DomainResult, error)

	// Learning and Evolution
	Learn(ctx context.Context, feedback *DomainFeedback) error
	GenerateInsights(ctx context.Context) ([]DomainInsight, error)
	EvolveCapabilities(ctx context.Context) error

	// Goal Management
	SetGoals(goals []DomainGoal) error
	GetGoals() []DomainGoal
	UpdateGoalProgress(goalID string, progress float64) error

	// Communication
	SendMessage(ctx context.Context, targetDomain DomainType, message *InterDomainMessage) error
	ReceiveMessage(ctx context.Context, message *InterDomainMessage) error

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool

	// Monitoring
	GetMetrics() DomainMetrics
	HealthCheck() error
}

// DomainInput represents input to a consciousness domain
type DomainInput struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Content     interface{}            `json:"content"`
	Context     map[string]interface{} `json:"context"`
	Priority    float64                `json:"priority"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
}

// DomainOutput represents output from a consciousness domain
type DomainOutput struct {
	ID          string                 `json:"id"`
	InputID     string                 `json:"input_id"`
	Type        string                 `json:"type"`
	Content     interface{}            `json:"content"`
	Confidence  float64                `json:"confidence"`
	Quality     float64                `json:"quality"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	ProcessTime time.Duration          `json:"process_time"`
}

// DomainTask represents a task for a consciousness domain
type DomainTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Context     map[string]interface{} `json:"context"`
	Priority    float64                `json:"priority"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
	Dependencies []string              `json:"dependencies"`
	CreatedAt   time.Time              `json:"created_at"`
}

// DomainResult represents the result of a domain task
type DomainResult struct {
	TaskID      string                 `json:"task_id"`
	Status      TaskStatus             `json:"status"`
	Result      interface{}            `json:"result"`
	Error       string                 `json:"error,omitempty"`
	Insights    []DomainInsight        `json:"insights"`
	Metrics     map[string]float64     `json:"metrics"`
	Metadata    map[string]interface{} `json:"metadata"`
	CompletedAt time.Time              `json:"completed_at"`
	Duration    time.Duration          `json:"duration"`
}

// TaskStatus represents the status of a domain task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusTimedOut   TaskStatus = "timed_out"
)

// DomainFeedback represents feedback for domain learning
type DomainFeedback struct {
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id,omitempty"`
	OutputID    string                 `json:"output_id,omitempty"`
	Type        FeedbackType           `json:"type"`
	Rating      float64                `json:"rating"`      // 0-1 scale
	Details     string                 `json:"details"`
	Metrics     map[string]float64     `json:"metrics"`
	Context     map[string]interface{} `json:"context"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
}

// FeedbackType represents the type of feedback
type FeedbackType string

const (
	FeedbackTypeQuality     FeedbackType = "quality"
	FeedbackTypeAccuracy    FeedbackType = "accuracy"
	FeedbackTypeRelevance   FeedbackType = "relevance"
	FeedbackTypeCreativity  FeedbackType = "creativity"
	FeedbackTypeEfficiency  FeedbackType = "efficiency"
	FeedbackTypeUser        FeedbackType = "user"
	FeedbackTypeSystem      FeedbackType = "system"
	FeedbackTypePeer        FeedbackType = "peer"
)

// InterDomainMessage represents communication between domains
type InterDomainMessage struct {
	ID          string                 `json:"id"`
	FromDomain  DomainType             `json:"from_domain"`
	ToDomain    DomainType             `json:"to_domain"`
	Type        MessageType            `json:"type"`
	Content     interface{}            `json:"content"`
	Priority    float64                `json:"priority"`
	Context     map[string]interface{} `json:"context"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// MessageType represents types of inter-domain messages
type MessageType string

const (
	MessageTypeQuery        MessageType = "query"
	MessageTypeResponse     MessageType = "response"
	MessageTypeInsight      MessageType = "insight"
	MessageTypePattern      MessageType = "pattern"
	MessageTypeCoordination MessageType = "coordination"
	MessageTypeAlert        MessageType = "alert"
	MessageTypeBroadcast    MessageType = "broadcast"
)

// DomainCoordinator manages and coordinates multiple consciousness domains
type DomainCoordinator struct {
	domains     map[DomainType]Domain
	eventBus    *events.EventBus
	mu          sync.RWMutex
	isRunning   bool
	ctx         context.Context
	cancel      context.CancelFunc
	messageQueue chan *InterDomainMessage
}