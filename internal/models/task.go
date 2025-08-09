package models

import (
	"context"
	"time"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeChat     TaskType = "chat"
	TaskTypeGenerate TaskType = "generate"
	TaskTypeEmbed    TaskType = "embed"
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

// Task represents a task in the queue
type Task struct {
	ID          string                 `json:"id"`
	Type        TaskType              `json:"type"`
	Priority    Priority              `json:"priority"`
	Status      TaskStatus            `json:"status"`
	Model       string                `json:"model"`
	Payload     any                   `json:"payload"`
	Options     map[string]any        `json:"options"`
	CreatedAt   time.Time             `json:"created_at"`
	ScheduledAt *time.Time            `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time            `json:"started_at,omitempty"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Error       string                `json:"error,omitempty"`
	Result      any                   `json:"result,omitempty"`
	RetryCount  int                   `json:"retry_count"`
	MaxRetries  int                   `json:"max_retries"`
	RemoteExecution bool              `json:"remote_execution,omitempty"`
	ExecutedOn      string            `json:"executed_on,omitempty"`
	Context     context.Context       `json:"-"`
	CancelFunc  context.CancelFunc    `json:"-"`
}

// TaskResult represents the result of a completed task
type TaskResult struct {
	TaskID  string      `json:"task_id"`
	Success bool        `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// StreamChunk represents a chunk of streamed data
type StreamChunk struct {
	Data  string `json:"data"`
	Error error  `json:"error,omitempty"`
	Done  bool   `json:"done"`
}

// TaskEvent represents an event in the task lifecycle
type TaskEvent struct {
	TaskID    string     `json:"task_id"`
	Type      string     `json:"type"`
	Status    TaskStatus `json:"status"`
	Timestamp time.Time  `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

// TaskFilter for querying tasks
type TaskFilter struct {
	Status   []TaskStatus `json:"status,omitempty"`
	Type     []TaskType   `json:"type,omitempty"`
	Priority []Priority   `json:"priority,omitempty"`
	Limit    int          `json:"limit,omitempty"`
	Offset   int          `json:"offset,omitempty"`
}

// QueueStats provides statistics about the queue
type QueueStats struct {
	PendingTasks   int            `json:"pending_tasks"`
	RunningTasks   int            `json:"running_tasks"`
	CompletedTasks int            `json:"completed_tasks"`
	FailedTasks    int            `json:"failed_tasks"`
	CancelledTasks int            `json:"cancelled_tasks"`
	TotalTasks     int            `json:"total_tasks"`
	QueuesByPriority map[Priority]int `json:"queues_by_priority"`
	WorkersActive  int            `json:"workers_active"`
	WorkersIdle    int            `json:"workers_idle"`
}