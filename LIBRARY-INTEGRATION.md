# Library Integration Guide

This guide shows how to embed ollama-queue into your Go applications. The client-server separation actually makes library integration more flexible.

## 🎯 Integration Options

### 1. Full Queue Manager Integration

Best for: Standalone applications that need full queue capabilities

```go
import "github.com/liliang-cn/ollama-queue/pkg/queue"
import "github.com/liliang-cn/ollama-queue/internal/models"

// Embed complete queue management
manager, err := queue.NewQueueManager(config)
manager.Start(ctx)
defer manager.Close()

// Use directly in your app
taskID, err := manager.SubmitTask(task)
```

**Benefits:**
- Full control over queue behavior
- No external dependencies
- Custom configuration
- Direct API access

### 2. Server + API Integration

Best for: Web applications that want to expose queue functionality

```go
// Embed server functionality into your app
app := &MyApp{queueManager: manager}

// Add queue endpoints to your existing API
queueAPI := router.Group("/api/queue")
queueAPI.POST("/tasks", app.submitTask)
queueAPI.GET("/stats", app.getStats)
```

**Benefits:**
- Queue API integrated with your app
- Single deployment unit
- Shared authentication/middleware
- Custom endpoints

### 3. Client Library Integration

Best for: Microservices connecting to external queue server

```go
import "github.com/liliang-cn/ollama-queue/pkg/client"

// Connect to external queue server
client := client.New("queue-server:8080")
taskID, err := client.SubmitTask(task)
```

**Benefits:**
- Lightweight dependency
- Service separation
- Easy to test/mock
- Minimal resource usage

## 📦 Available Packages

### Core Packages

```go
// Queue management (full functionality)
import "github.com/liliang-cn/ollama-queue/pkg/queue"

// HTTP client (lightweight)
import "github.com/liliang-cn/ollama-queue/pkg/client"

// Data models
import "github.com/liliang-cn/ollama-queue/internal/models"

// Configuration
import "github.com/liliang-cn/ollama-queue/pkg/config"
```

### Specialized Packages

```go
// Storage backends
import "github.com/liliang-cn/ollama-queue/pkg/storage"

// Task scheduling
import "github.com/liliang-cn/ollama-queue/pkg/scheduler"

// Model execution
import "github.com/liliang-cn/ollama-queue/pkg/executor"

// UI components (for web integration)
import "github.com/liliang-cn/ollama-queue/pkg/ui"
```

## 🔧 Configuration Options

### Minimal Configuration

```go
config := models.DefaultConfig()
config.StoragePath = "./my-queue-data"
config.OllamaHost = "http://localhost:11434"
```

### Advanced Configuration

```go
config := &models.Config{
    StoragePath:        "./data",
    OllamaHost:        "http://ollama:11434",
    ListenAddr:        "0.0.0.0:8080",
    MaxWorkers:        10,
    SchedulingInterval: 5 * time.Second,
    CleanupInterval:   1 * time.Hour,
    MaxCompletedTasks: 1000,
    RetryConfig: models.RetryConfig{
        MaxRetries:    3,
        InitialDelay:  1 * time.Second,
        BackoffFactor: 2.0,
        MaxDelay:      30 * time.Second,
    },
    RemoteScheduling: models.RemoteSchedulingConfig{
        Enabled:           true,
        LocalFirstPolicy:  true,
        MaxLocalQueueSize: 100,
        Endpoints: []models.RemoteEndpoint{
            {
                Name:     "backup-server",
                BaseURL:  "http://backup:8080",
                Priority: 1,
                Enabled:  true,
            },
        },
    },
}
```

## 🎯 Common Integration Patterns

### 1. Background Job Processor

```go
type JobProcessor struct {
    queue *queue.QueueManager
}

func (p *JobProcessor) ProcessEmail(email EmailData) error {
    task := &models.Task{
        Type: models.TaskTypeGenerate,
        Model: "qwen3",
        Payload: map[string]interface{}{
            "prompt": fmt.Sprintf("Generate email response for: %s", email.Subject),
        },
    }
    
    taskID, err := p.queue.SubmitTask(task)
    // Store taskID for later retrieval
    return err
}
```

### 2. API Gateway with Queue

```go
func (app *App) handleUserRequest(c *gin.Context) {
    // Validate request
    var req UserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Submit to queue
    task := &models.Task{
        Type: models.TaskTypeChat,
        Model: req.Model,
        Payload: req.Messages,
        Options: map[string]interface{}{
            "user_id": req.UserID,
        },
    }
    
    taskID, err := app.queue.SubmitTask(task)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(202, gin.H{
        "task_id": taskID,
        "status": "queued",
    })
}
```

### 3. Batch Processing Service

