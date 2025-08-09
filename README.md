# Ollama Queue

A high-performance **Client-Server task queue management system** for Ollama models, built in Go. Ollama Queue provides efficient task scheduling, priority management, persistence, and retry mechanisms for AI model operations with both library and standalone server usage.

**📖 [中文文档](README_zh.md)** | **🌟 [English](README.md)**

## Features

- 🚀 **High Performance**: Built with Go for optimal performance and concurrency
- 🏗️ **Client-Server Architecture**: Standalone server with HTTP API and WebSocket support
- 🌐 **Real-time Web UI**: Browser-based dashboard for queue monitoring and management
- 📋 **Priority Scheduling**: Four priority levels with FIFO ordering within each level
- 💾 **Persistent Storage**: BadgerDB-based storage with crash recovery
- 🔄 **Retry Mechanism**: Configurable retry with exponential backoff
- 🎯 **Multiple Task Types**: Support for chat, generate, and embed operations
- 📊 **Real-time Monitoring**: Task status tracking and queue statistics
- 🖥️ **CLI Interface**: Command-line tool for task management
- 📚 **Library Usage**: Use as a Go library in your applications
- 🌊 **Streaming Support**: Real-time streaming for chat and generation tasks
- 🔌 **HTTP Client**: Built-in client library for easy integration
- ☁️ **Remote Scheduling**: Supports offloading tasks to remote Ollama instances or OpenAI-compatible APIs, with improved stability and enhanced Web UI display.

## Quick Start

### Installation

```bash
go get github.com/liliang-cn/ollama-queue
```

### Server Mode (Recommended)

Start the server with web interface:

```bash
# Start server (default port 7125)
ollama-queue serve

# Start server on custom port
ollama-queue serve --port 9090

# Start server with custom data directory
ollama-queue serve --data-dir ./my-queue-data
```

Then open http://localhost:7125 in your browser to access the web interface.

### Client Mode

Use the CLI client to interact with the running server:

```bash
# Submit a chat task
ollama-queue submit chat --model llama2 --messages "user:Hello, how are you?" --priority high

# Submit a generation task
ollama-queue submit generate --model codellama --prompt "Write a Go function to sort an array"

# List all tasks
ollama-queue list

# Check queue status
ollama-queue status

# Cancel a task
ollama-queue cancel <task-id>

# Update task priority
ollama-queue priority <task-id> high
```

### HTTP Client Integration

Use the built-in HTTP client in your applications:

```go
package main

import (
	"fmt"
	"log"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// Connect to running server
	cli := client.New("localhost:7125")

	// Create and submit a chat task
	task := queue.NewChatTask("llama2", []models.ChatMessage{
		{Role: "user", Content: "Hello, how are you?"},
	}, queue.WithTaskPriority(models.PriorityHigh))

	taskID, err := cli.SubmitTask(task)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task submitted with ID: %s
", taskID)

	// Get task status
	taskInfo, err := cli.GetTask(taskID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task status: %s
", taskInfo.Status)
}
```

## Web Interface

The server provides a real-time web interface accessible at `http://localhost:7125`:

### Features
- **Task List**: View all tasks with real-time status updates
- **Task Submission**: Submit new tasks directly from the web interface
- **Priority Management**: Change task priorities on-the-fly
- **Task Cancellation**: Cancel running or pending tasks
- **Queue Statistics**: Monitor queue performance and status

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/` | Web interface |
| `GET` | `/ws` | WebSocket for real-time updates |
| `POST` | `/api/tasks` | Submit a new task |
| `GET` | `/api/tasks` | List all tasks |
| `GET` | `/api/tasks/:id` | Get specific task |
| `POST` | `/api/tasks/:id/cancel` | Cancel a task |
| `POST` | `/api/tasks/:id/priority` | Update task priority |
| `GET` | `/api/status` | Get queue statistics |

## Client-Server Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Browser   │    │   CLI Client    │    │   HTTP Client   │
│                 │    │                 │    │   Library       │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │ HTTP/WS              │ HTTP                  │ HTTP
          │                      │                       │
          └──────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────┴───────────────┐
                    │    Ollama Queue Server      │
                    │                             │
                    │  ┌─────────┐ ┌─────────────┐│
                    │  │ Web UI  │ │  HTTP API   ││
                    │  └─────────┘ └─────────────┘│
                    │                             │
                    │  ┌─────────┐ ┌─────────────┐│
                    │  │Priority │ │   Retry     ││
                    │  │Scheduler│ │  Scheduler  ││
                    │  └─────────┘ └─────────────┘│
                    │                             │
                    │  ┌─────────┐ ┌─────────────┐│
                    │  │ Storage │ │  Executor   ││
                    │  │(BadgerDB)│ │  (Ollama)   ││
                    │  └─────────┘ └─────────────┘│
                    └─────────────────────────────┘
```

