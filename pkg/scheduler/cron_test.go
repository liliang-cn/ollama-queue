package scheduler

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/stretchr/testify/assert"
)

// MockTaskSubmitter for testing
type MockTaskSubmitter struct {
	SubmitTaskFunc func(task *models.Task) (string, error)
	CallCount      int
	LastTask       *models.Task
	mu             sync.Mutex
}

func (m *MockTaskSubmitter) SubmitTask(task *models.Task) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.CallCount++
	m.LastTask = task
	
	if m.SubmitTaskFunc != nil {
		return m.SubmitTaskFunc(task)
	}
	return fmt.Sprintf("task-%d", m.CallCount), nil
}

// Enhanced MockStorage for cron testing
type CronMockStorage struct {
	MockStorage
	cronTasks map[string]*models.CronTask
	mu        sync.RWMutex
}

func NewCronMockStorage() *CronMockStorage {
	return &CronMockStorage{
		cronTasks: make(map[string]*models.CronTask),
	}
}

func (m *CronMockStorage) SaveCronTask(cronTask *models.CronTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cronTasks[cronTask.ID] = cronTask
	return nil
}

func (m *CronMockStorage) LoadCronTask(cronID string) (*models.CronTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if task, exists := m.cronTasks[cronID]; exists {
		// Return a copy
		taskCopy := *task
		return &taskCopy, nil
	}
	return nil, nil
}

func (m *CronMockStorage) DeleteCronTask(cronID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cronTasks, cronID)
	return nil
}

func (m *CronMockStorage) ListCronTasks(filter models.CronFilter) ([]*models.CronTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result []*models.CronTask
	for _, task := range m.cronTasks {
		// Apply filters
		if filter.Enabled != nil && task.Enabled != *filter.Enabled {
			continue
		}
		
		if len(filter.Names) > 0 {
			found := false
			for _, name := range filter.Names {
				if task.Name == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Create copy and add to result
		taskCopy := *task
		result = append(result, &taskCopy)
	}
	
	// Apply limit and offset
	if filter.Offset > 0 {
		if filter.Offset >= len(result) {
			return []*models.CronTask{}, nil
		}
		result = result[filter.Offset:]
	}
	
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}
	
	return result, nil
}

func TestNewCronScheduler(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	
	scheduler := NewCronScheduler(storage, submitter)
	
	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.storage)
	assert.NotNil(t, scheduler.taskSubmitter)
	assert.NotNil(t, scheduler.cronTasks)
	assert.NotNil(t, scheduler.nextRuns)
	assert.NotNil(t, scheduler.ctx)
	assert.NotNil(t, scheduler.ticker)
}

func TestCronSchedulerAddTask(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	cronTask := &models.CronTask{
		Name:     "Test Task",
		CronExpr: "0 9 * * *",
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskTypeChat,
			Model:    "qwen3",
			Priority: models.PriorityNormal,
			Payload: []models.ChatMessage{
				{Role: "user", Content: "Test message"},
			},
		},
	}
	
	err := scheduler.AddCronTask(cronTask)
	assert.NoError(t, err)
	
	// Verify task was added
	assert.NotEmpty(t, cronTask.ID)
	assert.NotNil(t, cronTask.NextRun)
	assert.False(t, cronTask.CreatedAt.IsZero())
	assert.False(t, cronTask.UpdatedAt.IsZero())
	
	// Verify task is in scheduler
	retrievedTask, err := scheduler.GetCronTask(cronTask.ID)
	assert.NoError(t, err)
	assert.Equal(t, cronTask.Name, retrievedTask.Name)
	assert.Equal(t, cronTask.CronExpr, retrievedTask.CronExpr)
}

