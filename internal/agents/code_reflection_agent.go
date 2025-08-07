package agents

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
)

// CodeReflectionAgent provides comprehensive self-analysis and improvement capabilities for the codebase
type CodeReflectionAgent struct {
	*BaseAgent

	// Core services
	ragSystem    *rag.RAGSystem
	modelManager *ai.ModelManager

	// Analysis configuration
	analysisConfig *CodeAnalysisConfig

	// State management
	lastAnalysisTime   time.Time
	lastIndexingTime   time.Time
	continuousMode     bool
	analysisQueue      chan *AnalysisTask
	fixQueue           chan *FixTask
	improvementHistory []*CodeImprovement

	// RAG integration for code knowledge
	codeIndex         *CodeIndex
	codebaseKnowledge map[string]*FileKnowledge

	// Senior engineer capabilities
	fixPlanner     *FixPlanner
	implementer    *CodeImplementer
	tester         *CodeTester
	backlogManager *BacklogManager

	// File system monitoring
	watchedPaths      []string
	excludePatterns   []*regexp.Regexp
	includeExtensions []string

	// Quality metrics and tracking
	codeQualityMetrics *CodeQualityMetrics
	issueTracker       *IssueTracker
	fixTracker         *FixTracker
}

// CodeIndex manages the RAG vector database for code knowledge
type CodeIndex struct {
	indexedFiles    map[string]*IndexedFile
	lastIndexUpdate time.Time
	vectorNamespace string
}

// IndexedFile represents a file in the code index
type IndexedFile struct {
	Path         string    `json:"path"`
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IndexedAt    time.Time `json:"indexed_at"`
	DocumentID   string    `json:"document_id"`
	ChunkCount   int       `json:"chunk_count"`
	Dependencies []string  `json:"dependencies"`
	Exports      []string  `json:"exports"`
	Imports      []string  `json:"imports"`
}

// FileKnowledge represents deep knowledge about a file
type FileKnowledge struct {
	Path            string                 `json:"path"`
	Language        string                 `json:"language"`
	Purpose         string                 `json:"purpose"`
	MainFunctions   []string               `json:"main_functions"`
	Dependencies    []string               `json:"dependencies"`
	ComplexityScore float64                `json:"complexity_score"`
	QualityScore    float64                `json:"quality_score"`
	TechnicalDebt   float64                `json:"technical_debt"`
	KnownIssues     []string               `json:"known_issues"`
	LastAnalyzed    time.Time              `json:"last_analyzed"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// FixPlanner creates comprehensive plans for code improvements
type FixPlanner struct {
	agent          *CodeReflectionAgent
	planningModel  string
	activePlans    map[string]*FixPlan
	planTemplates  map[string]*PlanTemplate
	testStrategy   *TestStrategy
	riskAssessment *RiskAssessment
}

// FixPlan represents a comprehensive plan to fix issues
type FixPlan struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Issues          []string               `json:"issues"` // Issue IDs being addressed
	Priority        int                    `json:"priority"`
	Complexity      string                 `json:"complexity"` // "simple", "moderate", "complex"
	EstimatedEffort time.Duration          `json:"estimated_effort"`
	Prerequisites   []string               `json:"prerequisites"`
	Steps           []*FixStep             `json:"steps"`
	TestPlan        *TestPlan              `json:"test_plan"`
	RollbackPlan    *RollbackPlan          `json:"rollback_plan"`
	Status          string                 `json:"status"` // "planned", "in_progress", "testing", "completed", "failed"
	CreatedAt       time.Time              `json:"created_at"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// FixStep represents a single step in a fix plan
type FixStep struct {
	ID           string                 `json:"id"`
	Order        int                    `json:"order"`
	Type         string                 `json:"type"` // "analyze", "modify", "test", "validate"
	Description  string                 `json:"description"`
	FilePath     string                 `json:"file_path,omitempty"`
	Action       string                 `json:"action"` // "replace", "insert", "delete", "refactor"
	Content      string                 `json:"content,omitempty"`
	LineStart    int                    `json:"line_start,omitempty"`
	LineEnd      int                    `json:"line_end,omitempty"`
	Dependencies []string               `json:"dependencies"`
	Validation   *ValidationCriteria    `json:"validation,omitempty"`
	Status       string                 `json:"status"` // "pending", "in_progress", "completed", "failed"
	ExecutedAt   *time.Time             `json:"executed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CodeImplementer executes fix plans
type CodeImplementer struct {
	agent        *CodeReflectionAgent
	backupDir    string
	dryRunMode   bool
	safetyChecks bool
}

// CodeTester validates implementations
type CodeTester struct {
	agent        *CodeReflectionAgent
	testCommands map[string][]string
	testTimeouts map[string]time.Duration
	testResults  map[string]*TestResult
}

// BacklogManager manages the improvement backlog
type BacklogManager struct {
	agent           *CodeReflectionAgent
	backlog         []*BacklogItem
	priorityMatrix  *PriorityMatrix
	sprintPlanning  *SprintPlanning
	velocityTracker *VelocityTracker
}

// FixTask represents a task to implement a fix
type FixTask struct {
	PlanID    string                 `json:"plan_id"`
	Priority  int                    `json:"priority"`
	CreatedAt time.Time              `json:"created_at"`
	Context   map[string]interface{} `json:"context"`
}

// FixTracker tracks all fixes and their outcomes
type FixTracker struct {
	AppliedFixes    []*AppliedFix      `json:"applied_fixes"`
	FailedFixes     []*FailedFix       `json:"failed_fixes"`
	FixMetrics      *FixMetrics        `json:"fix_metrics"`
	SuccessPatterns map[string]float64 `json:"success_patterns"`
}

// AppliedFix represents a successfully applied fix
type AppliedFix struct {
	PlanID       string                 `json:"plan_id"`
	IssuesFixed  []string               `json:"issues_fixed"`
	FilesChanged []string               `json:"files_changed"`
	LinesChanged int                    `json:"lines_changed"`
	TestsPassed  int                    `json:"tests_passed"`
	AppliedAt    time.Time              `json:"applied_at"`
	Impact       *FixImpact             `json:"impact"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Additional types for comprehensive functionality
type PlanTemplate struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IssueTypes  []string   `json:"issue_types"`
	Template    []*FixStep `json:"template"`
}

type TestStrategy struct {
	RequiredTests []string `json:"required_tests"`
	TestCommands  []string `json:"test_commands"`
	CoverageMins  float64  `json:"coverage_mins"`
}

type RiskAssessment struct {
	RiskFactors map[string]float64 `json:"risk_factors"`
	Thresholds  map[string]float64 `json:"thresholds"`
}

type TestPlan struct {
	PreTests    []string `json:"pre_tests"`
	PostTests   []string `json:"post_tests"`
	Validations []string `json:"validations"`
}

type RollbackPlan struct {
	BackupPaths []string `json:"backup_paths"`
	Commands    []string `json:"commands"`
	Validations []string `json:"validations"`
}

type ValidationCriteria struct {
	SyntaxCheck  bool     `json:"syntax_check"`
	TestsPass    bool     `json:"tests_pass"`
	LintingPass  bool     `json:"linting_pass"`
	CustomChecks []string `json:"custom_checks"`
}

type TestResult struct {
	TestName   string        `json:"test_name"`
	Passed     bool          `json:"passed"`
	Duration   time.Duration `json:"duration"`
	Output     string        `json:"output"`
	Error      string        `json:"error"`
	ExecutedAt time.Time     `json:"executed_at"`
}

type BacklogItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // "feature", "improvement", "refactor", "fix"
	Priority    int       `json:"priority"`
	Effort      int       `json:"effort"` // Story points
	Value       int       `json:"value"`  // Business value
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
}

type PriorityMatrix struct {
	Criteria map[string]float64 `json:"criteria"`
	Weights  map[string]float64 `json:"weights"`
}

type SprintPlanning struct {
	CurrentSprint  *Sprint       `json:"current_sprint"`
	SprintHistory  []*Sprint     `json:"sprint_history"`
	SprintLength   time.Duration `json:"sprint_length"`
	VelocityTarget int           `json:"velocity_target"`
}

type Sprint struct {
	ID        string         `json:"id"`
	StartDate time.Time      `json:"start_date"`
	EndDate   time.Time      `json:"end_date"`
	Items     []*BacklogItem `json:"items"`
	Completed int            `json:"completed"`
	Velocity  int            `json:"velocity"`
}

type VelocityTracker struct {
	WeeklyVelocity  []int     `json:"weekly_velocity"`
	MonthlyVelocity []int     `json:"monthly_velocity"`
	AverageVelocity float64   `json:"average_velocity"`
	TrendDirection  string    `json:"trend_direction"` // "up", "down", "stable"
	LastUpdated     time.Time `json:"last_updated"`
}

