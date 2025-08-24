package models

import (
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

// Priority represents task priority levels
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityNormal   Priority = 5
	PriorityHigh     Priority = 10
	PriorityCritical Priority = 15
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
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

// ChatMessage represents a single chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

// CreateOllamaTask creates a new Ollama-based task (helper function)
func CreateOllamaTask(action ExecutorAction, model string, payload map[string]interface{}) *GenericTask {
	task := CreateGenericTask(ExecutorTypeOllama, action, payload)
	task.Model = model
	return task
}

// Clone creates a deep copy of the generic task
func (gt *GenericTask) Clone() *GenericTask {
	newTask := *gt
	
	// Deep copy payload
	newPayload := make(map[string]interface{})
	for k, v := range gt.Payload {
		newPayload[k] = v
	}
	newTask.Payload = newPayload
	
	return &newTask
}

// GenerateID generates a new unique ID using UUID
func GenerateID() string {
	return uuid.New().String()
}