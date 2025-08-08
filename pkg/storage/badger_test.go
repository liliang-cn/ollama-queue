package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/liliang-cn/ollama-queue/internal/models"
)

func TestBadgerStorage_OpenClose(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ollama-queue-test")
	defer os.RemoveAll(tmpDir)

	storage := NewBadgerStorage(tmpDir)
	
	err := storage.Open()
	require.NoError(t, err)
	
	err = storage.Close()
	require.NoError(t, err)
}

func TestBadgerStorage_SaveAndLoadTask(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ollama-queue-test-save-load")
	defer os.RemoveAll(tmpDir)

	storage := NewBadgerStorage(tmpDir)
	require.NoError(t, storage.Open())
	defer storage.Close()

	task := &models.Task{
		ID:       "test-task-1",
		Type:     models.TaskTypeChat,
		Priority: models.PriorityNormal,
		Status:   models.StatusPending,
		Model:    "test-model",
		Payload: models.ChatTaskPayload{
			Messages: []models.ChatMessage{
				{Role: "user", Content: "Hello"},
			},
		},
		Options:    make(map[string]any),
		CreatedAt:  time.Now().Truncate(time.Second), // Truncate for comparison
		MaxRetries: 3,
	}

	// Save task
	err := storage.SaveTask(task)
	require.NoError(t, err)

	// Load task
	loadedTask, err := storage.LoadTask(task.ID)
	require.NoError(t, err)
	
	assert.Equal(t, task.ID, loadedTask.ID)
	assert.Equal(t, task.Type, loadedTask.Type)
	assert.Equal(t, task.Priority, loadedTask.Priority)
	assert.Equal(t, task.Status, loadedTask.Status)
	assert.Equal(t, task.Model, loadedTask.Model)
	assert.Equal(t, task.MaxRetries, loadedTask.MaxRetries)
	assert.Equal(t, task.CreatedAt.Unix(), loadedTask.CreatedAt.Unix())
}

func TestBadgerStorage_UpdateTaskStatus(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ollama-queue-test-update-status")
	defer os.RemoveAll(tmpDir)

	storage := NewBadgerStorage(tmpDir)
	require.NoError(t, storage.Open())
	defer storage.Close()

	task := &models.Task{
		ID:        "test-task-2",
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Save initial task
	require.NoError(t, storage.SaveTask(task))

	// Update status to running
	err := storage.UpdateTaskStatus(task.ID, models.StatusRunning)
	require.NoError(t, err)

	// Verify status was updated
	updatedTask, err := storage.LoadTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusRunning, updatedTask.Status)
	assert.NotNil(t, updatedTask.StartedAt)

	// Update status to completed
	err = storage.UpdateTaskStatus(task.ID, models.StatusCompleted)
	require.NoError(t, err)

	// Verify status was updated
	completedTask, err := storage.LoadTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusCompleted, completedTask.Status)
	assert.NotNil(t, completedTask.CompletedAt)
}

