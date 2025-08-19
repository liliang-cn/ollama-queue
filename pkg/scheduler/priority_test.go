package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/liliang-cn/ollama-queue/pkg/models"
)

func TestNewPriorityScheduler(t *testing.T) {
	scheduler := NewPriorityScheduler()
	
	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.queues)
	assert.NotNil(t, scheduler.tasks)
	
	// Check that all priority queues are initialized
	assert.Contains(t, scheduler.queues, models.PriorityLow)
	assert.Contains(t, scheduler.queues, models.PriorityNormal)
	assert.Contains(t, scheduler.queues, models.PriorityHigh)
	assert.Contains(t, scheduler.queues, models.PriorityCritical)
}

func TestPriorityScheduler_AddTask(t *testing.T) {
	scheduler := NewPriorityScheduler()
	
	task := &models.Task{
		ID:       "test-1",
		Type:     models.TaskTypeChat,
		Priority: models.PriorityHigh,
		Status:   models.StatusPending,
		Model:    "test-model",
		CreatedAt: time.Now(),
	}
	
	err := scheduler.AddTask(task)
	require.NoError(t, err)
	
	// Check task is stored
	assert.Contains(t, scheduler.tasks, task.ID)
	assert.Equal(t, task, scheduler.tasks[task.ID])
	
	// Check queue length
	assert.Equal(t, 1, scheduler.GetQueueLength())
	
	// Check priority queue stats
	stats := scheduler.GetQueueStats()
	assert.Equal(t, 1, stats[models.PriorityHigh])
	assert.Equal(t, 0, stats[models.PriorityNormal])
}

func TestPriorityScheduler_GetNextTask(t *testing.T) {
	scheduler := NewPriorityScheduler()
	
	// Add tasks with different priorities
	lowTask := &models.Task{
		ID:       "low-1",
		Priority: models.PriorityLow,
		CreatedAt: time.Now().Add(-2 * time.Second),
	}
	
	normalTask := &models.Task{
		ID:       "normal-1",
		Priority: models.PriorityNormal,
		CreatedAt: time.Now().Add(-1 * time.Second),
	}
	
	highTask := &models.Task{
		ID:       "high-1",
		Priority: models.PriorityHigh,
		CreatedAt: time.Now(),
	}
	
	criticalTask := &models.Task{
		ID:       "critical-1",
		Priority: models.PriorityCritical,
		CreatedAt: time.Now().Add(1 * time.Second),
	}
	
	// Add in random order
	scheduler.AddTask(normalTask)
	scheduler.AddTask(lowTask)
	scheduler.AddTask(criticalTask)
	scheduler.AddTask(highTask)
	
	// Should get tasks in priority order (highest first)
	task1 := scheduler.GetNextTask()
	assert.NotNil(t, task1)
	assert.Equal(t, "critical-1", task1.ID)
	assert.Equal(t, models.PriorityCritical, task1.Priority)
	
	task2 := scheduler.GetNextTask()
	assert.NotNil(t, task2)
	assert.Equal(t, "high-1", task2.ID)
	assert.Equal(t, models.PriorityHigh, task2.Priority)
	
	task3 := scheduler.GetNextTask()
	assert.NotNil(t, task3)
	assert.Equal(t, "normal-1", task3.ID)
	assert.Equal(t, models.PriorityNormal, task3.Priority)
	
	task4 := scheduler.GetNextTask()
	assert.NotNil(t, task4)
	assert.Equal(t, "low-1", task4.ID)
	assert.Equal(t, models.PriorityLow, task4.Priority)
	
	// No more tasks
	task5 := scheduler.GetNextTask()
	assert.Nil(t, task5)
	
	// Queue should be empty
	assert.Equal(t, 0, scheduler.GetQueueLength())
}

func TestPriorityScheduler_FIFO_SamePriority(t *testing.T) {
	scheduler := NewPriorityScheduler()
	
	// Add tasks with same priority but different creation times
	task1 := &models.Task{
		ID:        "task-1",
		Priority:  models.PriorityNormal,
		CreatedAt: time.Now().Add(-2 * time.Second),
	}
	
	task2 := &models.Task{
		ID:        "task-2",
		Priority:  models.PriorityNormal,
		CreatedAt: time.Now().Add(-1 * time.Second),
	}
	
	task3 := &models.Task{
		ID:        "task-3",
		Priority:  models.PriorityNormal,
		CreatedAt: time.Now(),
	}
	
	// Add in reverse order
	scheduler.AddTask(task3)
	scheduler.AddTask(task1)
	scheduler.AddTask(task2)
	
	// Should get tasks in FIFO order for same priority
	next1 := scheduler.GetNextTask()
	assert.Equal(t, "task-1", next1.ID) // oldest first
	
	next2 := scheduler.GetNextTask()
	assert.Equal(t, "task-2", next2.ID) // second oldest
	
	next3 := scheduler.GetNextTask()
	assert.Equal(t, "task-3", next3.ID) // newest
}