func TestCronSchedulerAddTaskInvalidExpression(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	cronTask := &models.CronTask{
		Name:     "Invalid Task",
		CronExpr: "invalid expression",
		Enabled:  true,
	}
	
	err := scheduler.AddCronTask(cronTask)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestCronSchedulerUpdateTask(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	// Add initial task
	cronTask := &models.CronTask{
		Name:     "Test Task",
		CronExpr: "0 9 * * *",
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:  models.TaskTypeChat,
			Model: "qwen3",
		},
	}
	
	err := scheduler.AddCronTask(cronTask)
	assert.NoError(t, err)
	
	originalID := cronTask.ID
	originalNextRun := cronTask.NextRun
	
	// Get the task again to avoid modifying the same instance
	taskToUpdate, err := scheduler.GetCronTask(originalID)
	assert.NoError(t, err)
	
	// Update the task
	taskToUpdate.Name = "Updated Task"
	taskToUpdate.CronExpr = "*/30 * * * *" // Change to every 30 minutes (very different)
	
	err = scheduler.UpdateCronTask(taskToUpdate)
	assert.NoError(t, err)
	
	// Verify update
	retrievedTask, err := scheduler.GetCronTask(originalID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Task", retrievedTask.Name)
	assert.Equal(t, "*/30 * * * *", retrievedTask.CronExpr)
	assert.NotEqual(t, originalNextRun, retrievedTask.NextRun) // Next run should be different
}

func TestCronSchedulerEnableDisable(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	// Add disabled task
	cronTask := &models.CronTask{
		Name:     "Test Task",
		CronExpr: "0 9 * * *",
		Enabled:  false,
	}
	
	err := scheduler.AddCronTask(cronTask)
	assert.NoError(t, err)
	
	// Verify task is disabled
	assert.False(t, cronTask.Enabled)
	assert.Nil(t, cronTask.NextRun)
	
	// Enable the task
	err = scheduler.EnableCronTask(cronTask.ID)
	assert.NoError(t, err)
	
	// Verify task is enabled
	retrievedTask, err := scheduler.GetCronTask(cronTask.ID)
	assert.NoError(t, err)
	assert.True(t, retrievedTask.Enabled)
	assert.NotNil(t, retrievedTask.NextRun)
	
	// Disable the task
	err = scheduler.DisableCronTask(cronTask.ID)
	assert.NoError(t, err)
	
	// Verify task is disabled
	retrievedTask, err = scheduler.GetCronTask(cronTask.ID)
	assert.NoError(t, err)
	assert.False(t, retrievedTask.Enabled)
	assert.Nil(t, retrievedTask.NextRun)
}

func TestCronSchedulerRemoveTask(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	// Add task
	cronTask := &models.CronTask{
		Name:     "Test Task",
		CronExpr: "0 9 * * *",
		Enabled:  true,
	}
	
	err := scheduler.AddCronTask(cronTask)
	assert.NoError(t, err)
	
	taskID := cronTask.ID
	
	// Verify task exists
	_, err = scheduler.GetCronTask(taskID)
	assert.NoError(t, err)
	
	// Remove task
	err = scheduler.RemoveCronTask(taskID)
	assert.NoError(t, err)
	
	// Verify task is removed
	retrievedTask, err := scheduler.GetCronTask(taskID)
	assert.Error(t, err)
	assert.Nil(t, retrievedTask)
}

func TestCronSchedulerListTasks(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	// Add multiple tasks
	tasks := []*models.CronTask{
		{
			Name:     "Task 1",
			CronExpr: "0 9 * * *",
			Enabled:  true,
		},
		{
			Name:     "Task 2",
			CronExpr: "0 18 * * *",
			Enabled:  false,
		},
		{
			Name:     "Task 3",
			CronExpr: "*/15 * * * *",
			Enabled:  true,
		},
	}
	
	for _, task := range tasks {
		err := scheduler.AddCronTask(task)
		assert.NoError(t, err)
	}
	
	// List all tasks
	allTasks, err := scheduler.ListCronTasks(models.CronFilter{})
	assert.NoError(t, err)
	assert.Len(t, allTasks, 3)
	
	// List only enabled tasks
	enabled := true
	enabledTasks, err := scheduler.ListCronTasks(models.CronFilter{Enabled: &enabled})
	assert.NoError(t, err)
	assert.Len(t, enabledTasks, 2)
	
	// List only disabled tasks
	disabled := false
	disabledTasks, err := scheduler.ListCronTasks(models.CronFilter{Enabled: &disabled})
	assert.NoError(t, err)
	assert.Len(t, disabledTasks, 1)
	
	// List with name filter
	tasksByName, err := scheduler.ListCronTasks(models.CronFilter{Names: []string{"Task 1"}})
	assert.NoError(t, err)
	assert.Len(t, tasksByName, 1)
	assert.Equal(t, "Task 1", tasksByName[0].Name)
	
	// List with limit
	limitedTasks, err := scheduler.ListCronTasks(models.CronFilter{Limit: 2})
	assert.NoError(t, err)
	assert.Len(t, limitedTasks, 2)
}

func TestCronSchedulerTaskExecution(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	// Add task that should run soon
	cronTask := &models.CronTask{
		Name:     "Test Task",
		CronExpr: "* * * * *", // Every minute
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskTypeChat,
			Model:    "qwen3",
			Priority: models.PriorityNormal,
			Payload: []models.ChatMessage{
				{Role: "user", Content: "Test message"},
			},
		},
	}
	
	err := scheduler.AddCronTask(cronTask)
	assert.NoError(t, err)
	
	// Manually trigger task check with a time that should match
	now := time.Now().Truncate(time.Minute)
	
	// Set next run to current time to simulate a due task
	cronTask.NextRun = &now
	scheduler.cronTasks[cronTask.ID] = cronTask
	scheduler.nextRuns[cronTask.ID] = now
	
	// Call checkAndRunTasks
	scheduler.checkAndRunTasks()
	
	// Verify task was submitted
	assert.Equal(t, 1, submitter.CallCount)
	assert.NotNil(t, submitter.LastTask)
	assert.Equal(t, models.TaskTypeChat, submitter.LastTask.Type)
	
	// Verify run count was incremented
	updatedTask, err := scheduler.GetCronTask(cronTask.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedTask.RunCount)
	assert.NotNil(t, updatedTask.LastRun)
	assert.NotNil(t, updatedTask.NextRun)
}