type FailedFix struct {
	PlanID   string    `json:"plan_id"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
	Reason   string    `json:"reason"`
	Attempts int       `json:"attempts"`
}

type FixMetrics struct {
	TotalFixes      int           `json:"total_fixes"`
	SuccessfulFixes int           `json:"successful_fixes"`
	FailedFixes     int           `json:"failed_fixes"`
	SuccessRate     float64       `json:"success_rate"`
	AverageFixTime  time.Duration `json:"average_fix_time"`
	ImpactScore     float64       `json:"impact_score"`
}

type FixImpact struct {
	QualityImprovement     float64 `json:"quality_improvement"`
	TechnicalDebtReduction float64 `json:"technical_debt_reduction"`
	PerformanceGain        float64 `json:"performance_gain"`
	SecurityImprovement    float64 `json:"security_improvement"`
	MaintainabilityGain    float64 `json:"maintainability_gain"`
}

// CodeAnalysisConfig holds configuration for code analysis
type CodeAnalysisConfig struct {
	RootPath         string        `json:"root_path"`
	AnalysisInterval time.Duration `json:"analysis_interval"`
	IndexingInterval time.Duration `json:"indexing_interval"`
	MaxFileSizeKB    int           `json:"max_file_size_kb"`
	AnalysisDepth    string        `json:"analysis_depth"` // "surface", "deep", "comprehensive"
	AutoFixEnabled   bool          `json:"auto_fix_enabled"`
	BackupChanges    bool          `json:"backup_changes"`
	RequireApproval  bool          `json:"require_approval"`
	MaxFixesPerRun   int           `json:"max_fixes_per_run"`
	MinConfidence    float64       `json:"min_confidence"`

	// Analysis types to perform
	StaticAnalysis        bool `json:"static_analysis"`
	SecurityAnalysis      bool `json:"security_analysis"`
	PerformanceAnalysis   bool `json:"performance_analysis"`
	ArchitectureAnalysis  bool `json:"architecture_analysis"`
	DocumentationAnalysis bool `json:"documentation_analysis"`
	DependencyAnalysis    bool `json:"dependency_analysis"`
	TestCoverageAnalysis  bool `json:"test_coverage_analysis"`
}

// AnalysisTask represents a code analysis task
type AnalysisTask struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // "full", "incremental", "file", "package", "index"
	TargetPath string                 `json:"target_path"`
	Priority   int                    `json:"priority"`
	CreatedAt  time.Time              `json:"created_at"`
	Context    map[string]interface{} `json:"context"`
}

// CodeImprovement represents a suggested improvement
type CodeImprovement struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "refactor", "optimize", "fix", "document", "modernize"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	FilePath    string                 `json:"file_path"`
	LineStart   int                    `json:"line_start"`
	LineEnd     int                    `json:"line_end"`
	Original    string                 `json:"original"`
	Suggested   string                 `json:"suggested"`
	Priority    int                    `json:"priority"`
	Confidence  float64                `json:"confidence"`
	Applied     bool                   `json:"applied"`
	PlanID      string                 `json:"plan_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	AppliedAt   *time.Time             `json:"applied_at,omitempty"`
	Impact      *FixImpact             `json:"impact,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CodeQualityMetrics tracks overall code quality
type CodeQualityMetrics struct {
	TotalFiles         int       `json:"total_files"`
	TotalLines         int       `json:"total_lines"`
	IndexedFiles       int       `json:"indexed_files"`
	QualityScore       float64   `json:"quality_score"`
	TechnicalDebt      float64   `json:"technical_debt"`
	Coverage           float64   `json:"coverage"`
	Complexity         float64   `json:"complexity"`
	Maintainability    float64   `json:"maintainability"`
	Duplication        float64   `json:"duplication"`
	SecurityScore      float64   `json:"security_score"`
	PerformanceScore   float64   `json:"performance_score"`
	DocumentationScore float64   `json:"documentation_score"`
	LastAnalyzed       time.Time `json:"last_analyzed"`
	LastIndexed        time.Time `json:"last_indexed"`
	ImprovementsTrend  []float64 `json:"improvements_trend"`
	VelocityTrend      []int     `json:"velocity_trend"`
}

// IssueTracker tracks code issues
type IssueTracker struct {
	ActiveIssues     []*CodeIssue             `json:"active_issues"`
	ResolvedIssues   []*CodeIssue             `json:"resolved_issues"`
	IssuesByType     map[string][]*CodeIssue  `json:"issues_by_type"`
	IssuesBySeverity map[string][]*CodeIssue  `json:"issues_by_severity"`
	IssueTrends      map[string][]TrendPoint  `json:"issue_trends"`
	IssuePatterns    map[string]*IssuePattern `json:"issue_patterns"`
}

// CodeIssue represents a code quality issue
type CodeIssue struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`     // "error", "warning", "suggestion", "security", "performance"
	Severity      string                 `json:"severity"` // "critical", "high", "medium", "low"
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	FilePath      string                 `json:"file_path"`
	Line          int                    `json:"line"`
	Column        int                    `json:"column"`
	Rule          string                 `json:"rule"`
	Category      string                 `json:"category"`
	Impact        float64                `json:"impact"`
	FixConfidence float64                `json:"fix_confidence"`
	AutoFixable   bool                   `json:"auto_fixable"`
	PlanID        string                 `json:"plan_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// IssuePattern represents a recurring pattern of issues
type IssuePattern struct {
	Pattern     string    `json:"pattern"`
	Count       int       `json:"count"`
	LastSeen    time.Time `json:"last_seen"`
	Confidence  float64   `json:"confidence"`
	AutoFixable bool      `json:"auto_fixable"`
}

// TrendPoint represents a data point in trend analysis
type TrendPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Context   map[string]interface{} `json:"context"`
}

// NewCodeReflectionAgent creates a new advanced code reflection agent
func NewCodeReflectionAgent(ragSystem *rag.RAGSystem, modelManager *ai.ModelManager) *CodeReflectionAgent {
	// Create agent info with enhanced capabilities
	agentInfo := AgentInfo{
		ID:          "code-reflection-agent",
		Name:        "Code Reflection Agent",
		Type:        "analyzer",
		Version:     "2.0.0",
		Description: "Advanced self-analyzing agent that indexes, analyzes, and automatically improves the codebase like a senior software engineer",
		Capabilities: []AgentCapability{
			{
				Name:        "code_analysis",
				Description: "Analyze code for issues and improvements",
				Version:     "2.0.0",
				InputTypes:  []string{"file_path", "code_content", "repository"},
				OutputTypes: []string{"analysis_report", "issue_list", "improvement_plan"},
			},
			{
				Name:        "issue_detection",
				Description: "Detect and categorize code quality issues",
				Version:     "2.0.0",
				InputTypes:  []string{"source_code", "file_tree"},
				OutputTypes: []string{"issues", "patterns", "suggestions"},
			},
			{
				Name:        "code_indexing",
				Description: "Index entire codebase into RAG vector database",
				Version:     "2.0.0",
				InputTypes:  []string{"repository", "file_tree"},
				OutputTypes: []string{"index", "knowledge_graph"},
			},
			{
				Name:        "automatic_fixing",
				Description: "Automatically plan and implement code fixes",
				Version:     "2.0.0",
				InputTypes:  []string{"issues", "improvement_plans"},
				OutputTypes: []string{"fix_plans", "implementations", "test_results"},
			},
			{
				Name:        "backlog_management",
				Description: "Manage improvement backlog like a senior engineer",
				Version:     "2.0.0",
				InputTypes:  []string{"issues", "requirements"},
				OutputTypes: []string{"backlog", "sprint_plans", "priorities"},
			},
		},
		Status:    "initializing",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create base agent
	baseAgent := NewBaseAgent(agentInfo)

	// Enhanced configuration with senior engineer capabilities
	defaultConfig := &CodeAnalysisConfig{
		RootPath:              ".",
		AnalysisInterval:      30 * time.Minute, // More frequent analysis
		IndexingInterval:      time.Hour,        // Regular re-indexing
		MaxFileSizeKB:         1000,             // Larger files
		AnalysisDepth:         "comprehensive",
		AutoFixEnabled:        true, // Enable automatic fixes
		BackupChanges:         true,
		RequireApproval:       false, // Allow autonomous fixes for low-risk changes
		MaxFixesPerRun:        5,
		MinConfidence:         0.8,
		StaticAnalysis:        true,
		SecurityAnalysis:      true,
		PerformanceAnalysis:   true,
		ArchitectureAnalysis:  true,
		DocumentationAnalysis: true,
		DependencyAnalysis:    true,
		TestCoverageAnalysis:  true,
	}

	// Create code reflection agent with enhanced capabilities
	cra := &CodeReflectionAgent{
		BaseAgent:          baseAgent,
		ragSystem:          ragSystem,
		modelManager:       modelManager,
		analysisConfig:     defaultConfig,
		analysisQueue:      make(chan *AnalysisTask, 200),
		fixQueue:           make(chan *FixTask, 100),
		improvementHistory: make([]*CodeImprovement, 0),
		watchedPaths:       []string{"internal", "pkg", "cmd", "frontend", "scripts"},
		includeExtensions:  []string{".go", ".md", ".yaml", ".yml", ".json", ".js", ".ts", ".tsx", ".css", ".html", ".sh", ".py", ".sql"},
		excludePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\.git/`),
			regexp.MustCompile(`build/`),
			regexp.MustCompile(`vendor/`),
			regexp.MustCompile(`node_modules/`),
			regexp.MustCompile(`\.next/`),
			regexp.MustCompile(`dist/`),
			regexp.MustCompile(`\.(log|tmp|cache|lock)$`),
			regexp.MustCompile(`_test\.go$`), // Include tests but handle separately
		},
		codeIndex: &CodeIndex{
			indexedFiles:    make(map[string]*IndexedFile),
			vectorNamespace: "codebase",
		},
		codebaseKnowledge: make(map[string]*FileKnowledge),
		codeQualityMetrics: &CodeQualityMetrics{
			QualityScore:      0.0,
			ImprovementsTrend: make([]float64, 0, 100),
			VelocityTrend:     make([]int, 0, 50),
		},
		issueTracker: &IssueTracker{
			ActiveIssues:     make([]*CodeIssue, 0),
			ResolvedIssues:   make([]*CodeIssue, 0),
			IssuesByType:     make(map[string][]*CodeIssue),
			IssuesBySeverity: make(map[string][]*CodeIssue),
			IssueTrends:      make(map[string][]TrendPoint),
			IssuePatterns:    make(map[string]*IssuePattern),
		},
		fixTracker: &FixTracker{
			AppliedFixes:    make([]*AppliedFix, 0),
			FailedFixes:     make([]*FailedFix, 0),
			FixMetrics:      &FixMetrics{},
			SuccessPatterns: make(map[string]float64),
		},
	}

	// Initialize senior engineer components
	cra.fixPlanner = &FixPlanner{
		agent:         cra,
		planningModel: "ollama-cogito",
		activePlans:   make(map[string]*FixPlan),
		planTemplates: make(map[string]*PlanTemplate),
		testStrategy: &TestStrategy{
			RequiredTests: []string{"go test ./...", "go vet ./...", "gofmt -l ."},
			TestCommands:  []string{"make test", "go test -race ./..."},
			CoverageMins:  0.8,
		},
		riskAssessment: &RiskAssessment{
			RiskFactors: map[string]float64{
				"file_size":        0.1,
				"complexity":       0.3,
				"test_coverage":    0.2,
				"change_frequency": 0.2,
				"critical_path":    0.2,
			},
			Thresholds: map[string]float64{
				"low_risk":    0.3,
				"medium_risk": 0.6,
				"high_risk":   0.8,
			},
		},
	}

	cra.implementer = &CodeImplementer{
		agent:        cra,
		backupDir:    "./backups/code_reflection",
		dryRunMode:   false,
		safetyChecks: true,
	}

	cra.tester = &CodeTester{
		agent: cra,
		testCommands: map[string][]string{
			"go":         {"go test", "go vet", "gofmt -l"},
			"javascript": {"npm test", "npm run lint"},
			"typescript": {"npm test", "npm run type-check"},
		},
		testTimeouts: map[string]time.Duration{
			"go":         5 * time.Minute,
			"javascript": 3 * time.Minute,
			"typescript": 3 * time.Minute,
		},
		testResults: make(map[string]*TestResult),
	}

	cra.backlogManager = &BacklogManager{
		agent:   cra,
		backlog: make([]*BacklogItem, 0),
		priorityMatrix: &PriorityMatrix{
			Criteria: map[string]float64{
				"business_value":  0.3,
				"technical_debt":  0.25,
				"security_risk":   0.2,
				"maintainability": 0.15,
				"performance":     0.1,
			},
			Weights: map[string]float64{
				"critical": 4.0,
				"high":     3.0,
				"medium":   2.0,
				"low":      1.0,
			},
		},
		sprintPlanning: &SprintPlanning{
			SprintLength:   7 * 24 * time.Hour, // Weekly sprints
			VelocityTarget: 20,                 // Target story points per sprint
			SprintHistory:  make([]*Sprint, 0),
		},
		velocityTracker: &VelocityTracker{
			WeeklyVelocity:  make([]int, 0, 12),
			MonthlyVelocity: make([]int, 0, 6),
			TrendDirection:  "stable",
		},
	}

	// Initialize plan templates
	cra.initializePlanTemplates()

	// Set process function for query handling
	baseAgent.SetProcessFunction(cra.processQuery)

	return cra
}

