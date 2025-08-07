package consciousness

import (
	"github.com/loreum-org/cortex/internal/consciousness/types"
)

// Re-export types from the types package for backward compatibility
type BaseDomain = types.BaseDomain
type DomainType = types.DomainType
type DomainState = types.DomainState
type DomainConfig = types.DomainConfig
type DomainMetrics = types.DomainMetrics
type DomainInput = types.DomainInput
type DomainOutput = types.DomainOutput
type DomainTask = types.DomainTask
type DomainResult = types.DomainResult
type DomainFeedback = types.DomainFeedback
type InterDomainMessage = types.InterDomainMessage
type DomainCoordinator = types.DomainCoordinator
type DomainGoal = types.DomainGoal
type DomainInsight = types.DomainInsight
type DomainImpact = types.DomainImpact
type TaskStatus = types.TaskStatus
type FeedbackType = types.FeedbackType
type MessageType = types.MessageType

// Re-export constants
const (
	DomainReasoning      = types.DomainReasoning
	DomainLearning       = types.DomainLearning
	DomainCreative       = types.DomainCreative
	DomainCommunication  = types.DomainCommunication
	DomainTechnical      = types.DomainTechnical
	DomainMemory         = types.DomainMemory
	DomainMetaCognitive  = types.DomainMetaCognitive
	DomainEmotional      = types.DomainEmotional

	TaskStatusPending    = types.TaskStatusPending
	TaskStatusRunning    = types.TaskStatusRunning
	TaskStatusCompleted  = types.TaskStatusCompleted
	TaskStatusFailed     = types.TaskStatusFailed
	TaskStatusCancelled  = types.TaskStatusCancelled
	TaskStatusTimedOut   = types.TaskStatusTimedOut

	FeedbackTypeQuality     = types.FeedbackTypeQuality
	FeedbackTypeAccuracy    = types.FeedbackTypeAccuracy
	FeedbackTypeRelevance   = types.FeedbackTypeRelevance
	FeedbackTypeCreativity  = types.FeedbackTypeCreativity
	FeedbackTypeEfficiency  = types.FeedbackTypeEfficiency
	FeedbackTypeUser        = types.FeedbackTypeUser
	FeedbackTypeSystem      = types.FeedbackTypeSystem
	FeedbackTypePeer        = types.FeedbackTypePeer

	MessageTypeQuery        = types.MessageTypeQuery
	MessageTypeResponse     = types.MessageTypeResponse
	MessageTypeInsight      = types.MessageTypeInsight
	MessageTypePattern      = types.MessageTypePattern
	MessageTypeCoordination = types.MessageTypeCoordination
	MessageTypeAlert        = types.MessageTypeAlert
	MessageTypeBroadcast    = types.MessageTypeBroadcast
)

// Re-export constructor functions
var NewBaseDomain = types.NewBaseDomain
var NewDomainCoordinator = types.NewDomainCoordinator