## Task Types

### Chat Tasks
For conversational interactions with language models.

```go
task := queue.NewChatTask("llama2", []models.ChatMessage{
    {Role: "system", Content: "You are a helpful assistant"},
    {Role: "user", Content: "Explain quantum computing"},
}, 
    queue.WithTaskPriority(models.PriorityHigh),
    queue.WithChatSystem("You are a physics expert"),
    queue.WithChatStreaming(true),
)
```

### Generate Tasks
For text generation and completion.

```go
task := queue.NewGenerateTask("codellama", "Write a function to calculate fibonacci numbers",
    queue.WithTaskPriority(models.PriorityNormal),
    queue.WithGenerateSystem("You are a coding assistant"),
    queue.WithGenerateTemplate("### Response:
{{ .Response }}"),
)
```

### Embed Tasks
For creating text embeddings.

```go
task := queue.NewEmbedTask("nomic-embed-text", "This is a sample text for embedding",
    queue.WithTaskPriority(models.PriorityNormal),
    queue.WithEmbedTruncate(true),
)
```

## Priority Levels

The system supports four priority levels:

- **Critical (15)**: Highest priority, processed first
- **High (10)**: High priority tasks
- **Normal (5)**: Default priority level
- **Low (1)**: Lowest priority tasks

Tasks with higher priority are always processed before lower priority tasks. Within the same priority level, tasks are processed in FIFO order.

## Configuration

### Using Configuration File

Create a `config.yaml` file:

```yaml
ollama:
  host: "http://localhost:11434"
  timeout: "5m"

queue:
  max_workers: 4
  storage_path: "./data"
  cleanup_interval: "1h"
  max_completed_tasks: 1000

scheduler:
  batch_size: 10
  scheduling_interval: "1s"

retry_config:
  max_retries: 3
  initial_delay: "1s"
  max_delay: "30s"
  backoff_factor: 2.0

logging:
  level: "info"
  file: "ollama-queue.log"
```

### Environment Variables

```bash
export OLLAMA_HOST="http://localhost:11434"
export QUEUE_MAX_WORKERS=4
export QUEUE_STORAGE_PATH="./data"
export RETRY_MAX_RETRIES=3
export LOG_LEVEL="info"
```

## Library Integration (Embedded Mode)

You can also embed the queue manager directly in your applications:

