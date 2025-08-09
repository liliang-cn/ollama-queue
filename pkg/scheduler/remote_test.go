package scheduler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/stretchr/testify/assert"
)

// MockStorage is a mock implementation of the storage.Storage interface
type MockStorage struct {
	SaveTaskFunc func(task *models.Task) error
	LoadTaskFunc func(taskID string) (*models.Task, error)
}

func (m *MockStorage) SaveTask(task *models.Task) error {
	if m.SaveTaskFunc != nil {
		return m.SaveTaskFunc(task)
	}
	return nil
}

func (m *MockStorage) LoadTask(taskID string) (*models.Task, error) {
	if m.LoadTaskFunc != nil {
		return m.LoadTaskFunc(taskID)
	}
	return nil, nil
}

func (m *MockStorage) Open() error                                                        { return nil }
func (m *MockStorage) Close() error                                                       { return nil }
func (m *MockStorage) GetTasksByStatus(status models.TaskStatus) ([]*models.Task, error)    { return nil, nil }
func (m *MockStorage) ListTasks(filter models.TaskFilter) ([]*models.Task, error)         { return nil, nil }
func (m *MockStorage) GetStats() (*models.QueueStats, error)                              { return nil, nil }
func (m *MockStorage) CleanupCompletedTasks(maxTasks int, maxAge time.Duration) error       { return nil }
func (m *MockStorage) UpdateTaskStatus(taskID string, status models.TaskStatus) error { return nil }
func (m *MockStorage) DeleteTask(taskID string) error { return nil }

// MockScheduler is a mock implementation of the Scheduler interface for testing
type MockScheduler struct {
	QueueLength int
	AddTaskFunc func(task *models.Task) error
}

func (m *MockScheduler) AddTask(task *models.Task) error {
	if m.AddTaskFunc != nil {
		return m.AddTaskFunc(task)
	}
	return nil
}
func (m *MockScheduler) GetNextTask() *models.Task                                        { return nil }
func (m *MockScheduler) RemoveTask(taskID string) error                                     { return nil }
func (m *MockScheduler) UpdateTaskPriority(taskID string, priority models.Priority) error { return nil }
func (m *MockScheduler) GetQueueLength() int                                              { return m.QueueLength }
func (m *MockScheduler) GetQueueStats() map[models.Priority]int                          { return nil }

func TestNewRemoteScheduler(t *testing.T) {
	storage := &MockStorage{}
	fallback := NewPriorityScheduler()
	config := RemoteSchedulerConfig{
		Endpoints: []RemoteEndpoint{
			{Name: "test", BaseURL: "http://localhost", Priority: 10, Available: true},
		},
		HealthCheckInterval: 1 * time.Minute,
		FallbackScheduler:   fallback,
		MaxLocalQueueSize:   50,
		LocalFirstPolicy:    true,
		Storage:             storage,
	}

	rs := NewRemoteScheduler(config)

	assert.NotNil(t, rs)
	assert.Equal(t, config.FallbackScheduler, rs.fallback)
	assert.Equal(t, config.MaxLocalQueueSize, rs.maxLocalQueueSize)
	assert.Equal(t, config.LocalFirstPolicy, rs.localFirstPolicy)
	assert.Equal(t, config.Storage, rs.storage)
	assert.True(t, rs.hasClient, "Client should be initialized")
}

func TestShouldUseRemote(t *testing.T) {
	tests := []struct {
		name              string
		localFirstPolicy  bool
		remoteExecution   bool
		priority          models.Priority
		localQueueSize    int
		maxLocalQueueSize int
		expected          bool
	}{
		{"LocalFirst_RemoteExecutionFlag", true, true, models.PriorityNormal, 0, 100, true},
		{"LocalFirst_CriticalPriority", true, false, models.PriorityCritical, 0, 100, true},
		{"LocalFirst_QueueFull_HighPriority", true, false, models.PriorityHigh, 100, 100, true},
		{"LocalFirst_QueueNotFull", true, false, models.PriorityHigh, 50, 100, false},
		{"LocalFirst_QueueFull_NormalPriority", true, false, models.PriorityNormal, 100, 100, false},
		{"NoLocalFirst_HighPriority", false, false, models.PriorityHigh, 0, 100, true},
		{"NoLocalFirst_NormalPriority", false, false, models.PriorityNormal, 0, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up a mock fallback scheduler with the desired queue size
			mockFallback := &MockScheduler{QueueLength: tt.localQueueSize}

			rs := &RemoteScheduler{
				localFirstPolicy:  tt.localFirstPolicy,
				maxLocalQueueSize: tt.maxLocalQueueSize,
				fallback:          mockFallback,
			}

			task := &models.Task{
				ID:              "test-task",
				Type:            models.TaskTypeChat,
				Priority:        tt.priority,
				RemoteExecution: tt.remoteExecution,
			}

			assert.Equal(t, tt.expected, rs.shouldUseRemote(task))
		})
	}
}

func TestAddTask_RemoteChatSuccess(t *testing.T) {
	// Create a test server to mock the OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := openai.ChatCompletion{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-3.5-turbo-0613",
			Choices: []openai.ChatCompletionChoice{
				{
					Index:        0,
					Message:      openai.ChatCompletionMessage{Role: "assistant", Content: "Hello there!"},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	saved := false
	storage := &MockStorage{
		SaveTaskFunc: func(task *models.Task) error {
			if task.Status == models.StatusCompleted {
				saved = true
				assert.Equal(t, "chatcmpl-123", task.Result.(*models.TaskResult).Data.(*openai.ChatCompletion).ID)
			}
			return nil
		},
	}

	client := openai.NewClient(option.WithBaseURL(server.URL))
	rs := &RemoteScheduler{
		storage:          storage,
		localFirstPolicy: false, // Ensure remote is preferred
		hasClient:        true,
		client:           client,
		endpoints: []RemoteEndpoint{
			{Name: "test-server", BaseURL: server.URL, APIKey: "", Priority: 1, Available: true},
		},
	}

	task := &models.Task{
		ID:       "test-task",
		Type:     models.TaskTypeChat,
		Priority: models.PriorityHigh,
		Payload:  models.ChatTaskPayload{Messages: []models.ChatMessage{{Role: "user", Content: "hello"}}},
	}

	err := rs.AddTask(task)

	assert.NoError(t, err)
	assert.True(t, saved, "Task should have been saved with completed status")
}

func TestAddTask_RemoteFailure_Fallback(t *testing.T) {
	// Create a test server that always returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fellback := false
	fallbackScheduler := &MockScheduler{
		AddTaskFunc: func(task *models.Task) error {
			fellback = true
			return nil
		},
	}

	client := openai.NewClient(option.WithBaseURL(server.URL))
	rs := &RemoteScheduler{
		storage:          &MockStorage{},
		localFirstPolicy: false, // Ensure remote is preferred
		hasClient:        true,
		client:           client,
		fallback:         fallbackScheduler,
	}

	task := &models.Task{
		ID:       "test-task",
		Type:     models.TaskTypeChat,
		Priority: models.PriorityHigh,
		Payload:  models.ChatTaskPayload{Messages: []models.ChatMessage{{Role: "user", Content: "hello"}}},
	}

	err := rs.AddTask(task)

	assert.NoError(t, err)
	assert.True(t, fellback, "Task should have been passed to the fallback scheduler")
}
