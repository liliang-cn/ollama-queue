// Package queue provides a task queue management system for Ollama models
package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/ollama-queue/pkg/models"
)

// QueueManagerInterface defines the public API for the queue manager
type QueueManagerInterface interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop() error
	Close() error

	// Task submission
	SubmitTask(task *models.Task) (string, error)
	SubmitTaskWithCallback(task *models.Task, callback models.TaskCallback) (string, error)
	SubmitTaskWithChannel(task *models.Task, resultChan chan *models.TaskResult) (string, error)
	SubmitStreamingTask(task *models.Task) (<-chan *models.StreamChunk, error)

	// Task management
	GetTask(taskID string) (*models.Task, error)
	GetTaskStatus(taskID string) (models.TaskStatus, error)
	CancelTask(taskID string) error
	UpdateTaskPriority(taskID string, priority models.Priority) error

	// Query operations
	ListTasks(filter models.TaskFilter) ([]*models.Task, error)
	GetQueueStats() (*models.QueueStats, error)

	// Event monitoring
	Subscribe(eventTypes []string) (<-chan *models.TaskEvent, error)
	Unsubscribe(eventChan <-chan *models.TaskEvent) error
}

// Option defines a configuration option function
type Option func(*models.Config)

// WithOllamaHost sets the Ollama host URL
func WithOllamaHost(host string) Option {
	return func(config *models.Config) {
		config.OllamaHost = host
	}
}

// WithMaxWorkers sets the maximum number of worker goroutines
func WithMaxWorkers(workers int) Option {
	return func(config *models.Config) {
		config.MaxWorkers = workers
	}
}

// WithStoragePath sets the storage path for the BadgerDB database
func WithStoragePath(path string) Option {
	return func(config *models.Config) {
		config.StoragePath = path
	}
}

// WithOllamaTimeout sets the timeout for Ollama operations
func WithOllamaTimeout(timeout time.Duration) Option {
	return func(config *models.Config) {
		config.OllamaTimeout = timeout
	}
}

// WithCleanupInterval sets the interval for cleaning up old tasks
func WithCleanupInterval(interval time.Duration) Option {
	return func(config *models.Config) {
		config.CleanupInterval = interval
	}
}

// WithMaxCompletedTasks sets the maximum number of completed tasks to keep
func WithMaxCompletedTasks(maxTasks int) Option {
	return func(config *models.Config) {
		config.MaxCompletedTasks = maxTasks
	}
}

// WithSchedulingInterval sets the interval for task scheduling
func WithSchedulingInterval(interval time.Duration) Option {
	return func(config *models.Config) {
		config.SchedulingInterval = interval
	}
}

// WithRetryConfig sets the retry configuration
func WithRetryConfig(retryConfig models.RetryConfig) Option {
	return func(config *models.Config) {
		config.RetryConfig = retryConfig
	}
}

// WithLogLevel sets the logging level
func WithLogLevel(level string) Option {
	return func(config *models.Config) {
		config.LogLevel = level
	}
}

// WithLogFile sets the log file path
func WithLogFile(file string) Option {
	return func(config *models.Config) {
		config.LogFile = file
	}
}

// NewQueueManagerWithOptions creates a new queue manager with options
func NewQueueManagerWithOptions(options ...Option) (*QueueManager, error) {
	config := models.DefaultConfig()
	
	for _, option := range options {
		option(config)
	}
	
	return NewQueueManager(config)
}

// Convenience functions for creating different types of tasks

// NewChatTask creates a new chat task
func NewChatTask(model string, messages []models.ChatMessage, options ...TaskOption) *models.Task {
	task := &models.Task{
		Type:     models.TaskTypeChat,
		Priority: models.PriorityNormal,
		Model:    model,
		Payload: models.ChatTaskPayload{
			Messages: messages,
		},
		Options: make(map[string]any),
	}
	
	for _, option := range options {
		option(task)
	}
	
	return task
}

// NewGenerateTask creates a new generation task
func NewGenerateTask(model string, prompt string, options ...TaskOption) *models.Task {
	task := &models.Task{
		Type:     models.TaskTypeGenerate,
		Priority: models.PriorityNormal,
		Model:    model,
		Payload: models.GenerateTaskPayload{
			Prompt: prompt,
		},
		Options: make(map[string]any),
	}
	
	for _, option := range options {
		option(task)
	}
	
	return task
}

// NewEmbedTask creates a new embedding task
func NewEmbedTask(model string, input any, options ...TaskOption) *models.Task {
	task := &models.Task{
		Type:     models.TaskTypeEmbed,
		Priority: models.PriorityNormal,
		Model:    model,
		Payload: models.EmbedTaskPayload{
			Input: input,
		},
		Options: make(map[string]any),
	}
	
	for _, option := range options {
		option(task)
	}
	
	return task
}

