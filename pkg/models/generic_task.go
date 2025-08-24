package models

import (
	"encoding/json"
	"time"
	
	"github.com/google/uuid"
)

// ExecutorType represents the type of executor (ollama, openai, script, etc.)
type ExecutorType string

// Common executor types
const (
	ExecutorTypeOllama ExecutorType = "ollama"
	ExecutorTypeOpenAI ExecutorType = "openai"
	ExecutorTypeScript ExecutorType = "script"
	ExecutorTypeHTTP   ExecutorType = "http"
)

// ExecutorAction represents the action within an executor
type ExecutorAction string

// Common executor actions
const (
	ActionChat     ExecutorAction = "chat"
	ActionGenerate ExecutorAction = "generate"
	ActionEmbed    ExecutorAction = "embed"
	ActionExecute  ExecutorAction = "execute"
	ActionRequest  ExecutorAction = "request"
)

// GenericTask represents a generic task that can be executed by any executor
type GenericTask struct {
	ID           string                 `json:"id"`
	ExecutorType ExecutorType           `json:"executor_type"`  // Which executor to use
	Action       ExecutorAction         `json:"action"`         // What action to perform
	Payload      map[string]interface{} `json:"payload"`        // Generic payload data
	Model        string                 `json:"model,omitempty"`// Optional model identifier
	Priority     Priority               `json:"priority"`
	Status       TaskStatus             `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	ExecutedOn   string                 `json:"executed_on,omitempty"`
	Result       interface{}            `json:"result,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Retries      int                    `json:"retries"`
	ScheduledAt  *time.Time             `json:"scheduled_at,omitempty"`
}

// TaskAdapter provides methods to work with both legacy and generic tasks
type TaskAdapter struct {
}

// NewTaskAdapter creates a new task adapter
func NewTaskAdapter() *TaskAdapter {
	return &TaskAdapter{}
}

// ConvertToGeneric converts a legacy Task to GenericTask
func (ta *TaskAdapter) ConvertToGeneric(legacyTask *Task) (*GenericTask, error) {
	genericTask := &GenericTask{
		ID:          legacyTask.ID,
		Priority:    legacyTask.Priority,
		Status:      legacyTask.Status,
		CreatedAt:   legacyTask.CreatedAt,
		StartedAt:   legacyTask.StartedAt,
		CompletedAt: legacyTask.CompletedAt,
		ExecutedOn:  legacyTask.ExecutedOn,
		Result:      legacyTask.Result,
		Error:       legacyTask.Error,
		Retries:     legacyTask.RetryCount,
		ScheduledAt: legacyTask.ScheduledAt,
		Model:       legacyTask.Model,
	}

	// Map legacy task types to generic executor + action
	switch legacyTask.Type {
	case TaskTypeChat:
		genericTask.ExecutorType = ExecutorTypeOllama
		genericTask.Action = ActionChat
	case TaskTypeGenerate:
		genericTask.ExecutorType = ExecutorTypeOllama
		genericTask.Action = ActionGenerate
	case TaskTypeEmbed:
		genericTask.ExecutorType = ExecutorTypeOllama
		genericTask.Action = ActionEmbed
	default:
		// For unknown types, try to infer from payload or use generic executor
		genericTask.ExecutorType = ExecutorTypeScript
		genericTask.Action = ActionExecute
	}

	// Convert payload
	payload := make(map[string]interface{})
	
	// Handle different payload types
	switch p := legacyTask.Payload.(type) {
	case map[string]interface{}:
		payload = p
	case ChatTaskPayload:
		payload = ta.chatPayloadToMap(p)
	case GenerateTaskPayload:
		payload = ta.generatePayloadToMap(p)
	case EmbedTaskPayload:
		payload = ta.embedPayloadToMap(p)
	default:
		// Try JSON marshaling/unmarshaling for other types
		data, err := json.Marshal(legacyTask.Payload)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, err
		}
	}
	
	genericTask.Payload = payload
	return genericTask, nil
}

