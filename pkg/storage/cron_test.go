package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBadgerStorageCronOperations(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "badger_cron_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage := NewBadgerStorage(filepath.Join(tempDir, "test.db"))
	err = storage.Open()
	require.NoError(t, err)
	defer storage.Close()

	t.Run("SaveAndLoadCronTask", func(t *testing.T) {
		cronTask := &models.CronTask{
			ID:       "test-cron-123",
			Name:     "Test Cron Task",
			CronExpr: "0 9 * * 1-5",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:     models.TaskTypeChat,
				Model:    "qwen3",
				Priority: models.PriorityNormal,
				Payload: []models.ChatMessage{
					{Role: "user", Content: "Test message"},
				},
				Options: map[string]interface{}{
					"stream": false,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			RunCount:  5,
			Metadata: map[string]interface{}{
				"department": "testing",
				"owner":      "test-user",
			},
		}

		// Set next run time
		nextRun := time.Now().Add(time.Hour)
		cronTask.NextRun = &nextRun

		// Set last run time
		lastRun := time.Now().Add(-time.Hour)
		cronTask.LastRun = &lastRun

		// Save cron task
		err := storage.SaveCronTask(cronTask)
		assert.NoError(t, err)

		// Load cron task
		loadedTask, err := storage.LoadCronTask("test-cron-123")
		assert.NoError(t, err)
		assert.NotNil(t, loadedTask)

		// Verify all fields
		assert.Equal(t, cronTask.ID, loadedTask.ID)
		assert.Equal(t, cronTask.Name, loadedTask.Name)
		assert.Equal(t, cronTask.CronExpr, loadedTask.CronExpr)
		assert.Equal(t, cronTask.Enabled, loadedTask.Enabled)
		assert.Equal(t, cronTask.RunCount, loadedTask.RunCount)
		assert.Equal(t, cronTask.TaskTemplate.Type, loadedTask.TaskTemplate.Type)
		assert.Equal(t, cronTask.TaskTemplate.Model, loadedTask.TaskTemplate.Model)
		assert.Equal(t, cronTask.TaskTemplate.Priority, loadedTask.TaskTemplate.Priority)

		// Verify timestamps (with small tolerance for serialization)
		assert.WithinDuration(t, cronTask.CreatedAt, loadedTask.CreatedAt, time.Second)
		assert.WithinDuration(t, cronTask.UpdatedAt, loadedTask.UpdatedAt, time.Second)
		assert.WithinDuration(t, *cronTask.NextRun, *loadedTask.NextRun, time.Second)
		assert.WithinDuration(t, *cronTask.LastRun, *loadedTask.LastRun, time.Second)

		// Verify metadata
		assert.Equal(t, cronTask.Metadata["department"], loadedTask.Metadata["department"])
		assert.Equal(t, cronTask.Metadata["owner"], loadedTask.Metadata["owner"])

		// Verify payload
		originalPayload, ok := cronTask.TaskTemplate.Payload.([]models.ChatMessage)
		assert.True(t, ok)
		
		// Note: JSON marshaling/unmarshaling converts to []interface{} containing map[string]interface{}
		loadedPayloadInterface, ok := loadedTask.TaskTemplate.Payload.([]interface{})
		assert.True(t, ok)
		assert.Len(t, loadedPayloadInterface, 1)
		
		loadedMessageMap, ok := loadedPayloadInterface[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, originalPayload[0].Role, loadedMessageMap["role"])
		assert.Equal(t, originalPayload[0].Content, loadedMessageMap["content"])

		// Verify options
		assert.Equal(t, cronTask.TaskTemplate.Options["stream"], loadedTask.TaskTemplate.Options["stream"])
	})

	t.Run("LoadNonExistentCronTask", func(t *testing.T) {
		loadedTask, err := storage.LoadCronTask("non-existent")
		assert.Error(t, err)
		assert.Nil(t, loadedTask)
		assert.Contains(t, err.Error(), "cron task not found")
	})

	t.Run("UpdateCronTask", func(t *testing.T) {
		cronTask := &models.CronTask{
			ID:       "update-test-123",
			Name:     "Original Name",
			CronExpr: "0 9 * * *",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:     models.TaskTypeGenerate,
				Model:    "qwen3",
				Priority: models.PriorityLow,
				Payload: map[string]interface{}{
					"prompt": "Original prompt",
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save initial task
		err := storage.SaveCronTask(cronTask)
		assert.NoError(t, err)

		// Update the task
		cronTask.Name = "Updated Name"
		cronTask.CronExpr = "0 18 * * *"
		cronTask.Enabled = false
		cronTask.UpdatedAt = time.Now()
		cronTask.RunCount = 10

		// Save updated task
		err = storage.SaveCronTask(cronTask)
		assert.NoError(t, err)

		// Load and verify updates
		loadedTask, err := storage.LoadCronTask("update-test-123")
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", loadedTask.Name)
		assert.Equal(t, "0 18 * * *", loadedTask.CronExpr)
		assert.False(t, loadedTask.Enabled)
		assert.Equal(t, 10, loadedTask.RunCount)
	})

	t.Run("DeleteCronTask", func(t *testing.T) {
		cronTask := &models.CronTask{
			ID:       "delete-test-123",
			Name:     "To Be Deleted",
			CronExpr: "0 9 * * *",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:  models.TaskTypeChat,
				Model: "qwen3",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save task
		err := storage.SaveCronTask(cronTask)
		assert.NoError(t, err)

		// Verify task exists
		loadedTask, err := storage.LoadCronTask("delete-test-123")
		assert.NoError(t, err)
		assert.NotNil(t, loadedTask)

		// Delete task
		err = storage.DeleteCronTask("delete-test-123")
		assert.NoError(t, err)

		// Verify task is deleted
		loadedTask, err = storage.LoadCronTask("delete-test-123")
		assert.Error(t, err)
		assert.Nil(t, loadedTask)
	})

	t.Run("DeleteNonExistentCronTask", func(t *testing.T) {
		// Should not error when deleting non-existent task
		err := storage.DeleteCronTask("non-existent")
		assert.NoError(t, err)
	})
}

func TestBadgerStorageListCronTasks(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "badger_list_cron_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage := NewBadgerStorage(filepath.Join(tempDir, "test.db"))
	err = storage.Open()
	require.NoError(t, err)
	defer storage.Close()

	// Create test tasks
	tasks := []*models.CronTask{
		{
			ID:       "task-1",
			Name:     "Enabled Task 1",
			CronExpr: "0 9 * * *",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:  models.TaskTypeChat,
				Model: "qwen3",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "task-2",
			Name:     "Disabled Task",
			CronExpr: "0 18 * * *",
			Enabled:  false,
			TaskTemplate: models.TaskTemplate{
				Type:  models.TaskTypeGenerate,
				Model: "qwen3",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "task-3",
			Name:     "Enabled Task 2",
			CronExpr: "*/15 * * * *",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:  models.TaskTypeEmbed,
				Model: "qwen3",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:       "task-4",
			Name:     "Special Name",
			CronExpr: "0 12 * * *",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:  models.TaskTypeChat,
				Model: "qwen3",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Save all tasks
	for _, task := range tasks {
		err := storage.SaveCronTask(task)
		assert.NoError(t, err)
	}

	t.Run("ListAllTasks", func(t *testing.T) {
		cronTasks, err := storage.ListCronTasks(models.CronFilter{})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 4)

		// Verify all task IDs are present
		taskIDs := make(map[string]bool)
		for _, task := range cronTasks {
			taskIDs[task.ID] = true
		}
		assert.True(t, taskIDs["task-1"])
		assert.True(t, taskIDs["task-2"])
		assert.True(t, taskIDs["task-3"])
		assert.True(t, taskIDs["task-4"])
	})

	t.Run("ListEnabledTasks", func(t *testing.T) {
		enabled := true
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Enabled: &enabled,
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 3)

		// Verify all returned tasks are enabled
		for _, task := range cronTasks {
			assert.True(t, task.Enabled)
		}
	})

	t.Run("ListDisabledTasks", func(t *testing.T) {
		disabled := false
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Enabled: &disabled,
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 1)
		assert.Equal(t, "task-2", cronTasks[0].ID)
		assert.False(t, cronTasks[0].Enabled)
	})

	t.Run("ListTasksByNames", func(t *testing.T) {
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Names: []string{"Enabled Task 1", "Special Name"},
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 2)

		taskNames := make(map[string]bool)
		for _, task := range cronTasks {
			taskNames[task.Name] = true
		}
		assert.True(t, taskNames["Enabled Task 1"])
		assert.True(t, taskNames["Special Name"])
	})

	t.Run("ListTasksWithLimit", func(t *testing.T) {
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Limit: 2,
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 2)
	})

	t.Run("ListTasksWithOffset", func(t *testing.T) {
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Offset: 2,
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 2)
	})

	t.Run("ListTasksWithLimitAndOffset", func(t *testing.T) {
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Limit:  1,
			Offset: 1,
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 1)
	})

	t.Run("ListTasksOffsetBeyondTotal", func(t *testing.T) {
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Offset: 10,
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 0)
	})

	t.Run("ListTasksNoMatches", func(t *testing.T) {
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Names: []string{"Non-existent Task"},
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 0)
	})

	t.Run("CombinedFilters", func(t *testing.T) {
		enabled := true
		cronTasks, err := storage.ListCronTasks(models.CronFilter{
			Enabled: &enabled,
			Names:   []string{"Enabled Task 1", "Disabled Task"},
		})
		assert.NoError(t, err)
		assert.Len(t, cronTasks, 1) // Only "Enabled Task 1" matches both filters
		assert.Equal(t, "Enabled Task 1", cronTasks[0].Name)
		assert.True(t, cronTasks[0].Enabled)
	})
}