// initializePlanTemplates sets up common fix plan templates
func (cra *CodeReflectionAgent) initializePlanTemplates() {
	// Error handling template
	cra.fixPlanner.planTemplates["error_handling"] = &PlanTemplate{
		Name:        "Error Handling Fix",
		Description: "Template for adding proper error handling",
		IssueTypes:  []string{"error_handling", "missing_error_check"},
		Template: []*FixStep{
			{
				Type:        "analyze",
				Description: "Analyze the function context and error sources",
				Order:       1,
			},
			{
				Type:        "modify",
				Description: "Add proper error handling",
				Order:       2,
				Action:      "insert",
			},
			{
				Type:        "test",
				Description: "Run tests to ensure error handling works",
				Order:       3,
			},
		},
	}

	// Code style template
	cra.fixPlanner.planTemplates["style_fix"] = &PlanTemplate{
		Name:        "Code Style Fix",
		Description: "Template for fixing code style issues",
		IssueTypes:  []string{"style", "formatting", "long_line"},
		Template: []*FixStep{
			{
				Type:        "modify",
				Description: "Apply code formatting fixes",
				Order:       1,
				Action:      "replace",
			},
			{
				Type:        "validate",
				Description: "Validate formatting with linters",
				Order:       2,
			},
		},
	}

	// Security fix template
	cra.fixPlanner.planTemplates["security_fix"] = &PlanTemplate{
		Name:        "Security Fix",
		Description: "Template for addressing security issues",
		IssueTypes:  []string{"security", "vulnerability"},
		Template: []*FixStep{
			{
				Type:        "analyze",
				Description: "Analyze security vulnerability",
				Order:       1,
			},
			{
				Type:        "modify",
				Description: "Apply security fix",
				Order:       2,
				Action:      "replace",
			},
			{
				Type:        "test",
				Description: "Run security tests",
				Order:       3,
			},
			{
				Type:        "validate",
				Description: "Validate security improvement",
				Order:       4,
			},
		},
	}
}

// Initialize initializes the code reflection agent
func (cra *CodeReflectionAgent) Initialize(ctx context.Context, config map[string]interface{}) error {
	if err := cra.BaseAgent.Initialize(ctx, config); err != nil {
		return err
	}

	// Apply any configuration overrides
	if rootPath, exists := config["root_path"]; exists {
		if path, ok := rootPath.(string); ok {
			cra.analysisConfig.RootPath = path
		}
	}

	if interval, exists := config["analysis_interval"]; exists {
		if intervalStr, ok := interval.(string); ok {
			if duration, err := time.ParseDuration(intervalStr); err == nil {
				cra.analysisConfig.AnalysisInterval = duration
			}
		}
	}

	log.Printf("[CodeReflectionAgent] Initialized with root path: %s", cra.analysisConfig.RootPath)
	return nil
}

// Start starts the code reflection agent
func (cra *CodeReflectionAgent) Start(ctx context.Context) error {
	if err := cra.BaseAgent.Start(ctx); err != nil {
		return err
	}

	// Enable continuous mode by default for enhanced capabilities
	cra.continuousMode = true

	// Start continuous analysis if enabled
	if cra.continuousMode {
		go cra.continuousAnalysisLoop(ctx)
	}

	// Start analysis worker
	go cra.analysisWorker(ctx)

	// Trigger initial analysis
	go func() {
		// Wait a bit for system to stabilize
		time.Sleep(5 * time.Second)
		cra.triggerFullAnalysis()
	}()

	log.Printf("[CodeReflectionAgent] Started successfully")

	// Start enhanced capabilities by default
	return cra.StartEnhanced(ctx)
}

// processQuery processes queries for the code reflection agent
func (cra *CodeReflectionAgent) processQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	startTime := time.Now()

	// Parse query type and content
	switch {
	case strings.Contains(query.Text, "analyze code") || query.Type == "analyze_code":
		return cra.handleAnalyzeCodeQuery(ctx, query)
	case strings.Contains(query.Text, "code issues") || query.Type == "get_issues":
		return cra.handleGetIssuesQuery(ctx, query)
	case strings.Contains(query.Text, "code quality") || query.Type == "get_metrics":
		return cra.handleGetMetricsQuery(ctx, query)
	case strings.Contains(query.Text, "trigger analysis") || query.Type == "trigger_analysis":
		return cra.handleTriggerAnalysisQuery(ctx, query)
	case strings.Contains(query.Text, "codebase indexing") || query.Type == "trigger_analysis" && query.Metadata["analysis_type"] == "index":
		return cra.handleIndexingQuery(ctx, query)
	case query.Type == "codebase_query":
		return cra.handleCodebaseQuery(ctx, query)
	case query.Type == "smart_analysis":
		return cra.handleSmartAnalysisQuery(ctx, query)
	case query.Type == "file_knowledge":
		return cra.handleFileKnowledgeQuery(ctx, query)
	case query.Type == "get_improvements":
		return cra.handleGetImprovementsQuery(ctx, query)
	default:
		return &types.Response{
			QueryID:   query.ID,
			Text:      "I can help with code analysis, issue detection, quality metrics, codebase indexing, smart analysis, and code improvements. How can I assist you?",
			Status:    "success",
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"agent_type":      "code_reflection",
				"processing_time": time.Since(startTime).String(),
			},
		}, nil
	}
}