// ConvertToLegacy converts a GenericTask to legacy Task format
func (ta *TaskAdapter) ConvertToLegacy(genericTask *GenericTask) (*Task, error) {
	legacyTask := &Task{
		ID:          genericTask.ID,
		Priority:    genericTask.Priority,
		Status:      genericTask.Status,
		CreatedAt:   genericTask.CreatedAt,
		StartedAt:   genericTask.StartedAt,
		CompletedAt: genericTask.CompletedAt,
		ExecutedOn:  genericTask.ExecutedOn,
		Result:      genericTask.Result,
		Error:       genericTask.Error,
		RetryCount:  genericTask.Retries,
		ScheduledAt: genericTask.ScheduledAt,
		Model:       genericTask.Model,
		Payload:     genericTask.Payload,
	}

	// Map generic executor + action back to legacy task types
	if genericTask.ExecutorType == ExecutorTypeOllama {
		switch genericTask.Action {
		case ActionChat:
			legacyTask.Type = TaskTypeChat
		case ActionGenerate:
			legacyTask.Type = TaskTypeGenerate
		case ActionEmbed:
			legacyTask.Type = TaskTypeEmbed
		default:
			legacyTask.Type = TaskTypeGenerate // Default fallback
		}
	} else {
		// For non-Ollama executors, map to the closest legacy type
		switch genericTask.Action {
		case ActionChat:
			legacyTask.Type = TaskTypeChat
		case ActionEmbed:
			legacyTask.Type = TaskTypeEmbed
		default:
			legacyTask.Type = TaskTypeGenerate
		}
	}

	return legacyTask, nil
}

// Helper methods to convert specific payload types

func (ta *TaskAdapter) chatPayloadToMap(payload ChatTaskPayload) map[string]interface{} {
	result := make(map[string]interface{})
	result["messages"] = payload.Messages
	if payload.System != "" {
		result["system"] = payload.System
	}
	result["stream"] = payload.Stream
	return result
}

func (ta *TaskAdapter) generatePayloadToMap(payload GenerateTaskPayload) map[string]interface{} {
	result := make(map[string]interface{})
	result["prompt"] = payload.Prompt
	if payload.System != "" {
		result["system"] = payload.System
	}
	if payload.Template != "" {
		result["template"] = payload.Template
	}
	result["raw"] = payload.Raw
	result["stream"] = payload.Stream
	return result
}

func (ta *TaskAdapter) embedPayloadToMap(payload EmbedTaskPayload) map[string]interface{} {
	result := make(map[string]interface{})
	result["input"] = payload.Input
	result["truncate"] = payload.Truncate
	return result
}

// CreateGenericTask creates a new generic task
func CreateGenericTask(executorType ExecutorType, action ExecutorAction, payload map[string]interface{}) *GenericTask {
	return &GenericTask{
		ID:           GenerateID(),
		ExecutorType: executorType,
		Action:       action,
		Payload:      payload,
		Priority:     PriorityNormal,
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		Retries:      0,
	}
}

// CreateOllamaTask creates a new Ollama-based task (backward compatibility helper)
func CreateOllamaTask(action ExecutorAction, model string, payload map[string]interface{}) *GenericTask {
	task := CreateGenericTask(ExecutorTypeOllama, action, payload)
	task.Model = model
	return task
}

// IsGenericSupported checks if a task can be executed as a generic task
func (gt *GenericTask) IsGenericSupported() bool {
	return gt.ExecutorType != "" && gt.Action != ""
}

// Clone creates a deep copy of the generic task
func (gt *GenericTask) Clone() *GenericTask {
	data, _ := json.Marshal(gt)
	var clone GenericTask
	json.Unmarshal(data, &clone)
	return &clone
}

// GenericTaskFilter represents filters for querying generic tasks
type GenericTaskFilter struct {
	ExecutorTypes []ExecutorType   `json:"executor_types,omitempty"`
	Actions       []ExecutorAction `json:"actions,omitempty"`
	Status        []TaskStatus     `json:"status,omitempty"`
	Priority      []Priority       `json:"priority,omitempty"`
	Limit         int              `json:"limit,omitempty"`
}

// GenerateID generates a new unique ID using UUID
func GenerateID() string {
	return uuid.New().String()
}