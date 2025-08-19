package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, "http://localhost:11434", config.OllamaHost)
	assert.Equal(t, 5*time.Minute, config.OllamaTimeout)
	assert.Equal(t, 4, config.MaxWorkers)
	assert.Equal(t, "./data", config.StoragePath)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
	assert.Equal(t, 1000, config.MaxCompletedTasks)
	assert.Equal(t, 1*time.Second, config.SchedulingInterval)
	assert.Equal(t, 10, config.BatchSize)
	assert.Equal(t, 3, config.RetryConfig.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryConfig.InitialDelay)
	assert.Equal(t, 30*time.Second, config.RetryConfig.MaxDelay)
	assert.Equal(t, 2.0, config.RetryConfig.BackoffFactor)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, "", config.LogFile)
}

func TestTaskTypes(t *testing.T) {
	assert.Equal(t, TaskType("chat"), TaskTypeChat)
	assert.Equal(t, TaskType("generate"), TaskTypeGenerate)
	assert.Equal(t, TaskType("embed"), TaskTypeEmbed)
}

func TestPriorityValues(t *testing.T) {
	assert.Equal(t, Priority(1), PriorityLow)
	assert.Equal(t, Priority(5), PriorityNormal)
	assert.Equal(t, Priority(10), PriorityHigh)
	assert.Equal(t, Priority(15), PriorityCritical)
}

func TestTaskStatusValues(t *testing.T) {
	assert.Equal(t, TaskStatus("pending"), StatusPending)
	assert.Equal(t, TaskStatus("running"), StatusRunning)
	assert.Equal(t, TaskStatus("completed"), StatusCompleted)
	assert.Equal(t, TaskStatus("failed"), StatusFailed)
	assert.Equal(t, TaskStatus("cancelled"), StatusCancelled)
}

func TestTaskCreation(t *testing.T) {
	now := time.Now()
	
	task := &Task{
		ID:       "test-id",
		Type:     TaskTypeChat,
		Priority: PriorityNormal,
		Status:   StatusPending,
		Model:    "test-model",
		Payload: ChatTaskPayload{
			Messages: []ChatMessage{
				{Role: "user", Content: "Hello"},
			},
		},
		Options:    make(map[string]any),
		CreatedAt:  now,
		MaxRetries: 3,
	}
	
	assert.Equal(t, "test-id", task.ID)
	assert.Equal(t, TaskTypeChat, task.Type)
	assert.Equal(t, PriorityNormal, task.Priority)
	assert.Equal(t, StatusPending, task.Status)
	assert.Equal(t, "test-model", task.Model)
	assert.Equal(t, now, task.CreatedAt)
	assert.Equal(t, 3, task.MaxRetries)
	assert.Equal(t, 0, task.RetryCount)
}

func TestTaskResult(t *testing.T) {
	result := &TaskResult{
		TaskID:  "test-id",
		Success: true,
		Data:    "test result",
	}
	
	assert.Equal(t, "test-id", result.TaskID)
	assert.True(t, result.Success)
	assert.Equal(t, "test result", result.Data)
	assert.Empty(t, result.Error)
}

func TestStreamChunk(t *testing.T) {
	// Test data chunk
	chunk := &StreamChunk{
		Data: "Hello",
		Done: false,
	}
	
	assert.Equal(t, "Hello", chunk.Data)
	assert.False(t, chunk.Done)
	assert.NoError(t, chunk.Error)
	
	// Test done chunk
	doneChunk := &StreamChunk{
		Done: true,
	}
	
	assert.True(t, doneChunk.Done)
	assert.Empty(t, doneChunk.Data)
	assert.NoError(t, doneChunk.Error)
	
	// Test error chunk
	errorChunk := &StreamChunk{
		Error: assert.AnError,
		Done:  true,
	}
	
	assert.Error(t, errorChunk.Error)
	assert.True(t, errorChunk.Done)
}

func TestChatTaskPayload(t *testing.T) {
	payload := ChatTaskPayload{
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello"},
		},
		System: "Custom system message",
		Stream: true,
	}
	
	assert.Len(t, payload.Messages, 2)
	assert.Equal(t, "system", payload.Messages[0].Role)
	assert.Equal(t, "You are a helpful assistant", payload.Messages[0].Content)
	assert.Equal(t, "user", payload.Messages[1].Role)
	assert.Equal(t, "Hello", payload.Messages[1].Content)
	assert.Equal(t, "Custom system message", payload.System)
	assert.True(t, payload.Stream)
}

func TestGenerateTaskPayload(t *testing.T) {
	payload := GenerateTaskPayload{
		Prompt:   "Generate a function",
		System:   "You are a coding assistant",
		Template: "Custom template",
		Raw:      true,
		Stream:   false,
	}
	
	assert.Equal(t, "Generate a function", payload.Prompt)
	assert.Equal(t, "You are a coding assistant", payload.System)
	assert.Equal(t, "Custom template", payload.Template)
	assert.True(t, payload.Raw)
	assert.False(t, payload.Stream)
}

func TestEmbedTaskPayload(t *testing.T) {
	payload := EmbedTaskPayload{
		Input:    "Text to embed",
		Truncate: true,
	}
	
	assert.Equal(t, "Text to embed", payload.Input)
	assert.True(t, payload.Truncate)
}

func TestTaskFilter(t *testing.T) {
	filter := TaskFilter{
		Status:   []TaskStatus{StatusPending, StatusRunning},
		Type:     []TaskType{TaskTypeChat},
		Priority: []Priority{PriorityHigh},
		Limit:    10,
		Offset:   5,
	}
	
	assert.Len(t, filter.Status, 2)
	assert.Contains(t, filter.Status, StatusPending)
	assert.Contains(t, filter.Status, StatusRunning)
	assert.Len(t, filter.Type, 1)
	assert.Contains(t, filter.Type, TaskTypeChat)
	assert.Len(t, filter.Priority, 1)
	assert.Contains(t, filter.Priority, PriorityHigh)
	assert.Equal(t, 10, filter.Limit)
	assert.Equal(t, 5, filter.Offset)
}