// handleAnalyzeCodeQuery handles code analysis requests
func (cra *CodeReflectionAgent) handleAnalyzeCodeQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	// Trigger analysis based on query parameters
	analysisType := "full"
	targetPath := cra.analysisConfig.RootPath

	if metadata := query.Metadata; metadata != nil {
		if aType, exists := metadata["analysis_type"]; exists {
			analysisType = aType
		}
		if path, exists := metadata["target_path"]; exists {
			targetPath = path
		}
	}

	cra.TriggerAnalysis(analysisType, targetPath)

	return &types.Response{
		QueryID:   query.ID,
		Text:      fmt.Sprintf("Started %s code analysis for %s", analysisType, targetPath),
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"analysis_type": analysisType,
			"target_path":   targetPath,
		},
	}, nil
}

// handleGetIssuesQuery handles requests for current issues
func (cra *CodeReflectionAgent) handleGetIssuesQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	issues := cra.GetActiveIssues()

	var response strings.Builder
	response.WriteString(fmt.Sprintf("Found %d active code issues:\n\n", len(issues)))

	// Group issues by severity
	issuesBySeverity := make(map[string][]*CodeIssue)
	for _, issue := range issues {
		issuesBySeverity[issue.Severity] = append(issuesBySeverity[issue.Severity], issue)
	}

	for _, severity := range []string{"critical", "high", "medium", "low"} {
		if issues, exists := issuesBySeverity[severity]; exists {
			response.WriteString(fmt.Sprintf("%s Issues (%d):\n", strings.Title(severity), len(issues)))
			for _, issue := range issues {
				response.WriteString(fmt.Sprintf("- %s (%s:%d): %s\n",
					issue.Title, issue.FilePath, issue.Line, issue.Description))
			}
			response.WriteString("\n")
		}
	}

	return &types.Response{
		QueryID:   query.ID,
		Text:      response.String(),
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"total_issues": fmt.Sprintf("%d", len(issues)),
		},
	}, nil
}

// handleGetMetricsQuery handles requests for quality metrics
func (cra *CodeReflectionAgent) handleGetMetricsQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	metrics := cra.codeQualityMetrics

	response := fmt.Sprintf(`Code Quality Metrics:
- Total Files: %d
- Total Lines: %d
- Quality Score: %.2f/10
- Technical Debt: %.2f
- Maintainability: %.2f
- Complexity: %.2f
- Last Analyzed: %s

Active Issues: %d
Improvements Applied: %d`,
		metrics.TotalFiles,
		metrics.TotalLines,
		metrics.QualityScore,
		metrics.TechnicalDebt,
		metrics.Maintainability,
		metrics.Complexity,
		metrics.LastAnalyzed.Format("2006-01-02 15:04:05"),
		len(cra.issueTracker.ActiveIssues),
		len(cra.improvementHistory))

	return &types.Response{
		QueryID:   query.ID,
		Text:      response,
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"quality_score": fmt.Sprintf("%.2f", metrics.QualityScore),
			"total_files":   fmt.Sprintf("%d", metrics.TotalFiles),
		},
	}, nil
}

// handleTriggerAnalysisQuery handles manual analysis trigger requests
func (cra *CodeReflectionAgent) handleTriggerAnalysisQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	cra.triggerFullAnalysis()

	return &types.Response{
		QueryID:   query.ID,
		Text:      "Full codebase analysis has been triggered",
		Status:    "success",
		Timestamp: time.Now().Unix(),
	}, nil
}

// continuousAnalysisLoop runs continuous analysis
func (cra *CodeReflectionAgent) continuousAnalysisLoop(ctx context.Context) {
	ticker := time.NewTicker(cra.analysisConfig.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Since(cra.lastAnalysisTime) >= cra.analysisConfig.AnalysisInterval {
				cra.triggerIncrementalAnalysis()
			}
		}
	}
}

// analysisWorker processes analysis tasks from the queue
func (cra *CodeReflectionAgent) analysisWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-cra.analysisQueue:
			cra.processAnalysisTask(ctx, task)
		}
	}
}

// triggerFullAnalysis starts a complete codebase analysis
func (cra *CodeReflectionAgent) triggerFullAnalysis() {
	task := &AnalysisTask{
		ID:         fmt.Sprintf("full_analysis_%d", time.Now().Unix()),
		Type:       "full",
		TargetPath: cra.analysisConfig.RootPath,
		Priority:   5,
		CreatedAt:  time.Now(),
		Context: map[string]interface{}{
			"trigger": "scheduled",
		},
	}

	select {
	case cra.analysisQueue <- task:
		log.Printf("[CodeReflectionAgent] Queued full codebase analysis")
	default:
		log.Printf("[CodeReflectionAgent] Analysis queue full, skipping full analysis")
	}
}

// triggerIncrementalAnalysis starts an incremental analysis
func (cra *CodeReflectionAgent) triggerIncrementalAnalysis() {
	task := &AnalysisTask{
		ID:         fmt.Sprintf("incremental_analysis_%d", time.Now().Unix()),
		Type:       "incremental",
		TargetPath: cra.analysisConfig.RootPath,
		Priority:   3,
		CreatedAt:  time.Now(),
		Context: map[string]interface{}{
			"since":   cra.lastAnalysisTime,
			"trigger": "continuous",
		},
	}

	select {
	case cra.analysisQueue <- task:
		log.Printf("[CodeReflectionAgent] Queued incremental analysis")
	default:
		log.Printf("[CodeReflectionAgent] Analysis queue full, skipping incremental analysis")
	}
}

// processAnalysisTask processes a single analysis task
func (cra *CodeReflectionAgent) processAnalysisTask(ctx context.Context, task *AnalysisTask) {
	log.Printf("[CodeReflectionAgent] Processing analysis task: %s (%s)", task.ID, task.Type)

	startTime := time.Now()
	cra.lastAnalysisTime = startTime

	switch task.Type {
	case "full":
		cra.performFullCodebaseAnalysis(ctx, task)
	case "incremental":
		cra.performIncrementalAnalysis(ctx, task)
	case "file":
		cra.performFileAnalysis(ctx, task)
	case "package":
		cra.performPackageAnalysis(ctx, task)
	case "index":
		cra.performIndexing(ctx, task)
	default:
		log.Printf("[CodeReflectionAgent] Unknown analysis type: %s", task.Type)
		return
	}

	duration := time.Since(startTime)
	log.Printf("[CodeReflectionAgent] Completed analysis task %s in %v", task.ID, duration)
}

// performFullCodebaseAnalysis performs a comprehensive analysis of the entire codebase
func (cra *CodeReflectionAgent) performFullCodebaseAnalysis(ctx context.Context, task *AnalysisTask) {
	log.Printf("[CodeReflectionAgent] Starting full codebase analysis...")

	// Clear previous issues for fresh analysis
	cra.issueTracker.ActiveIssues = make([]*CodeIssue, 0)
	cra.issueTracker.IssuesByType = make(map[string][]*CodeIssue)

	// Walk through all files
	var allFiles []string
	filepath.WalkDir(cra.analysisConfig.RootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file should be excluded
		if cra.shouldExcludeFile(path) {
			return nil
		}

		// Check file extension
		if cra.shouldIncludeFile(path) {
			allFiles = append(allFiles, path)
		}

		return nil
	})

	log.Printf("[CodeReflectionAgent] Found %d files to analyze", len(allFiles))

	// Analyze files
	for _, filePath := range allFiles {
		if err := cra.analyzeFile(ctx, filePath); err != nil {
			log.Printf("[CodeReflectionAgent] Error analyzing file %s: %v", filePath, err)
		}

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			log.Printf("[CodeReflectionAgent] Analysis cancelled")
			return
		default:
		}
	}

	// Generate overall quality report
	cra.generateQualityReport()

	// Report significant findings to AGI
	cra.reportAnalysisToAGI(task)

	// Trigger smart analysis if we found issues
	if len(cra.issueTracker.ActiveIssues) > 0 {
		go func() {
			// Run smart analysis in background
			if err := cra.TriggerSmartAnalysis(ctx); err != nil {
				log.Printf("[CodeReflectionAgent] Smart analysis failed: %v", err)
			}
		}()
	}

	log.Printf("[CodeReflectionAgent] Full codebase analysis completed")
}

// analyzeFile performs comprehensive analysis on a single file
func (cra *CodeReflectionAgent) analyzeFile(ctx context.Context, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Skip large files
	if len(content) > cra.analysisConfig.MaxFileSizeKB*1024 {
		return nil
	}

	fileContent := string(content)
	ext := filepath.Ext(filePath)

	// Perform analysis based on file type
	switch ext {
	case ".go":
		cra.analyzeGoFile(filePath, fileContent)
	case ".md":
		cra.analyzeMarkdownFile(filePath, fileContent)
	case ".yaml", ".yml":
		cra.analyzeYAMLFile(filePath, fileContent)
	case ".json":
		cra.analyzeJSONFile(filePath, fileContent)
	default:
		cra.analyzeGenericFile(filePath, fileContent)
	}

	return nil
}

