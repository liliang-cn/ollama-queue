package executor

import (
	"context"
	
	"github.com/liliang-cn/ollama-queue/pkg/models"
)

// ExecutorType represents the type of executor (ollama, openai, script, etc.)
type ExecutorType = models.ExecutorType

// ExecutorAction represents the action within an executor (chat, generate, embed, execute, etc.)  
type ExecutorAction = models.ExecutorAction

// GenericTask represents a generic task structure
type GenericTask = models.GenericTask

// ExecutorMetadata contains information about an executor's capabilities
type ExecutorMetadata struct {
	Name        string           `json:"name"`
	Type        ExecutorType     `json:"type"`
	Description string           `json:"description"`
	Version     string           `json:"version"`
	Actions     []ExecutorAction `json:"actions"`     // Supported actions
	Config      map[string]any   `json:"config"`      // Configuration schema
}

// TaskResult represents the result of task execution
type TaskResult struct {
	TaskID  string `json:"task_id"`
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// StreamChunk represents a chunk of streaming data
type StreamChunk struct {
	Data  string `json:"data,omitempty"`
	Error error  `json:"error,omitempty"`  
	Done  bool   `json:"done"`
}

// GenericExecutor interface defines the generic executor operations
type GenericExecutor interface {
	// GetMetadata returns information about this executor
	GetMetadata() *ExecutorMetadata
	
	// CanExecute checks if the executor can handle a specific action
	CanExecute(action ExecutorAction) bool
	
	// Execute executes a task synchronously
	Execute(ctx context.Context, task *GenericTask) (*TaskResult, error)
	
	// ExecuteStream executes a task with streaming output
	ExecuteStream(ctx context.Context, task *GenericTask) (<-chan *StreamChunk, error)
	
	// Validate validates that a task payload is correct for this executor
	Validate(action ExecutorAction, payload map[string]any) error
	
	// Close cleans up executor resources
	Close() error
}

// ExecutorRegistry manages registered executors
type ExecutorRegistry struct {
	executors map[ExecutorType]GenericExecutor
}

// NewExecutorRegistry creates a new executor registry
func NewExecutorRegistry() *ExecutorRegistry {
	return &ExecutorRegistry{
		executors: make(map[ExecutorType]GenericExecutor),
	}
}

// Register registers an executor with the registry
func (r *ExecutorRegistry) Register(executorType ExecutorType, executor GenericExecutor) error {
	if executor == nil {
		return ErrNilExecutor
	}
	
	r.executors[executorType] = executor
	return nil
}

// Unregister removes an executor from the registry
func (r *ExecutorRegistry) Unregister(executorType ExecutorType) {
	delete(r.executors, executorType)
}

// GetExecutor retrieves an executor by type
func (r *ExecutorRegistry) GetExecutor(executorType ExecutorType) (GenericExecutor, bool) {
	executor, exists := r.executors[executorType]
	return executor, exists
}

// ListExecutors returns all registered executors metadata
func (r *ExecutorRegistry) ListExecutors() []*ExecutorMetadata {
	var metadata []*ExecutorMetadata
	for _, executor := range r.executors {
		metadata = append(metadata, executor.GetMetadata())
	}
	return metadata
}

// CanExecuteTask checks if any registered executor can handle the task
func (r *ExecutorRegistry) CanExecuteTask(task *GenericTask) bool {
	executor, exists := r.executors[task.ExecutorType]
	if !exists {
		return false
	}
	return executor.CanExecute(task.Action)
}

// ExecuteTask executes a task using the appropriate executor
func (r *ExecutorRegistry) ExecuteTask(ctx context.Context, task *GenericTask) (*TaskResult, error) {
	executor, exists := r.executors[task.ExecutorType]
	if !exists {
		return nil, ErrExecutorNotFound
	}
	
	if !executor.CanExecute(task.Action) {
		return nil, ErrActionNotSupported
	}
	
	// Validate task payload
	if err := executor.Validate(task.Action, task.Payload); err != nil {
		return nil, err
	}
	
	return executor.Execute(ctx, task)
}

// ExecuteTaskStream executes a task with streaming using the appropriate executor
func (r *ExecutorRegistry) ExecuteTaskStream(ctx context.Context, task *GenericTask) (<-chan *StreamChunk, error) {
	executor, exists := r.executors[task.ExecutorType]
	if !exists {
		return nil, ErrExecutorNotFound
	}
	
	if !executor.CanExecute(task.Action) {
		return nil, ErrActionNotSupported
	}
	
	// Validate task payload
	if err := executor.Validate(task.Action, task.Payload); err != nil {
		return nil, err
	}
	
	return executor.ExecuteStream(ctx, task)
}

// Close closes all registered executors
func (r *ExecutorRegistry) Close() error {
	var lastErr error
	for _, executor := range r.executors {
		if err := executor.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}