func TestCronSchedulerStartStop(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	// Start scheduler
	err := scheduler.Start()
	assert.NoError(t, err)
	
	// Add a task
	cronTask := &models.CronTask{
		Name:     "Test Task",
		CronExpr: "0 9 * * *",
		Enabled:  true,
	}
	
	err = scheduler.AddCronTask(cronTask)
	assert.NoError(t, err)
	
	// Stop scheduler
	err = scheduler.Stop()
	assert.NoError(t, err)
	
	// Verify context is cancelled
	select {
	case <-scheduler.ctx.Done():
		// Context should be cancelled
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestCronSchedulerLoadExistingTasks(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	
	// Pre-populate storage with tasks
	existingTask := &models.CronTask{
		ID:       "existing-123",
		Name:     "Existing Task",
		CronExpr: "0 9 * * *",
		Enabled:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	storage.SaveCronTask(existingTask)
	
	scheduler := NewCronScheduler(storage, submitter)
	
	// Start scheduler (this should load existing tasks)
	err := scheduler.Start()
	assert.NoError(t, err)
	
	// Verify existing task was loaded
	loadedTask, err := scheduler.GetCronTask("existing-123")
	assert.NoError(t, err)
	assert.NotNil(t, loadedTask)
	assert.Equal(t, "Existing Task", loadedTask.Name)
	assert.NotNil(t, loadedTask.NextRun) // Should have calculated next run
	
	scheduler.Stop()
}

func TestCronSchedulerGetNextRuns(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	// Add multiple enabled tasks
	tasks := []*models.CronTask{
		{
			Name:     "Task 1",
			CronExpr: "0 9 * * *",
			Enabled:  true,
		},
		{
			Name:     "Task 2",
			CronExpr: "0 18 * * *",
			Enabled:  true,
		},
		{
			Name:     "Task 3",
			CronExpr: "*/15 * * * *",
			Enabled:  false, // Disabled task
		},
	}
	
	var taskIDs []string
	for _, task := range tasks {
		err := scheduler.AddCronTask(task)
		assert.NoError(t, err)
		taskIDs = append(taskIDs, task.ID)
	}
	
	// Get next runs
	nextRuns := scheduler.GetNextRuns()
	
	// Should only include enabled tasks (2 tasks)
	assert.Len(t, nextRuns, 2)
	assert.Contains(t, nextRuns, taskIDs[0])
	assert.Contains(t, nextRuns, taskIDs[1])
	assert.NotContains(t, nextRuns, taskIDs[2]) // Disabled task
}

func TestCronSchedulerConcurrency(t *testing.T) {
	storage := NewCronMockStorage()
	submitter := &MockTaskSubmitter{}
	scheduler := NewCronScheduler(storage, submitter)
	
	err := scheduler.Start()
	assert.NoError(t, err)
	defer scheduler.Stop()
	
	// Add multiple tasks concurrently
	var wg sync.WaitGroup
	numTasks := 10
	
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			cronTask := &models.CronTask{
				Name:     fmt.Sprintf("Concurrent Task %d", index),
				CronExpr: "0 9 * * *",
				Enabled:  true,
				TaskTemplate: models.TaskTemplate{
					Type:  models.TaskTypeChat,
					Model: "qwen3",
				},
			}
			
			err := scheduler.AddCronTask(cronTask)
			assert.NoError(t, err)
		}(i)
	}
	
	wg.Wait()
	
	// Verify all tasks were added
	allTasks, err := scheduler.ListCronTasks(models.CronFilter{})
	assert.NoError(t, err)
	assert.Len(t, allTasks, numTasks)
}