// analyzeGoFile performs Go-specific analysis
func (cra *CodeReflectionAgent) analyzeGoFile(filePath, content string) {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		// Check for common issues
		if strings.Contains(trimmedLine, "TODO") || strings.Contains(trimmedLine, "FIXME") {
			cra.addIssue(&CodeIssue{
				ID:          fmt.Sprintf("todo_%s_%d", filePath, lineNum),
				Type:        "suggestion",
				Severity:    "low",
				Title:       "TODO/FIXME Comment",
				Description: "Consider addressing this TODO or FIXME comment",
				FilePath:    filePath,
				Line:        lineNum,
				Category:    "technical_debt",
				CreatedAt:   time.Now(),
			})
		}

		// Check for missing error handling
		if strings.Contains(trimmedLine, ":=") && strings.Contains(trimmedLine, "err") &&
			!strings.Contains(trimmedLine, "if err") {
			if i+1 < len(lines) && !strings.Contains(lines[i+1], "if err") {
				cra.addIssue(&CodeIssue{
					ID:          fmt.Sprintf("error_handling_%s_%d", filePath, lineNum),
					Type:        "warning",
					Severity:    "medium",
					Title:       "Potential Missing Error Handling",
					Description: "Error returned but not immediately checked",
					FilePath:    filePath,
					Line:        lineNum,
					Category:    "error_handling",
					CreatedAt:   time.Now(),
				})
			}
		}

		// Check for long lines
		if len(line) > 120 {
			cra.addIssue(&CodeIssue{
				ID:          fmt.Sprintf("long_line_%s_%d", filePath, lineNum),
				Type:        "suggestion",
				Severity:    "low",
				Title:       "Long Line",
				Description: "Line exceeds 120 characters",
				FilePath:    filePath,
				Line:        lineNum,
				Category:    "style",
				CreatedAt:   time.Now(),
			})
		}
	}

	// Update metrics
	cra.codeQualityMetrics.TotalFiles++
	cra.codeQualityMetrics.TotalLines += len(lines)
}

// analyzeMarkdownFile performs Markdown-specific analysis
func (cra *CodeReflectionAgent) analyzeMarkdownFile(filePath, content string) {
	lines := strings.Split(content, "\n")

	// Check for broken links, missing sections, etc.
	for i, line := range lines {
		lineNum := i + 1

		// Check for empty headings
		if strings.HasPrefix(strings.TrimSpace(line), "#") &&
			len(strings.TrimSpace(strings.TrimLeft(line, "# "))) == 0 {
			cra.addIssue(&CodeIssue{
				ID:          fmt.Sprintf("empty_heading_%s_%d", filePath, lineNum),
				Type:        "warning",
				Severity:    "low",
				Title:       "Empty Heading",
				Description: "Heading without content",
				FilePath:    filePath,
				Line:        lineNum,
				Category:    "documentation",
				CreatedAt:   time.Now(),
			})
		}
	}
}

// analyzeYAMLFile performs YAML-specific analysis
func (cra *CodeReflectionAgent) analyzeYAMLFile(filePath, content string) {
	// Basic YAML syntax and structure checks
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1

		// Check for inconsistent indentation (basic check)
		if strings.Contains(line, "\t") {
			cra.addIssue(&CodeIssue{
				ID:          fmt.Sprintf("yaml_tabs_%s_%d", filePath, lineNum),
				Type:        "warning",
				Severity:    "medium",
				Title:       "Tabs in YAML",
				Description: "YAML should use spaces for indentation, not tabs",
				FilePath:    filePath,
				Line:        lineNum,
				Category:    "syntax",
				CreatedAt:   time.Now(),
			})
		}
	}
}

// analyzeJSONFile performs JSON-specific analysis
func (cra *CodeReflectionAgent) analyzeJSONFile(filePath, content string) {
	// Basic JSON validation would go here
	// For now, just check if file is empty
	if strings.TrimSpace(content) == "" {
		cra.addIssue(&CodeIssue{
			ID:          fmt.Sprintf("empty_json_%s", filePath),
			Type:        "warning",
			Severity:    "low",
			Title:       "Empty JSON File",
			Description: "JSON file appears to be empty",
			FilePath:    filePath,
			Line:        1,
			Category:    "structure",
			CreatedAt:   time.Now(),
		})
	}
}

// analyzeGenericFile performs generic file analysis
func (cra *CodeReflectionAgent) analyzeGenericFile(filePath, content string) {
	// Check if file is empty
	if strings.TrimSpace(content) == "" {
		cra.addIssue(&CodeIssue{
			ID:          fmt.Sprintf("empty_file_%s", filePath),
			Type:        "suggestion",
			Severity:    "low",
			Title:       "Empty File",
			Description: "File appears to be empty",
			FilePath:    filePath,
			Line:        1,
			Category:    "structure",
			CreatedAt:   time.Now(),
		})
	}
}

// addIssue adds an issue to the tracker
func (cra *CodeReflectionAgent) addIssue(issue *CodeIssue) {
	cra.issueTracker.ActiveIssues = append(cra.issueTracker.ActiveIssues, issue)

	// Add to type-based grouping
	if cra.issueTracker.IssuesByType[issue.Type] == nil {
		cra.issueTracker.IssuesByType[issue.Type] = make([]*CodeIssue, 0)
	}
	cra.issueTracker.IssuesByType[issue.Type] = append(cra.issueTracker.IssuesByType[issue.Type], issue)
}

// shouldExcludeFile checks if a file should be excluded from analysis
func (cra *CodeReflectionAgent) shouldExcludeFile(path string) bool {
	for _, pattern := range cra.excludePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// shouldIncludeFile checks if a file should be included in analysis
func (cra *CodeReflectionAgent) shouldIncludeFile(path string) bool {
	ext := filepath.Ext(path)
	for _, includeExt := range cra.includeExtensions {
		if ext == includeExt {
			return true
		}
	}
	return false
}

// generateQualityReport generates an overall quality report
func (cra *CodeReflectionAgent) generateQualityReport() {
	totalIssues := len(cra.issueTracker.ActiveIssues)
	criticalIssues := 0
	highIssues := 0
	mediumIssues := 0

	for _, issue := range cra.issueTracker.ActiveIssues {
		switch issue.Severity {
		case "critical":
			criticalIssues++
		case "high":
			highIssues++
		case "medium":
			mediumIssues++
		}
	}

	log.Printf("[CodeReflectionAgent] Code quality analysis: %d total issues (%d critical, %d high, %d medium)",
		totalIssues, criticalIssues, highIssues, mediumIssues)

	// Log breakdown by type
	for issueType, issues := range cra.issueTracker.IssuesByType {
		log.Printf("[CodeReflectionAgent] - %s: %d issues", issueType, len(issues))
	}
}

// reportAnalysisToAGI reports analysis results to the AGI system
func (cra *CodeReflectionAgent) reportAnalysisToAGI(task *AnalysisTask) {
	if !cra.IsAGIIntegrationEnabled() {
		return
	}

	totalIssues := len(cra.issueTracker.ActiveIssues)
	improvementsProposed := len(cra.improvementHistory)

	cra.ReportInsightToAGI(
		InsightTypeKnowledge,
		"Code Analysis Completed",
		fmt.Sprintf("Analyzed codebase and found %d issues", totalIssues),
		map[string]interface{}{
			"analysis_type":         task.Type,
			"total_issues":          totalIssues,
			"improvements_proposed": improvementsProposed,
			"issue_breakdown":       cra.getIssueBreakdown(),
		},
		0.8,
	)
}

// getIssueBreakdown returns a breakdown of issues by type
func (cra *CodeReflectionAgent) getIssueBreakdown() map[string]int {
	breakdown := make(map[string]int)
	for issueType, issues := range cra.issueTracker.IssuesByType {
		breakdown[issueType] = len(issues)
	}
	return breakdown
}

// Additional required methods for task processing
func (cra *CodeReflectionAgent) performIncrementalAnalysis(ctx context.Context, task *AnalysisTask) {
	log.Printf("[CodeReflectionAgent] Starting incremental analysis...")
	cra.performFullCodebaseAnalysis(ctx, task)
}

func (cra *CodeReflectionAgent) performFileAnalysis(ctx context.Context, task *AnalysisTask) {
	log.Printf("[CodeReflectionAgent] Analyzing file: %s", task.TargetPath)

	if err := cra.analyzeFile(ctx, task.TargetPath); err != nil {
		log.Printf("[CodeReflectionAgent] Error analyzing file %s: %v", task.TargetPath, err)
	}
}

func (cra *CodeReflectionAgent) performPackageAnalysis(ctx context.Context, task *AnalysisTask) {
	log.Printf("[CodeReflectionAgent] Analyzing package: %s", task.TargetPath)

	filepath.WalkDir(task.TargetPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && cra.shouldIncludeFile(path) && !cra.shouldExcludeFile(path) {
			if err := cra.analyzeFile(ctx, path); err != nil {
				log.Printf("[CodeReflectionAgent] Error analyzing file %s: %v", path, err)
			}
		}

		return nil
	})
}

// Public interface methods
func (cra *CodeReflectionAgent) GetActiveIssues() []*CodeIssue {
	return cra.issueTracker.ActiveIssues
}