func TestBadgerStorage_DeleteTask(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ollama-queue-test-delete")
	defer os.RemoveAll(tmpDir)

	storage := NewBadgerStorage(tmpDir)
	require.NoError(t, storage.Open())
	defer storage.Close()

	task := &models.Task{
		ID:        "test-task-3",
		Status:    models.StatusPending,
		Priority:  models.PriorityNormal,
		CreatedAt: time.Now(),
	}

	// Save task
	require.NoError(t, storage.SaveTask(task))

	// Verify task exists
	_, err := storage.LoadTask(task.ID)
	require.NoError(t, err)

	// Delete task
	err = storage.DeleteTask(task.ID)
	require.NoError(t, err)

	// Verify task no longer exists
	_, err = storage.LoadTask(task.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestBadgerStorage_GetTasksByStatus(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ollama-queue-test-get-by-status")
	defer os.RemoveAll(tmpDir)

	storage := NewBadgerStorage(tmpDir)
	require.NoError(t, storage.Open())
	defer storage.Close()

	// Create tasks with different statuses
	pendingTask1 := &models.Task{
		ID:        "pending-1",
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}
	
	pendingTask2 := &models.Task{
		ID:        "pending-2",
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}
	
	runningTask := &models.Task{
		ID:        "running-1",
		Status:    models.StatusRunning,
		CreatedAt: time.Now(),
	}

	// Save tasks
	require.NoError(t, storage.SaveTask(pendingTask1))
	require.NoError(t, storage.SaveTask(pendingTask2))
	require.NoError(t, storage.SaveTask(runningTask))

	// Get pending tasks
	pendingTasks, err := storage.GetTasksByStatus(models.StatusPending)
	require.NoError(t, err)
	assert.Len(t, pendingTasks, 2)

	// Verify task IDs
	taskIDs := make([]string, len(pendingTasks))
	for i, task := range pendingTasks {
		taskIDs[i] = task.ID
	}
	assert.Contains(t, taskIDs, "pending-1")
	assert.Contains(t, taskIDs, "pending-2")

	// Get running tasks
	runningTasks, err := storage.GetTasksByStatus(models.StatusRunning)
	require.NoError(t, err)
	assert.Len(t, runningTasks, 1)
	assert.Equal(t, "running-1", runningTasks[0].ID)

	// Get completed tasks (should be empty)
	completedTasks, err := storage.GetTasksByStatus(models.StatusCompleted)
	require.NoError(t, err)
	assert.Len(t, completedTasks, 0)
}

func TestBadgerStorage_ListTasksWithFilter(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ollama-queue-test-list-filter")
	defer os.RemoveAll(tmpDir)

	storage := NewBadgerStorage(tmpDir)
	require.NoError(t, storage.Open())
	defer storage.Close()

	// Create tasks with different types and priorities
	tasks := []*models.Task{
		{
			ID:        "task-1",
			Type:      models.TaskTypeChat,
			Priority:  models.PriorityHigh,
			Status:    models.StatusPending,
			CreatedAt: time.Now(),
		},
		{
			ID:        "task-2",
			Type:      models.TaskTypeGenerate,
			Priority:  models.PriorityNormal,
			Status:    models.StatusRunning,
			CreatedAt: time.Now(),
		},
		{
			ID:        "task-3",
			Type:      models.TaskTypeChat,
			Priority:  models.PriorityLow,
			Status:    models.StatusCompleted,
			CreatedAt: time.Now(),
		},
	}

	// Save all tasks
	for _, task := range tasks {
		require.NoError(t, storage.SaveTask(task))
	}

	// Test filter by type
	chatFilter := models.TaskFilter{
		Type: []models.TaskType{models.TaskTypeChat},
	}
	chatTasks, err := storage.ListTasks(chatFilter)
	require.NoError(t, err)
	assert.Len(t, chatTasks, 2)

	// Test filter by status
	runningFilter := models.TaskFilter{
		Status: []models.TaskStatus{models.StatusRunning},
	}
	runningTasks, err := storage.ListTasks(runningFilter)
	require.NoError(t, err)
	assert.Len(t, runningTasks, 1)
	assert.Equal(t, "task-2", runningTasks[0].ID)

	// Test limit
	limitFilter := models.TaskFilter{
		Limit: 2,
	}
	limitedTasks, err := storage.ListTasks(limitFilter)
	require.NoError(t, err)
	assert.Len(t, limitedTasks, 2)
}

func TestBadgerStorage_GetStats(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ollama-queue-test-stats")
	defer os.RemoveAll(tmpDir)

	storage := NewBadgerStorage(tmpDir)
	require.NoError(t, storage.Open())
	defer storage.Close()

	// Create tasks with different statuses
	tasks := []*models.Task{
		{ID: "pending-1", Status: models.StatusPending, Priority: models.PriorityHigh, CreatedAt: time.Now()},
		{ID: "pending-2", Status: models.StatusPending, Priority: models.PriorityNormal, CreatedAt: time.Now()},
		{ID: "running-1", Status: models.StatusRunning, Priority: models.PriorityNormal, CreatedAt: time.Now()},
		{ID: "completed-1", Status: models.StatusCompleted, Priority: models.PriorityLow, CreatedAt: time.Now()},
		{ID: "failed-1", Status: models.StatusFailed, Priority: models.PriorityLow, CreatedAt: time.Now()},
	}

	// Save all tasks
	for _, task := range tasks {
		require.NoError(t, storage.SaveTask(task))
	}

	// Get stats
	stats, err := storage.GetStats()
	require.NoError(t, err)

	assert.Equal(t, 2, stats.PendingTasks)
	assert.Equal(t, 1, stats.RunningTasks)
	assert.Equal(t, 1, stats.CompletedTasks)
	assert.Equal(t, 1, stats.FailedTasks)
	assert.Equal(t, 0, stats.CancelledTasks)
	assert.Equal(t, 5, stats.TotalTasks)

	// Check priority queues for pending tasks
	assert.Equal(t, 1, stats.QueuesByPriority[models.PriorityHigh])
	assert.Equal(t, 1, stats.QueuesByPriority[models.PriorityNormal])
	assert.Equal(t, 0, stats.QueuesByPriority[models.PriorityLow]) // Not pending
}