```go
func (s *BatchService) ProcessDocuments(docs []Document) error {
    tasks := make([]*models.Task, len(docs))
    
    for i, doc := range docs {
        tasks[i] = &models.Task{
            Type: models.TaskTypeEmbed,
            Model: "qwen3",
            Payload: map[string]interface{}{
                "input": doc.Content,
            },
            Options: map[string]interface{}{
                "doc_id": doc.ID,
            },
        }
    }
    
    // Submit batch
    results, err := s.client.SubmitBatchTasks(tasks)
    return s.processBatchResults(results, err)
}
```

### 4. Scheduled Task Service

```go
func (s *SchedulerService) SetupRecurringTasks() error {
    cronTasks := []*models.CronTask{
        {
            Name:     "Daily Backup",
            CronExpr: "0 2 * * *", // 2 AM daily
            TaskTemplate: models.TaskTemplate{
                Type: models.TaskTypeGenerate,
                Model: "qwen3",
                Payload: map[string]interface{}{
                    "prompt": "Generate backup report",
                },
            },
        },
        {
            Name:     "Weekly Analytics",
            CronExpr: "0 9 * * 1", // 9 AM Mondays
            TaskTemplate: models.TaskTemplate{
                Type: models.TaskTypeGenerate,
                Model: "qwen3",
                Payload: map[string]interface{}{
                    "prompt": "Generate weekly analytics",
                },
            },
        },
    }
    
    for _, cronTask := range cronTasks {
        _, err := s.client.AddCronTask(cronTask)
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

## 🔄 Task Result Handling

### Synchronous Processing

```go
// Submit and wait for result
taskID, err := manager.SubmitTask(task)
if err != nil {
    return err
}

// Poll for completion
for {
    task, err := manager.GetTask(taskID)
    if err != nil {
        return err
    }
    
    if task.Status == models.StatusCompleted {
        return processResult(task.Result)
    } else if task.Status == models.StatusFailed {
        return fmt.Errorf("task failed: %s", task.Error)
    }
    
    time.Sleep(1 * time.Second)
}
```

### Asynchronous Processing with Callbacks

```go
// Submit with callback
taskID, err := manager.SubmitTaskWithCallback(task, func(result *models.TaskResult) {
    if result.Success {
        log.Printf("Task %s completed: %v", result.TaskID, result.Data)
        // Process result
    } else {
        log.Printf("Task %s failed: %s", result.TaskID, result.Error)
        // Handle error
    }
})
```

### Event-Based Processing

```go
// Subscribe to events
eventChan, err := manager.Subscribe([]string{
    "task_completed", 
    "task_failed",
    "task_cancelled",
})

go func() {
    for event := range eventChan {
        switch event.Type {
        case "task_completed":
            handleTaskCompletion(event)
        case "task_failed":
            handleTaskFailure(event)
        }
    }
}()
```

## 🧪 Testing Integration

### Mock Queue Manager

```go
type MockQueueManager struct {
    tasks map[string]*models.Task
}

func (m *MockQueueManager) SubmitTask(task *models.Task) (string, error) {
    taskID := "test-" + uuid.New().String()
    task.ID = taskID
    task.Status = models.StatusPending
    m.tasks[taskID] = task
    return taskID, nil
}

func (m *MockQueueManager) GetTask(taskID string) (*models.Task, error) {
    if task, exists := m.tasks[taskID]; exists {
        return task, nil
    }
    return nil, fmt.Errorf("task not found")
}
```

### Integration Tests

```go
func TestQueueIntegration(t *testing.T) {
    // Setup test queue
    config := models.DefaultConfig()
    config.StoragePath = t.TempDir()
    
    manager, err := queue.NewQueueManager(config)
    require.NoError(t, err)
    defer manager.Close()
    
    err = manager.Start(context.Background())
    require.NoError(t, err)
    
    // Test task submission
    task := &models.Task{
        Type: models.TaskTypeGenerate,
        Model: "test-model",
        Payload: map[string]interface{}{
            "prompt": "test prompt",
        },
    }
    
    taskID, err := manager.SubmitTask(task)
    require.NoError(t, err)
    require.NotEmpty(t, taskID)
    
    // Verify task was queued
    retrievedTask, err := manager.GetTask(taskID)
    require.NoError(t, err)
    assert.Equal(t, models.StatusPending, retrievedTask.Status)
}
```

## 📚 Best Practices

### 1. Resource Management

```go
// Always close resources
defer manager.Close()

// Use contexts for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 2. Error Handling

```go
// Handle specific error types
if err != nil {
    switch {
    case errors.Is(err, models.ErrTaskNotFound):
        // Handle missing task
    case errors.Is(err, models.ErrInvalidTask):
        // Handle validation error
    default:
        // Handle other errors
    }
}
```

### 3. Configuration Management

```go
// Use environment-specific configs
var config *models.Config
switch os.Getenv("ENV") {
case "production":
    config = loadProductionConfig()
case "staging":
    config = loadStagingConfig()
default:
    config = models.DefaultConfig()
}
```

This flexible architecture allows you to choose the integration approach that best fits your application's needs!