### Basic Library Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// Create queue manager with default configuration
	qm, err := queue.NewQueueManagerWithOptions(
		queue.WithOllamaHost("http://localhost:11434"),
		queue.WithMaxWorkers(4),
		queue.WithStoragePath("./data"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer qm.Close()

	// Start the queue manager
	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Create and submit a chat task
	task := queue.NewChatTask("llama2", []models.ChatMessage{
		{Role: "user", Content: "Hello, how are you?"},
	}, queue.WithTaskPriority(models.PriorityHigh))

	taskID, err := qm.SubmitTask(task)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task submitted with ID: %s
", taskID)

	// Wait for completion using callback
	_, err = qm.SubmitTaskWithCallback(task, func(result *models.TaskResult) {
		if result.Success {
			fmt.Printf("Task completed successfully: %v
", result.Data)
		} else {
			fmt.Printf("Task failed: %s
", result.Error)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	// Wait for a while to let the task complete
	time.Sleep(10 * time.Second)
}
```


### Use in Web Applications

Integrate Ollama Queue client into your web services:

```go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

type ChatRequest struct {
	Message string `json:"message"`
	Model   string `json:"model"`
}

var queueClient *client.Client

func main() {
	// Connect to queue server
	queueClient = client.New("localhost:7125")

	r := gin.Default()
	r.POST("/chat", handleChat)
	r.GET("/task/:id", handleTaskStatus)

	r.Run(":3000")
}

func handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := queue.NewChatTask(req.Model, []models.ChatMessage{
		{Role: "user", Content: req.Message},
	})

	taskID, err := queueClient.SubmitTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "submitted",
	})
}

func handleTaskStatus(c *gin.Context) {
	taskID := c.Param("id")

	task, err := queueClient.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": task.ID,
		"status":  task.Status,
		"result":  task.Result,
		"error":   task.Error,
	})
}
```

### Async Task Processing

Create a task processor for handling background operations:

```go
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

type TaskProcessor struct {
	qm *queue.QueueManager
	wg sync.WaitGroup
}

func NewTaskProcessor() *TaskProcessor {
	qm, err := queue.NewQueueManagerWithOptions(
		queue.WithMaxWorkers(6),
		queue.WithStoragePath("./processor_data"),
	)
	if err != nil {
		panic(err)
	}

	processor := &TaskProcessor{qm: qm}

	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		panic(err)
	}

	return processor
}

func (tp *TaskProcessor) ProcessChatAsync(message, model string, callback func(string, error)) {
	task := queue.NewChatTask(model, []models.ChatMessage{
		{Role: "user", Content: message},
	})

	tp.wg.Add(1)
	_, err := tp.qm.SubmitTaskWithCallback(task, func(result *models.TaskResult) {
		defer tp.wg.Done()

		if result.Success {
			callback(fmt.Sprintf("%v", result.Data), nil)
		} else {
			callback("", fmt.Errorf(result.Error))
		}
	})

	if err != nil {
		tp.wg.Done()
		callback("", err)
	}
}

func (tp *TaskProcessor) ProcessBatch(messages []string, model string) ([]string, error) {
	var tasks []*models.Task

	for _, msg := range messages {
		task := queue.NewChatTask(model, []models.ChatMessage{
			{Role: "user", Content: msg},
		})
		tasks = append(tasks, task)
	}

	results, err := tp.qm.SubmitBatchTasks(tasks)
	if err != nil {
		return nil, err
	}

	var responses []string
	for _, result := range results {
		if result.Success {
			responses = append(responses, fmt.Sprintf("%v", result.Data))
		} else {
			responses = append(responses, fmt.Sprintf("Error: %s", result.Error))
		}
	}

	return responses, nil
}

func (tp *TaskProcessor) Close() error {
	tp.wg.Wait()
	return tp.qm.Close()
}