func TestBadgerStorageCronTaskIndexing(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "badger_index_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage := NewBadgerStorage(filepath.Join(tempDir, "test.db"))
	err = storage.Open()
	require.NoError(t, err)
	defer storage.Close()

	cronTask := &models.CronTask{
		ID:       "index-test-123",
		Name:     "Index Test Task",
		CronExpr: "0 9 * * *",
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:  models.TaskTypeChat,
			Model: "qwen3",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save task (this should create index entries)
	err = storage.SaveCronTask(cronTask)
	assert.NoError(t, err)

	// Update task to change indexed fields
	cronTask.Name = "Updated Index Test Task"
	cronTask.Enabled = false
	err = storage.SaveCronTask(cronTask)
	assert.NoError(t, err)

	// Verify the task can still be retrieved with new values
	disabled := false
	cronTasks, err := storage.ListCronTasks(models.CronFilter{
		Enabled: &disabled,
	})
	assert.NoError(t, err)
	assert.Len(t, cronTasks, 1)
	assert.Equal(t, "Updated Index Test Task", cronTasks[0].Name)

	// Delete task and verify it's removed from indexes
	err = storage.DeleteCronTask("index-test-123")
	assert.NoError(t, err)

	// Task should no longer be found
	cronTasks, err = storage.ListCronTasks(models.CronFilter{})
	assert.NoError(t, err)
	assert.Len(t, cronTasks, 0)
}

func TestBadgerStorageCronTaskComplexPayloads(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "badger_payload_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage := NewBadgerStorage(filepath.Join(tempDir, "test.db"))
	err = storage.Open()
	require.NoError(t, err)
	defer storage.Close()

	t.Run("ChatTaskPayload", func(t *testing.T) {
		cronTask := &models.CronTask{
			ID:       "chat-payload-test",
			Name:     "Chat Payload Test",
			CronExpr: "0 9 * * *",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:     models.TaskTypeChat,
				Model:    "qwen3",
				Priority: models.PriorityHigh,
				Payload: []models.ChatMessage{
					{Role: "system", Content: "You are a helpful assistant"},
					{Role: "user", Content: "Hello, how are you?"},
					{Role: "assistant", Content: "I'm doing well, thank you!"},
				},
				Options: map[string]interface{}{
					"stream":      false,
					"temperature": 0.7,
					"max_tokens":  100,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := storage.SaveCronTask(cronTask)
		assert.NoError(t, err)

		loadedTask, err := storage.LoadCronTask("chat-payload-test")
		assert.NoError(t, err)
		assert.Equal(t, models.TaskTypeChat, loadedTask.TaskTemplate.Type)
		assert.Equal(t, models.PriorityHigh, loadedTask.TaskTemplate.Priority)

		// Verify options
		assert.Equal(t, false, loadedTask.TaskTemplate.Options["stream"])
		assert.Equal(t, 0.7, loadedTask.TaskTemplate.Options["temperature"])
		assert.Equal(t, float64(100), loadedTask.TaskTemplate.Options["max_tokens"]) // JSON unmarshaling converts to float64
	})

	t.Run("GenerateTaskPayload", func(t *testing.T) {
		cronTask := &models.CronTask{
			ID:       "generate-payload-test",
			Name:     "Generate Payload Test",
			CronExpr: "*/30 * * * *",
			Enabled:  true,
			TaskTemplate: models.TaskTemplate{
				Type:     models.TaskTypeGenerate,
				Model:    "qwen3",
				Priority: models.PriorityNormal,
				Payload: map[string]interface{}{
					"prompt":  "Generate a creative story",
					"system":  "You are a creative writer",
					"context": []string{"fantasy", "adventure", "magic"},
				},
				Options: map[string]interface{}{
					"stream":      true,
					"temperature": 0.9,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := storage.SaveCronTask(cronTask)
		assert.NoError(t, err)

		loadedTask, err := storage.LoadCronTask("generate-payload-test")
		assert.NoError(t, err)
		assert.Equal(t, models.TaskTypeGenerate, loadedTask.TaskTemplate.Type)

		// Verify payload
		payload, ok := loadedTask.TaskTemplate.Payload.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Generate a creative story", payload["prompt"])
		assert.Equal(t, "You are a creative writer", payload["system"])

		context, ok := payload["context"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, context, 3)
		assert.Equal(t, "fantasy", context[0])
	})

	t.Run("EmbedTaskPayload", func(t *testing.T) {
		cronTask := &models.CronTask{
			ID:       "embed-payload-test",
			Name:     "Embed Payload Test",
			CronExpr: "0 */6 * * *",
			Enabled:  false,
			TaskTemplate: models.TaskTemplate{
				Type:     models.TaskTypeEmbed,
				Model:    "qwen3",
				Priority: models.PriorityLow,
				Payload: map[string]interface{}{
					"input": []string{
						"First document to embed",
						"Second document to embed",
						"Third document to embed",
					},
					"truncate": true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := storage.SaveCronTask(cronTask)
		assert.NoError(t, err)

		loadedTask, err := storage.LoadCronTask("embed-payload-test")
		assert.NoError(t, err)
		assert.Equal(t, models.TaskTypeEmbed, loadedTask.TaskTemplate.Type)
		assert.False(t, loadedTask.Enabled)

		// Verify payload
		payload, ok := loadedTask.TaskTemplate.Payload.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, true, payload["truncate"])

		input, ok := payload["input"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, input, 3)
		assert.Equal(t, "First document to embed", input[0])
	})
}