// TaskOption defines a task configuration option function
type TaskOption func(*models.Task)

// WithTaskPriority sets the task priority
func WithTaskPriority(priority models.Priority) TaskOption {
	return func(task *models.Task) {
		task.Priority = priority
	}
}

// WithTaskMaxRetries sets the maximum number of retries for a task
func WithTaskMaxRetries(maxRetries int) TaskOption {
	return func(task *models.Task) {
		task.MaxRetries = maxRetries
	}
}

// WithTaskScheduledAt sets the scheduled execution time for a task
func WithTaskScheduledAt(scheduledAt time.Time) TaskOption {
	return func(task *models.Task) {
		task.ScheduledAt = &scheduledAt
	}
}

// WithChatSystem sets the system message for chat tasks
func WithChatSystem(system string) TaskOption {
	return func(task *models.Task) {
		if payload, ok := task.Payload.(models.ChatTaskPayload); ok {
			payload.System = system
			task.Payload = payload
		}
	}
}

// WithChatStreaming enables streaming for chat tasks
func WithChatStreaming(stream bool) TaskOption {
	return func(task *models.Task) {
		if payload, ok := task.Payload.(models.ChatTaskPayload); ok {
			payload.Stream = stream
			task.Payload = payload
		}
	}
}

// WithGenerateSystem sets the system message for generation tasks
func WithGenerateSystem(system string) TaskOption {
	return func(task *models.Task) {
		if payload, ok := task.Payload.(models.GenerateTaskPayload); ok {
			payload.System = system
			task.Payload = payload
		}
	}
}

// WithGenerateTemplate sets the template for generation tasks
func WithGenerateTemplate(template string) TaskOption {
	return func(task *models.Task) {
		if payload, ok := task.Payload.(models.GenerateTaskPayload); ok {
			payload.Template = template
			task.Payload = payload
		}
	}
}

// WithGenerateRaw sets the raw mode for generation tasks
func WithGenerateRaw(raw bool) TaskOption {
	return func(task *models.Task) {
		if payload, ok := task.Payload.(models.GenerateTaskPayload); ok {
			payload.Raw = raw
			task.Payload = payload
		}
	}
}

// WithGenerateStreaming enables streaming for generation tasks
func WithGenerateStreaming(stream bool) TaskOption {
	return func(task *models.Task) {
		if payload, ok := task.Payload.(models.GenerateTaskPayload); ok {
			payload.Stream = stream
			task.Payload = payload
		}
	}
}

// WithEmbedTruncate sets the truncate option for embedding tasks
func WithEmbedTruncate(truncate bool) TaskOption {
	return func(task *models.Task) {
		if payload, ok := task.Payload.(models.EmbedTaskPayload); ok {
			payload.Truncate = truncate
			task.Payload = payload
		}
	}
}

// Batch task processing functions

// SubmitBatchTasks submits multiple tasks and waits for all to complete
func (qm *QueueManager) SubmitBatchTasks(tasks []*models.Task) ([]*models.TaskResult, error) {
	if len(tasks) == 0 {
		return []*models.TaskResult{}, nil
	}
	
	results := make([]*models.TaskResult, len(tasks))
	resultChans := make([]chan *models.TaskResult, len(tasks))
	
	// Submit all tasks
	for i, task := range tasks {
		resultChans[i] = make(chan *models.TaskResult, 1)
		_, err := qm.SubmitTaskWithChannel(task, resultChans[i])
		if err != nil {
			return nil, fmt.Errorf("failed to submit task %d: %w", i, err)
		}
	}
	
	// Wait for all results
	for i, resultChan := range resultChans {
		results[i] = <-resultChan
		close(resultChan)
	}
	
	return results, nil
}

// SubmitBatchTasksAsync submits multiple tasks and calls callback when all complete
func (qm *QueueManager) SubmitBatchTasksAsync(tasks []*models.Task, callback models.BatchCallback) ([]string, error) {
	if len(tasks) == 0 {
		go callback([]*models.TaskResult{})
		return []string{}, nil
	}
	
	taskIDs := make([]string, len(tasks))
	results := make([]*models.TaskResult, len(tasks))
	completed := make([]bool, len(tasks))
	var completedCount int
	var mu sync.Mutex
	
	for i, task := range tasks {
		taskID, err := qm.SubmitTaskWithCallback(task, func(result *models.TaskResult) {
			mu.Lock()
			defer mu.Unlock()
			
			// Find the task index
			for j, id := range taskIDs {
				if id == result.TaskID {
					results[j] = result
					completed[j] = true
					completedCount++
					break
				}
			}
			
			// Check if all tasks are completed
			if completedCount == len(tasks) {
				callback(results)
			}
		})
		
		if err != nil {
			return nil, fmt.Errorf("failed to submit task %d: %w", i, err)
		}
		
		taskIDs[i] = taskID
	}
	
	return taskIDs, nil
}