func main() {
	processor := NewTaskProcessor()
	defer processor.Close()

	// Example of async chat processing
	processor.ProcessChatAsync("Hello, how are you?", "llama2", func(response string, err error) {
		if err != nil {
			fmt.Printf("Async chat failed: %v
", err)
			return
		}
		fmt.Printf("Async chat response: %s
", response)
	})

	// Example of batch processing
	messages := []string{"First message", "Second message"}
	responses, err := processor.ProcessBatch(messages, "llama2")
	if err != nil {
		fmt.Printf("Batch processing failed: %v
", err)
	} else {
		fmt.Println("Batch processing responses:", responses)
	}

	// Wait for async operations to complete
	time.Sleep(10 * time.Second)
}
```

### Configuration Options

Complete configuration setup for production use:

```go
// Full configuration example
qm, err := queue.NewQueueManagerWithOptions(
    // Ollama configuration
    queue.WithOllamaHost("http://localhost:11434"),
    queue.WithOllamaTimeout(5*time.Minute),
    
    // Queue configuration
    queue.WithMaxWorkers(8),
    queue.WithStoragePath("./my_app_queue"),
    queue.WithCleanupInterval(time.Hour),
    queue.WithMaxCompletedTasks(1000),
    queue.WithSchedulingInterval(time.Second),
    
    // Retry configuration
    queue.WithRetryConfig(models.RetryConfig{
        MaxRetries:    3,
        InitialDelay:  time.Second,
        MaxDelay:      30 * time.Second,
        BackoffFactor: 2.0,
    }),
    
    // Logging configuration
    queue.WithLogLevel("info"),
    queue.WithLogFile("./logs/queue.log"),
)
```

## Advanced Usage

### Streaming Tasks

```go
// Submit streaming task
streamChan, err := qm.SubmitStreamingTask(task)
if err != nil {
    log.Fatal(err)
}

// Process streaming output
for chunk := range streamChan {
    if chunk.Error != nil {
        fmt.Printf("Stream error: %v
", chunk.Error)
        break
    }
    
    if chunk.Done {
        fmt.Println("
Stream completed")
        break
    }
    
    fmt.Print(chunk.Data)
}
```

### Batch Processing

```go
// Submit multiple tasks
tasks := []*models.Task{
    queue.NewChatTask("llama2", []models.ChatMessage{...}),
    queue.NewGenerateTask("codellama", "Write a function"),
    queue.NewEmbedTask("nomic-embed-text", "Sample text"),
}

// Process batch synchronously
results, err := qm.SubmitBatchTasks(tasks)
if err != nil {
    log.Fatal(err)
}

// Process results
for i, result := range results {
    fmt.Printf("Task %d: Success=%v, Data=%v
", i, result.Success, result.Data)
}
```

### Event Monitoring

```go
// Subscribe to task events
eventChan, err := qm.Subscribe([]string{"task_completed", "task_failed"})
if err != nil {
    log.Fatal(err)
}

// Monitor events
go func() {
    for event := range eventChan {
        fmt.Printf("Event: %s, Task: %s, Status: %s
", 
            event.Type, event.TaskID, event.Status)
    }
}()
```

## API Reference

### QueueManager Interface

```go
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
```

## CLI Commands

| Command | Description | Example |
|---------|-------------|---------|
| `serve` | Start the queue server with web interface | `ollama-queue serve --port 7125` |
| `submit` | Submit a new task to the server | `ollama-queue submit chat --model llama2 --messages "user:Hello"` |
| `list` | List tasks with optional filtering | `ollama-queue list --status running --limit 10` |
| `status` | Show task status or queue statistics | `ollama-queue status <task-id>` |
| `cancel` | Cancel one or more tasks | `ollama-queue cancel <task-id1> <task-id2>` |
| `priority` | Update task priority | `ollama-queue priority <task-id> high` |

## Architecture

The system is designed with a flexible Client-Server architecture that supports multiple usage patterns:

### Server Mode
- Standalone HTTP server with REST API
- Real-time WebSocket communication
- Built-in web interface for monitoring
- Persistent task storage with BadgerDB

### Client Integration
- HTTP client library for programmatic access
- CLI tool for command-line operations
- Direct library integration for embedded usage

## Performance

- **Throughput**: Handles thousands of tasks per second
- **Latency**: Sub-millisecond task scheduling
- **Memory**: Efficient memory usage with configurable limits
- **Storage**: Persistent storage with automatic cleanup
- **Concurrency**: Configurable worker pool for optimal resource utilization

## Error Handling

The system provides comprehensive error handling:

- **Task Failures**: Automatic retry with exponential backoff
- **Connection Errors**: Graceful degradation and recovery
- **Storage Errors**: Data consistency and corruption protection
- **Resource Limits**: Queue size and memory limits with backpressure

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `go test ./...`
6. Submit a pull request

## Testing

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test ./... -cover
```

Run specific package tests:
```bash
go test ./pkg/scheduler -v
go test ./pkg/storage -v
go test ./internal/models -v
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 Documentation: [docs/](docs/) | [中文文档](README_zh.md)
- 🐛 Issues: [GitHub Issues](https://github.com/liliang-cn/ollama-queue/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/liliang-cn/ollama-queue/discussions)

## Related Projects

- [Ollama](https://github.com/ollama/ollama) - Run large language models locally
- [Ollama-Go](https://github.com/liliang-cn/ollama-go) - Go client for Ollama