func (cra *CodeReflectionAgent) GetImprovementHistory() []*CodeImprovement {
	return cra.improvementHistory
}

func (cra *CodeReflectionAgent) TriggerAnalysis(analysisType, targetPath string) {
	task := &AnalysisTask{
		ID:         fmt.Sprintf("manual_%s_%d", analysisType, time.Now().UnixNano()),
		Type:       analysisType,
		TargetPath: targetPath,
		Priority:   5,
		CreatedAt:  time.Now(),
		Context: map[string]interface{}{
			"trigger": "manual",
		},
	}

	select {
	case cra.analysisQueue <- task:
		log.Printf("[CodeReflectionAgent] Queued manual analysis: %s for %s", analysisType, targetPath)
	default:
		log.Printf("[CodeReflectionAgent] Analysis queue full, skipping manual analysis")
	}
}

// performIndexing indexes the codebase into the RAG vector database
func (cra *CodeReflectionAgent) performIndexing(ctx context.Context, task *AnalysisTask) {
	log.Printf("[CodeReflectionAgent] Starting codebase indexing...")

	if cra.ragSystem == nil {
		log.Printf("[CodeReflectionAgent] Warning: RAG system not available for indexing")
		return
	}

	startTime := time.Now()
	var indexedCount int
	var allFiles []string

	// Walk through all files
	filepath.WalkDir(cra.analysisConfig.RootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file should be excluded
		if cra.shouldExcludeFile(path) {
			return nil
		}

		// Check file extension
		if cra.shouldIncludeFile(path) {
			allFiles = append(allFiles, path)
		}

		return nil
	})

	log.Printf("[CodeReflectionAgent] Found %d files to index", len(allFiles))

	// Index files into RAG vector database
	for _, filePath := range allFiles {
		if err := cra.indexFile(ctx, filePath); err != nil {
			log.Printf("[CodeReflectionAgent] Error indexing file %s: %v", filePath, err)
		} else {
			indexedCount++
		}

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			log.Printf("[CodeReflectionAgent] Indexing cancelled")
			return
		default:
		}
	}

	// Update metrics
	cra.codeQualityMetrics.IndexedFiles = indexedCount
	cra.codeQualityMetrics.LastIndexed = time.Now()
	cra.lastIndexingTime = time.Now()
	cra.codeIndex.lastIndexUpdate = time.Now()

	duration := time.Since(startTime)
	log.Printf("[CodeReflectionAgent] Completed indexing %d files in %v", indexedCount, duration)

	// Report indexing to AGI
	if cra.IsAGIIntegrationEnabled() {
		cra.ReportInsightToAGI(
			InsightTypeKnowledge,
			"Codebase Indexed",
			fmt.Sprintf("Successfully indexed %d files into RAG vector database", indexedCount),
			map[string]interface{}{
				"indexed_files": indexedCount,
				"total_files":   len(allFiles),
				"duration":      duration.String(),
			},
			0.9,
		)
	}
}

// indexFile indexes a single file into the RAG vector database
func (cra *CodeReflectionAgent) indexFile(ctx context.Context, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Skip large files
	if len(content) > cra.analysisConfig.MaxFileSizeKB*1024 {
		return nil
	}

	fileContent := string(content)

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	// Generate document ID
	documentID := fmt.Sprintf("codebase_%s_%d", strings.ReplaceAll(filePath, "/", "_"), fileInfo.ModTime().Unix())

	// Extract file knowledge
	knowledge := cra.extractFileKnowledge(filePath, fileContent)

	// Create enhanced document content for RAG
	documentContent := cra.createDocumentContent(filePath, fileContent, knowledge)

	// Add document to RAG system
	err = cra.ragSystem.AddDocument(ctx, documentContent, map[string]interface{}{
		"document_id":  documentID,
		"file_path":    filePath,
		"file_type":    filepath.Ext(filePath),
		"language":     knowledge.Language,
		"purpose":      knowledge.Purpose,
		"functions":    strings.Join(knowledge.MainFunctions, ", "),
		"dependencies": strings.Join(knowledge.Dependencies, ", "),
		"complexity":   knowledge.ComplexityScore,
		"quality":      knowledge.QualityScore,
		"indexed_at":   time.Now().Format(time.RFC3339),
	})

	if err != nil {
		return fmt.Errorf("failed to add document to RAG: %w", err)
	}

	// Update code index
	cra.codeIndex.indexedFiles[filePath] = &IndexedFile{
		Path:         filePath,
		Hash:         fmt.Sprintf("%x", content), // Simple hash
		Size:         fileInfo.Size(),
		ModTime:      fileInfo.ModTime(),
		IndexedAt:    time.Now(),
		DocumentID:   documentID,
		ChunkCount:   1, // Simplified
		Dependencies: knowledge.Dependencies,
		Exports:      knowledge.MainFunctions,
		Imports:      knowledge.Dependencies,
	}

	// Store file knowledge
	cra.codebaseKnowledge[filePath] = knowledge

	return nil
}

// extractFileKnowledge extracts deep knowledge about a file
func (cra *CodeReflectionAgent) extractFileKnowledge(filePath, content string) *FileKnowledge {
	ext := filepath.Ext(filePath)
	language := cra.detectLanguage(ext)

	knowledge := &FileKnowledge{
		Path:          filePath,
		Language:      language,
		Purpose:       cra.inferFilePurpose(filePath, content),
		MainFunctions: cra.extractFunctions(content, language),
		Dependencies:  cra.extractDependencies(content, language),
		LastAnalyzed:  time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	// Calculate complexity and quality scores
	knowledge.ComplexityScore = cra.calculateComplexity(content, language)
	knowledge.QualityScore = cra.calculateQualityScore(content, language)
	knowledge.TechnicalDebt = cra.calculateTechnicalDebt(content, language)

	return knowledge
}

// detectLanguage detects the programming language from file extension
func (cra *CodeReflectionAgent) detectLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".rs":
		return "rust"
	case ".php":
		return "php"
	case ".rb":
		return "ruby"
	case ".md":
		return "markdown"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".sh":
		return "shell"
	case ".sql":
		return "sql"
	default:
		return "unknown"
	}
}

// inferFilePurpose tries to infer what the file is for
func (cra *CodeReflectionAgent) inferFilePurpose(filePath, content string) string {
	fileName := filepath.Base(filePath)

	// Common patterns
	if strings.Contains(fileName, "test") {
		return "testing"
	}
	if strings.Contains(fileName, "main") {
		return "entry_point"
	}
	if strings.Contains(fileName, "config") {
		return "configuration"
	}
	if strings.Contains(fileName, "util") || strings.Contains(fileName, "helper") {
		return "utility"
	}
	if strings.Contains(filePath, "internal/api") {
		return "api_handler"
	}
	if strings.Contains(filePath, "internal/") {
		return "internal_logic"
	}
	if strings.Contains(filePath, "pkg/") {
		return "library_package"
	}
	if strings.Contains(filePath, "cmd/") {
		return "command_line_tool"
	}
	if strings.Contains(filePath, "frontend") {
		return "user_interface"
	}

	// Analyze content patterns
	if strings.Contains(content, "func main(") {
		return "entry_point"
	}
	if strings.Contains(content, "type") && strings.Contains(content, "interface") {
		return "interface_definition"
	}
	if strings.Contains(content, "package") {
		return "go_package"
	}

	return "unknown"
}

// extractFunctions extracts function/method names from code
func (cra *CodeReflectionAgent) extractFunctions(content, language string) []string {
	var functions []string

	switch language {
	case "go":
		// Extract Go functions: func FunctionName(
		re := regexp.MustCompile(`func\s+([A-Za-z][A-Za-z0-9_]*)\s*\(`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				functions = append(functions, match[1])
			}
		}
	case "javascript", "typescript":
		// Extract JS/TS functions: function name( or const name = function
		re := regexp.MustCompile(`(?:function\s+([A-Za-z][A-Za-z0-9_]*)|const\s+([A-Za-z][A-Za-z0-9_]*)\s*=\s*(?:function|\([^)]*\)\s*=>))`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				functions = append(functions, match[1])
			} else if len(match) > 2 && match[2] != "" {
				functions = append(functions, match[2])
			}
		}
	case "python":
		// Extract Python functions: def function_name(
		re := regexp.MustCompile(`def\s+([A-Za-z][A-Za-z0-9_]*)\s*\(`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				functions = append(functions, match[1])
			}
		}
	}

	return functions
}

// extractDependencies extracts dependencies/imports from code
func (cra *CodeReflectionAgent) extractDependencies(content, language string) []string {
	var dependencies []string

	switch language {
	case "go":
		// Extract Go imports
		re := regexp.MustCompile(`import\s+(?:\(\s*(?:[^)]+)\s*\)|"([^"]+)")`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				dependencies = append(dependencies, match[1])
			}
		}
	case "javascript", "typescript":
		// Extract JS/TS imports
		re := regexp.MustCompile(`import\s+.+\s+from\s+['"]([^'"]+)['"]`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				dependencies = append(dependencies, match[1])
			}
		}
	case "python":
		// Extract Python imports
		re := regexp.MustCompile(`(?:import\s+([A-Za-z][A-Za-z0-9_.]*)|from\s+([A-Za-z][A-Za-z0-9_.]*)\s+import)`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				dependencies = append(dependencies, match[1])
			} else if len(match) > 2 && match[2] != "" {
				dependencies = append(dependencies, match[2])
			}
		}
	}

	return dependencies
}

