# Ollama Queue

**Universal Task Queue Go Library**

A flexible task execution library supporting multiple executor plugins including Ollama, OpenAI, script execution, and more. Fully decoupled architecture with support for both synchronous and streaming task execution.

**🌟 [English](README.md)** | **📖 [中文文档](README_zh.md)**

## ✨ Features

- 🚀 **Universal Task Queue** - Not limited to specific execution engines
- 🔌 **Plugin Architecture** - Support for multiple executor types (Ollama, OpenAI, Script, HTTP)
- ⚡ **Sync & Stream Execution** - Real-time output streaming support
- 🛡️ **Type Safe** - Strongly typed task and configuration system
- 📦 **Pure Go Library** - No external dependencies, easy to embed in other projects

## 🚀 Quick Start

### Installation

```bash
go get github.com/liliang-cn/ollama-queue
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/liliang-cn/ollama-queue/pkg/models"
    "github.com/liliang-cn/ollama-queue/pkg/executor"
)

func main() {
    // 1. Create executor registry
    registry := executor.NewExecutorRegistry()
    
    // 2. Register Ollama executor
    config := models.DefaultConfig()
    ollamaPlugin, err := executor.NewOllamaPlugin(config)
    if err != nil {
        log.Fatal(err)
    }
    registry.Register(models.ExecutorTypeOllama, ollamaPlugin)
    
    // 3. Create chat task
    task := models.CreateOllamaTask(
        models.ActionChat,
        "qwen3",
        map[string]interface{}{
            "messages": []models.ChatMessage{
                {Role: "user", Content: "Hello!"},
            },
        },
    )
    
    // 4. Execute task
    result, err := registry.ExecuteTask(context.Background(), task)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Result: %+v", result.Data)
}
```

## Supported Executors

### Ollama Executor

```go
// Chat task
chatTask := models.CreateOllamaTask(
    models.ActionChat,
    "llama3",
    map[string]interface{}{
        "messages": []models.ChatMessage{
            {Role: "user", Content: "Explain Go goroutines"},
        },
        "system": "You are a Go language expert",
    },
)

// Text generation task
generateTask := models.CreateOllamaTask(
    models.ActionGenerate,
    "llama3",
    map[string]interface{}{
        "prompt": "Write a quicksort algorithm",
        "system": "Please implement in Go language",
    },
)

// Text embedding task
embedTask := models.CreateOllamaTask(
    models.ActionEmbed,
    "nomic-embed-text",
    map[string]interface{}{
        "input": "This is the text to embed",
    },
)
```

### Script Executor

```go
// Configure script executor
scriptConfig := executor.ScriptConfig{
    WorkingDir:  "./scripts",
    AllowedExts: []string{".py", ".sh", ".js"},
}
scriptPlugin, _ := executor.NewScriptPlugin(scriptConfig)
registry.Register(models.ExecutorTypeScript, scriptPlugin)

// Execute command
commandTask := models.CreateGenericTask(
    models.ExecutorTypeScript,
    models.ActionExecute,
    map[string]interface{}{
        "command": "echo 'Hello from script!'",
    },
)

// Execute script file
scriptTask := models.CreateGenericTask(
    models.ExecutorTypeScript,
    models.ActionExecute,
    map[string]interface{}{
        "script": "process_data.py",
        "args":   []string{"--input", "data.txt"},
        "env": map[string]string{
            "PYTHONPATH": "/opt/myapp",
        },
    },
)
```

### OpenAI Executor

```go
// Configure OpenAI executor
openaiConfig := executor.OpenAIConfig{
    APIKey:  "your-api-key",
    BaseURL: "https://api.openai.com/v1",
}
openaiPlugin, _ := executor.NewOpenAIPlugin(openaiConfig)
registry.Register(models.ExecutorTypeOpenAI, openaiPlugin)

// Chat task
openaiTask := models.CreateGenericTask(
    models.ExecutorTypeOpenAI,
    models.ActionChat,
    map[string]interface{}{
        "messages": []models.ChatMessage{
            {Role: "user", Content: "Explain machine learning"},
        },
    },
)
openaiTask.Model = "gpt-4"
```

## Streaming Execution

```go
// Create streaming task
task := models.CreateOllamaTask(
    models.ActionChat,
    "qwen3",
    map[string]interface{}{
        "messages": []models.ChatMessage{
            {Role: "user", Content: "Write a poem"},
        },
    },
)

// Execute streaming task
streamChan, err := registry.ExecuteStreamTask(context.Background(), task)
if err != nil {
    log.Fatal(err)
}

// Handle streaming output
for chunk := range streamChan {
    if chunk.Error != nil {
        log.Printf("Error: %v", chunk.Error)
        break
    }
    
    if chunk.Data != nil {
        fmt.Print(chunk.Data)
    }
    
    if chunk.Done {
        fmt.Println("\nCompleted")
        break
    }
}
```

## Core Concepts

### GenericTask

Universal task structure supporting all executor types:

```go
type GenericTask struct {
    ID           string                 // Task ID
    ExecutorType ExecutorType           // Executor type
    Action       ExecutorAction         // Execution action
    Payload      map[string]interface{} // Task payload
    Model        string                 // Model name (optional)
    Priority     Priority               // Task priority
    Status       TaskStatus             // Task status
    // ... timestamp and result fields
}
```

### ExecutorRegistry

Executor registry managing all executor plugins:

```go
registry := executor.NewExecutorRegistry()

// Register executor
registry.Register(models.ExecutorTypeOllama, ollamaPlugin)

// Check if task can be executed
if registry.CanExecuteTask(task) {
    result, err := registry.ExecuteTask(ctx, task)
}

// List all registered executors
executors := registry.ListExecutors()
```

## Task Status and Priority

### Task Status

- `StatusPending` - Waiting for execution
- `StatusRunning` - Currently executing
- `StatusCompleted` - Execution completed
- `StatusFailed` - Execution failed
- `StatusCancelled` - Cancelled

### Task Priority

- `PriorityLow` (1) - Low priority
- `PriorityNormal` (5) - Normal priority
- `PriorityHigh` (10) - High priority
- `PriorityCritical` (15) - Critical priority

## Example Code

### Basic Usage Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/liliang-cn/ollama-queue/pkg/models"
    "github.com/liliang-cn/ollama-queue/pkg/executor"
)

func main() {
    // Create executor registry
    registry := executor.NewExecutorRegistry()
    
    // Register Ollama executor
    config := models.DefaultConfig()
    ollamaPlugin, err := executor.NewOllamaPlugin(config)
    if err != nil {
        log.Fatal(err)
    }
    registry.Register(models.ExecutorTypeOllama, ollamaPlugin)
    
    // Register script executor
    scriptConfig := executor.ScriptConfig{
        WorkingDir:  "./",
        AllowedExts: []string{".sh", ".py", ".js"},
    }
    scriptPlugin, err := executor.NewScriptPlugin(scriptConfig)
    if err != nil {
        log.Fatal(err)
    }
    registry.Register(models.ExecutorTypeScript, scriptPlugin)
    
    // Execute script task
    task := models.CreateGenericTask(
        models.ExecutorTypeScript,
        models.ActionExecute,
        map[string]interface{}{
            "command": "echo 'Hello from ollama-queue library!'",
        },
    )
    
    result, err := registry.ExecuteTask(context.Background(), task)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Task execution result: %+v\n", result.Data)
}
```

## License

MIT License