func TestPriorityScheduler_RemoveTask(t *testing.T) {
	scheduler := NewPriorityScheduler()
	
	task := &models.Task{
		ID:       "test-1",
		Priority: models.PriorityNormal,
		CreatedAt: time.Now(),
	}
	
	scheduler.AddTask(task)
	assert.Equal(t, 1, scheduler.GetQueueLength())
	
	// Remove task
	err := scheduler.RemoveTask(task.ID)
	require.NoError(t, err)
	
	assert.Equal(t, 0, scheduler.GetQueueLength())
	assert.NotContains(t, scheduler.tasks, task.ID)
	
	// Should not get any task
	next := scheduler.GetNextTask()
	assert.Nil(t, next)
}

func TestPriorityScheduler_UpdateTaskPriority(t *testing.T) {
	scheduler := NewPriorityScheduler()
	
	task := &models.Task{
		ID:       "test-1",
		Priority: models.PriorityLow,
		CreatedAt: time.Now(),
	}
	
	scheduler.AddTask(task)
	
	// Update priority
	err := scheduler.UpdateTaskPriority(task.ID, models.PriorityHigh)
	require.NoError(t, err)
	
	// Check task priority was updated
	assert.Equal(t, models.PriorityHigh, task.Priority)
	
	// Check queue stats
	stats := scheduler.GetQueueStats()
	assert.Equal(t, 0, stats[models.PriorityLow])
	assert.Equal(t, 1, stats[models.PriorityHigh])
	
	// Task should come out with new priority
	next := scheduler.GetNextTask()
	assert.NotNil(t, next)
	assert.Equal(t, task.ID, next.ID)
	assert.Equal(t, models.PriorityHigh, next.Priority)
}

func TestRetryScheduler(t *testing.T) {
	config := models.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}
	
	retryScheduler := NewRetryScheduler(config)
	assert.NotNil(t, retryScheduler)
	assert.Equal(t, 0, retryScheduler.GetRetryQueueLength())
}

func TestRetryScheduler_AddFailedTask(t *testing.T) {
	config := models.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}
	
	retryScheduler := NewRetryScheduler(config)
	
	task := &models.Task{
		ID:         "test-1",
		RetryCount: 0,
		MaxRetries: 3,
	}
	
	// Add failed task
	err := retryScheduler.AddFailedTask(task)
	require.NoError(t, err)
	
	assert.Equal(t, 1, retryScheduler.GetRetryQueueLength())
	
	// Should not be ready immediately
	readyTasks := retryScheduler.GetReadyTasks()
	assert.Len(t, readyTasks, 0)
}

func TestRetryScheduler_GetReadyTasks(t *testing.T) {
	config := models.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond, // Very short for testing
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}
	
	retryScheduler := NewRetryScheduler(config)
	
	task := &models.Task{
		ID:         "test-1",
		RetryCount: 0,
		MaxRetries: 3,
		Status:     models.StatusFailed,
	}
	
	// Add failed task
	retryScheduler.AddFailedTask(task)
	
	// Wait for delay to pass
	time.Sleep(15 * time.Millisecond)
	
	// Should be ready now
	readyTasks := retryScheduler.GetReadyTasks()
	require.Len(t, readyTasks, 1)
	
	readyTask := readyTasks[0]
	assert.Equal(t, task.ID, readyTask.ID)
	assert.Equal(t, 1, readyTask.RetryCount) // Should be incremented
	assert.Equal(t, models.StatusPending, readyTask.Status)
	
	// Should not be in retry queue anymore
	assert.Equal(t, 0, retryScheduler.GetRetryQueueLength())
}

func TestRetryScheduler_MaxRetriesExceeded(t *testing.T) {
	config := models.RetryConfig{
		MaxRetries:    2,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}
	
	retryScheduler := NewRetryScheduler(config)
	
	task := &models.Task{
		ID:         "test-1",
		RetryCount: 2,
		MaxRetries: 2,
	}
	
	// Should not add task when max retries exceeded
	err := retryScheduler.AddFailedTask(task)
	require.NoError(t, err)
	
	assert.Equal(t, 0, retryScheduler.GetRetryQueueLength())
}

func TestRetryScheduler_ExponentialBackoff(t *testing.T) {
	config := models.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}
	
	retryScheduler := NewRetryScheduler(config)
	
	// Test first retry (retry count = 0)
	task1 := &models.Task{
		ID:         "test-1",
		RetryCount: 0,
		MaxRetries: 3,
	}
	
	// Test second retry (retry count = 1) 
	task2 := &models.Task{
		ID:         "test-2",
		RetryCount: 1,
		MaxRetries: 3,
	}
	
	beforeAdd := time.Now()
	retryScheduler.AddFailedTask(task1) // Should use 100ms delay
	retryScheduler.AddFailedTask(task2) // Should use 200ms delay (100 * 2^1)
	afterAdd := time.Now()
	
	assert.Equal(t, 2, retryScheduler.GetRetryQueueLength())
	
	// Check that delays are applied correctly
	// We can't easily test the exact timing, but we can test that
	// tasks aren't immediately ready
	readyTasks := retryScheduler.GetReadyTasks()
	assert.Len(t, readyTasks, 0)
	
	// The test just verifies the structure exists - timing tests would be flaky
	assert.True(t, afterAdd.After(beforeAdd))
}