// calculateComplexity calculates a complexity score for the file
func (cra *CodeReflectionAgent) calculateComplexity(content, language string) float64 {
	lines := strings.Split(content, "\n")

	// Basic complexity metrics
	codeLines := 0
	complexity := 0.0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "#") {
			codeLines++
		}

		// Count complexity indicators
		if strings.Contains(trimmed, "if ") {
			complexity += 1.0
		}
		if strings.Contains(trimmed, "for ") || strings.Contains(trimmed, "while ") {
			complexity += 2.0
		}
		if strings.Contains(trimmed, "switch ") {
			complexity += 1.5
		}
		if strings.Contains(trimmed, "case ") {
			complexity += 0.5
		}
	}

	// Normalize by code length
	if codeLines > 0 {
		return complexity / float64(codeLines) * 10.0 // Scale to 0-10
	}

	return 0.0
}

// calculateQualityScore calculates a quality score for the file
func (cra *CodeReflectionAgent) calculateQualityScore(content, language string) float64 {
	lines := strings.Split(content, "\n")

	score := 10.0 // Start with perfect score
	totalLines := len(lines)

	// Deduct points for various issues
	for _, line := range lines {
		// Long lines
		if len(line) > 120 {
			score -= 0.1
		}

		// TODO/FIXME comments
		if strings.Contains(line, "TODO") || strings.Contains(line, "FIXME") {
			score -= 0.5
		}

		// Poor formatting
		if strings.Contains(line, "\t") && language == "yaml" {
			score -= 0.2
		}
	}

	// Deduct for missing documentation
	hasComments := false
	for _, line := range lines {
		if strings.Contains(line, "//") || strings.Contains(line, "/*") || strings.Contains(line, "#") {
			hasComments = true
			break
		}
	}

	if !hasComments && totalLines > 20 {
		score -= 2.0
	}

	// Ensure score is in valid range
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}

	return score
}

// calculateTechnicalDebt calculates technical debt score
func (cra *CodeReflectionAgent) calculateTechnicalDebt(content, language string) float64 {
	lines := strings.Split(content, "\n")
	debt := 0.0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count debt indicators
		if strings.Contains(trimmed, "TODO") {
			debt += 1.0
		}
		if strings.Contains(trimmed, "FIXME") {
			debt += 2.0
		}
		if strings.Contains(trimmed, "HACK") {
			debt += 3.0
		}
		if strings.Contains(trimmed, "XXX") {
			debt += 2.0
		}

		// Long lines indicate poor formatting
		if len(line) > 150 {
			debt += 0.5
		}

		// Deeply nested code
		if strings.Count(line, "    ") > 4 || strings.Count(line, "\t") > 3 {
			debt += 0.3
		}
	}

	return debt
}

// createDocumentContent creates rich content for RAG indexing
func (cra *CodeReflectionAgent) createDocumentContent(filePath, content string, knowledge *FileKnowledge) string {
	var doc strings.Builder

	// File metadata
	doc.WriteString(fmt.Sprintf("File: %s\n", filePath))
	doc.WriteString(fmt.Sprintf("Language: %s\n", knowledge.Language))
	doc.WriteString(fmt.Sprintf("Purpose: %s\n", knowledge.Purpose))
	doc.WriteString(fmt.Sprintf("Quality Score: %.2f/10\n", knowledge.QualityScore))
	doc.WriteString(fmt.Sprintf("Complexity Score: %.2f/10\n", knowledge.ComplexityScore))
	doc.WriteString(fmt.Sprintf("Technical Debt: %.2f\n", knowledge.TechnicalDebt))

	// Functions and dependencies
	if len(knowledge.MainFunctions) > 0 {
		doc.WriteString(fmt.Sprintf("Functions: %s\n", strings.Join(knowledge.MainFunctions, ", ")))
	}
	if len(knowledge.Dependencies) > 0 {
		doc.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(knowledge.Dependencies, ", ")))
	}

	doc.WriteString("\nCode Content:\n")
	doc.WriteString(content)

	return doc.String()
}

// Enhanced Start method to include continuous indexing
func (cra *CodeReflectionAgent) StartEnhanced(ctx context.Context) error {
	if err := cra.Start(ctx); err != nil {
		return err
	}

	// Start continuous indexing loop
	go cra.continuousIndexingLoop(ctx)

	// Trigger initial indexing
	go func() {
		time.Sleep(10 * time.Second) // Wait for system to stabilize
		cra.triggerCodebaseIndexing()
	}()

	log.Printf("[CodeReflectionAgent] Enhanced RAG indexing capabilities started")
	return nil
}

// continuousIndexingLoop runs continuous indexing
func (cra *CodeReflectionAgent) continuousIndexingLoop(ctx context.Context) {
	ticker := time.NewTicker(cra.analysisConfig.IndexingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Since(cra.lastIndexingTime) >= cra.analysisConfig.IndexingInterval {
				cra.triggerCodebaseIndexing()
			}
		}
	}
}

// triggerCodebaseIndexing starts indexing the codebase
func (cra *CodeReflectionAgent) triggerCodebaseIndexing() {
	task := &AnalysisTask{
		ID:         fmt.Sprintf("index_codebase_%d", time.Now().Unix()),
		Type:       "index",
		TargetPath: cra.analysisConfig.RootPath,
		Priority:   4,
		CreatedAt:  time.Now(),
		Context: map[string]interface{}{
			"trigger": "scheduled",
		},
	}

	select {
	case cra.analysisQueue <- task:
		log.Printf("[CodeReflectionAgent] Queued codebase indexing")
	default:
		log.Printf("[CodeReflectionAgent] Analysis queue full, skipping indexing")
	}
}

// GetCodebaseKnowledge returns the knowledge about a specific file
func (cra *CodeReflectionAgent) GetCodebaseKnowledge(filePath string) *FileKnowledge {
	knowledge, exists := cra.codebaseKnowledge[filePath]
	if !exists {
		return nil
	}
	return knowledge
}

// QueryCodebase uses RAG to search through the indexed codebase
func (cra *CodeReflectionAgent) QueryCodebase(ctx context.Context, query string) (string, error) {
	if cra.ragSystem == nil {
		return "", fmt.Errorf("RAG system not available")
	}

	// Use RAG system to search through the indexed codebase
	response, err := cra.ragSystem.QueryWithContext(ctx, query, true, 5)
	if err != nil {
		return "", fmt.Errorf("failed to query codebase: %w", err)
	}

	// Add insights to AGI if available
	if cra.IsAGIIntegrationEnabled() {
		cra.ReportInsightToAGI(
			InsightTypeKnowledge,
			"Codebase Query",
			fmt.Sprintf("Searched codebase with query: %s", query),
			map[string]interface{}{
				"query":    query,
				"response": response,
			},
			0.7,
		)
	}

	return response, nil
}

// TriggerSmartAnalysis triggers AI-powered code analysis
func (cra *CodeReflectionAgent) TriggerSmartAnalysis(ctx context.Context) error {
	if cra.modelManager == nil {
		return fmt.Errorf("model manager not available")
	}

	// First, ensure codebase is indexed
	cra.triggerCodebaseIndexing()

	// Wait a moment for indexing to start
	time.Sleep(2 * time.Second)

	// Perform smart analysis using AI and RAG
	issues := cra.GetActiveIssues()
	if len(issues) == 0 {
		log.Printf("[CodeReflectionAgent] No active issues found for smart analysis")
		return nil
	}

	// Group issues by file and severity
	issuesByFile := make(map[string][]*CodeIssue)
	for _, issue := range issues {
		issuesByFile[issue.FilePath] = append(issuesByFile[issue.FilePath], issue)
	}

	// Analyze each file with issues using RAG
	for filePath, fileIssues := range issuesByFile {
		knowledge := cra.GetCodebaseKnowledge(filePath)
		if knowledge == nil {
			continue
		}

		// Create a smart analysis query
		query := cra.buildSmartAnalysisQuery(filePath, fileIssues, knowledge)

		// Query the RAG system for insights
		response, err := cra.QueryCodebase(ctx, query)
		if err != nil {
			log.Printf("[CodeReflectionAgent] Smart analysis failed for %s: %v", filePath, err)
			continue
		}

		// Store the analysis results
		log.Printf("[CodeReflectionAgent] Smart analysis for %s: %s", filePath, response)

		// Create improvement suggestions based on AI response
		cra.processSmartAnalysisResponse(filePath, response, fileIssues)
	}

	return nil
}

// buildSmartAnalysisQuery builds an intelligent query for code analysis
func (cra *CodeReflectionAgent) buildSmartAnalysisQuery(filePath string, issues []*CodeIssue, knowledge *FileKnowledge) string {
	var query strings.Builder

	query.WriteString(fmt.Sprintf("Analyze the file %s with the following characteristics:\n", filePath))
	query.WriteString(fmt.Sprintf("- Language: %s\n", knowledge.Language))
	query.WriteString(fmt.Sprintf("- Purpose: %s\n", knowledge.Purpose))
	query.WriteString(fmt.Sprintf("- Quality Score: %.2f/10\n", knowledge.QualityScore))
	query.WriteString(fmt.Sprintf("- Complexity Score: %.2f/10\n", knowledge.ComplexityScore))
	query.WriteString(fmt.Sprintf("- Technical Debt: %.2f\n", knowledge.TechnicalDebt))

	if len(knowledge.MainFunctions) > 0 {
		query.WriteString(fmt.Sprintf("- Main Functions: %s\n", strings.Join(knowledge.MainFunctions, ", ")))
	}

	query.WriteString("\nCurrent Issues:\n")
	for _, issue := range issues {
		query.WriteString(fmt.Sprintf("- %s (Line %d): %s\n", issue.Type, issue.Line, issue.Description))
	}

	query.WriteString("\nPlease provide specific recommendations for improving this code, addressing the issues, and enhancing overall quality.")

	return query.String()
}

// processSmartAnalysisResponse processes AI analysis response and creates improvements
func (cra *CodeReflectionAgent) processSmartAnalysisResponse(filePath, response string, issues []*CodeIssue) {
	// Create a code improvement based on the AI response
	improvement := &CodeImprovement{
		ID:          fmt.Sprintf("smart_improvement_%s_%d", strings.ReplaceAll(filePath, "/", "_"), time.Now().Unix()),
		Type:        "refactor",
		Title:       fmt.Sprintf("Smart Analysis Improvement for %s", filepath.Base(filePath)),
		Description: response,
		FilePath:    filePath,
		Priority:    cra.calculateImprovementPriority(issues),
		Confidence:  0.7, // AI-generated improvements have moderate confidence
		Applied:     false,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"source":       "smart_analysis",
			"issues_count": len(issues),
			"ai_generated": true,
		},
	}

	// Add to improvement history
	cra.improvementHistory = append(cra.improvementHistory, improvement)

	log.Printf("[CodeReflectionAgent] Created smart improvement: %s", improvement.Title)
}

// calculateImprovementPriority calculates priority based on issues
func (cra *CodeReflectionAgent) calculateImprovementPriority(issues []*CodeIssue) int {
	priority := 1

	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			priority += 4
		case "high":
			priority += 3
		case "medium":
			priority += 2
		case "low":
			priority += 1
		}
	}

	// Cap at 10
	if priority > 10 {
		priority = 10
	}

	return priority
}

// Enhanced query handlers for new API capabilities

// handleIndexingQuery handles codebase indexing requests
func (cra *CodeReflectionAgent) handleIndexingQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	// Trigger codebase indexing
	cra.triggerCodebaseIndexing()

	return &types.Response{
		QueryID:   query.ID,
		Text:      "Codebase indexing has been triggered. The entire codebase will be indexed into the RAG vector database for intelligent code analysis.",
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"operation": "indexing_triggered",
			"target":    cra.analysisConfig.RootPath,
		},
	}, nil
}

// handleCodebaseQuery handles queries to search through the indexed codebase
func (cra *CodeReflectionAgent) handleCodebaseQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	// Use the QueryCodebase method to search through indexed code
	result, err := cra.QueryCodebase(ctx, query.Text)
	if err != nil {
		return &types.Response{
			QueryID:   query.ID,
			Text:      fmt.Sprintf("Failed to query codebase: %v", err),
			Status:    "error",
			Timestamp: time.Now().Unix(),
		}, err
	}

	return &types.Response{
		QueryID:   query.ID,
		Text:      fmt.Sprintf("Codebase search results for '%s':\n\n%s", query.Text, result),
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"operation":     "codebase_search",
			"query":         query.Text,
			"indexed_files": fmt.Sprintf("%d", cra.codeQualityMetrics.IndexedFiles),
		},
	}, nil
}

// handleSmartAnalysisQuery handles smart AI-powered analysis requests
func (cra *CodeReflectionAgent) handleSmartAnalysisQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	// Trigger smart analysis
	err := cra.TriggerSmartAnalysis(ctx)
	if err != nil {
		return &types.Response{
			QueryID:   query.ID,
			Text:      fmt.Sprintf("Failed to trigger smart analysis: %v", err),
			Status:    "error",
			Timestamp: time.Now().Unix(),
		}, err
	}

	activeIssues := len(cra.GetActiveIssues())
	improvements := len(cra.GetImprovementHistory())

	return &types.Response{
		QueryID:   query.ID,
		Text:      fmt.Sprintf("Smart analysis completed! Analyzed %d active issues and generated AI-powered improvement suggestions. Check the improvements endpoint for detailed recommendations.", activeIssues),
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"operation":          "smart_analysis",
			"active_issues":      fmt.Sprintf("%d", activeIssues),
			"total_improvements": fmt.Sprintf("%d", improvements),
		},
	}, nil
}

// handleFileKnowledgeQuery handles requests for specific file knowledge
func (cra *CodeReflectionAgent) handleFileKnowledgeQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	filePath := query.Metadata["file_path"]
	if filePath == "" {
		return &types.Response{
			QueryID:   query.ID,
			Text:      "No file path specified for knowledge query",
			Status:    "error",
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("file path required")
	}

	knowledge := cra.GetCodebaseKnowledge(filePath)
	if knowledge == nil {
		return &types.Response{
			QueryID:   query.ID,
			Text:      fmt.Sprintf("No knowledge found for file: %s. The file may not be indexed yet.", filePath),
			Status:    "not_found",
			Timestamp: time.Now().Unix(),
		}, nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("Knowledge for file: %s\n\n", filePath))
	response.WriteString(fmt.Sprintf("Language: %s\n", knowledge.Language))
	response.WriteString(fmt.Sprintf("Purpose: %s\n", knowledge.Purpose))
	response.WriteString(fmt.Sprintf("Quality Score: %.2f/10\n", knowledge.QualityScore))
	response.WriteString(fmt.Sprintf("Complexity Score: %.2f/10\n", knowledge.ComplexityScore))
	response.WriteString(fmt.Sprintf("Technical Debt: %.2f\n", knowledge.TechnicalDebt))

	if len(knowledge.MainFunctions) > 0 {
		response.WriteString(fmt.Sprintf("Main Functions: %s\n", strings.Join(knowledge.MainFunctions, ", ")))
	}

	if len(knowledge.Dependencies) > 0 {
		response.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(knowledge.Dependencies, ", ")))
	}

	if len(knowledge.KnownIssues) > 0 {
		response.WriteString(fmt.Sprintf("Known Issues: %s\n", strings.Join(knowledge.KnownIssues, ", ")))
	}

	return &types.Response{
		QueryID:   query.ID,
		Text:      response.String(),
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"file_path":        filePath,
			"language":         knowledge.Language,
			"quality_score":    fmt.Sprintf("%.2f", knowledge.QualityScore),
			"complexity_score": fmt.Sprintf("%.2f", knowledge.ComplexityScore),
		},
	}, nil
}

// handleGetImprovementsQuery handles requests for improvement suggestions
func (cra *CodeReflectionAgent) handleGetImprovementsQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	improvements := cra.GetImprovementHistory()

	if len(improvements) == 0 {
		return &types.Response{
			QueryID:   query.ID,
			Text:      "No improvement suggestions available yet. Run smart analysis to generate AI-powered improvement recommendations.",
			Status:    "success",
			Timestamp: time.Now().Unix(),
			Metadata: map[string]string{
				"improvements_count": "0",
			},
		}, nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("Found %d improvement suggestions:\n\n", len(improvements)))

	// Sort improvements by priority (highest first)
	sortedImprovements := make([]*CodeImprovement, len(improvements))
	copy(sortedImprovements, improvements)
	sort.Slice(sortedImprovements, func(i, j int) bool {
		return sortedImprovements[i].Priority > sortedImprovements[j].Priority
	})

	// Show top 10 improvements
	maxShow := 10
	if len(sortedImprovements) < maxShow {
		maxShow = len(sortedImprovements)
	}

	for i := 0; i < maxShow; i++ {
		improvement := sortedImprovements[i]
		status := "Pending"
		if improvement.Applied {
			status = "Applied"
		}

		response.WriteString(fmt.Sprintf("%d. %s (Priority: %d, Confidence: %.2f, Status: %s)\n",
			i+1, improvement.Title, improvement.Priority, improvement.Confidence, status))
		response.WriteString(fmt.Sprintf("   File: %s\n", improvement.FilePath))
		response.WriteString(fmt.Sprintf("   Type: %s\n", improvement.Type))

		// Truncate description if too long
		description := improvement.Description
		if len(description) > 200 {
			description = description[:200] + "..."
		}
		response.WriteString(fmt.Sprintf("   Description: %s\n\n", description))
	}

	if len(improvements) > maxShow {
		response.WriteString(fmt.Sprintf("... and %d more improvements available.\n", len(improvements)-maxShow))
	}

	return &types.Response{
		QueryID:   query.ID,
		Text:      response.String(),
		Status:    "success",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"improvements_count": fmt.Sprintf("%d", len(improvements)),
			"showing":            fmt.Sprintf("%d", maxShow),
		},
	}, nil
}
