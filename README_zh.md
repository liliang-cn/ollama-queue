# Ollama Queue

基于Go语言构建的高性能 **客户端-服务器任务队列管理系统**。Ollama Queue 提供了高效的任务调度、优先级管理、持久化存储和重试机制，支持库集成和独立服务器两种使用方式。

**🌟 [English](README.md)** | **📖 [中文文档](README_zh.md)**

## 特性

- 🚀 **高性能**: 使用Go语言构建，具有最佳性能和并发能力
- 🏗️ **客户端-服务器架构**: 独立服务器，支持HTTP API和WebSocket
- 🌐 **实时Web界面**: 基于浏览器的队列监控和管理仪表板
- 📋 **优先级调度**: 四级优先级系统，同级别内采用FIFO排序
- 💾 **持久化存储**: 基于BadgerDB的存储系统，支持崩溃恢复
- 🔄 **重试机制**: 可配置的重试机制，支持指数退避算法
- 🎯 **多任务类型**: 支持聊天、生成和嵌入操作
- 📊 **实时监控**: 任务状态跟踪和队列统计
- 🖥️ **CLI接口**: 命令行工具进行任务管理
- 📚 **库集成**: 可作为Go库在应用程序中使用
- 🌊 **流式支持**: 聊天和生成任务的实时流式输出
- 🔌 **HTTP客户端**: 内置客户端库，便于集成

## 快速开始

### 安装

```bash
go get github.com/liliang-cn/ollama-queue
```

### 服务器模式 (推荐)

启动带有Web界面的服务器：

```bash
# 启动服务器 (默认端口 7125)
ollama-queue serve

# 在自定义端口启动服务器
ollama-queue serve --port 9090

# 使用自定义数据目录启动服务器
ollama-queue serve --data-dir ./my-queue-data
```

然后在浏览器中打开 http://localhost:7125 访问Web界面。

### 客户端模式

使用CLI客户端与运行中的服务器交互：

```bash
# 提交聊天任务
ollama-queue submit chat --model llama2 --messages "user:你好，你能帮我什么？" --priority high

# 提交生成任务
ollama-queue submit generate --model codellama --prompt "编写一个Go函数来排序数组"

# 列出所有任务
ollama-queue list

# 检查队列状态
ollama-queue status

# 取消任务
ollama-queue cancel <task-id>

# 更新任务优先级
ollama-queue priority <task-id> high
```

### HTTP客户端集成

在应用程序中使用内置的HTTP客户端：

```go
package main

import (
    "fmt"
    "log"

    "github.com/liliang-cn/ollama-queue/pkg/client"
    "github.com/liliang-cn/ollama-queue/internal/models"
    "github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
    // 连接到运行中的服务器
    cli := client.New("localhost:7125")

    // 创建并提交聊天任务
    task := queue.NewChatTask("llama2", []models.ChatMessage{
        {Role: "user", Content: "你好，你是谁？"},
    }, queue.WithTaskPriority(models.PriorityHigh))

    taskID, err := cli.SubmitTask(task)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("任务已提交，ID: %s\n", taskID)

    // 获取任务状态
    taskInfo, err := cli.GetTask(taskID)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("任务状态: %s\n", taskInfo.Status)
}
```

## Web界面

服务器提供了一个实时Web界面，可通过 `http://localhost:7125` 访问：

### 功能特性
- **任务列表**: 查看所有任务及实时状态更新
- **任务提交**: 直接从Web界面提交新任务
- **优先级管理**: 实时调整任务优先级
- **任务取消**: 取消正在运行或等待中的任务
- **队列统计**: 监控队列性能和状态

### API端点

| 方法 | 端点 | 描述 |
|------|------|------|
| `GET` | `/` | Web界面 |
| `GET` | `/ws` | WebSocket实时更新 |
| `POST` | `/api/tasks` | 提交新任务 |
| `GET` | `/api/tasks` | 列出所有任务 |
| `GET` | `/api/tasks/:id` | 获取特定任务 |
| `POST` | `/api/tasks/:id/cancel` | 取消任务 |
| `POST` | `/api/tasks/:id/priority` | 更新任务优先级 |
| `GET` | `/api/status` | 获取队列统计 |

## 客户端-服务器架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web浏览器     │    │   CLI客户端     │    │   HTTP客户端    │
│                 │    │                 │    │   库           │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │ HTTP/WS              │ HTTP                  │ HTTP
          │                      │                       │
          └──────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────┴───────────────┐
                    │    Ollama Queue服务器       │
                    │                             │
                    │  ┌─────────┐ ┌─────────────┐│
                    │  │ Web UI  │ │  HTTP API   ││
                    │  └─────────┘ └─────────────┘│
                    │                             │
                    │  ┌─────────┐ ┌─────────────┐│
                    │  │优先级   │ │   重试      ││
                    │  │调度器   │ │  调度器     ││
                    │  └─────────┘ └─────────────┘│
                    │                             │
                    │  ┌─────────┐ ┌─────────────┐│
                    │  │ 存储    │ │  执行器     ││
                    │  │(BadgerDB)│ │  (Ollama)   ││
                    │  └─────────┘ └─────────────┘│
                    └─────────────────────────────┘
```

## 任务类型

### 聊天任务
用于与语言模型进行对话交互。

```go
task := queue.NewChatTask("llama2", []models.ChatMessage{
    {Role: "system", Content: "你是一个有用的助手"},
    {Role: "user", Content: "解释一下量子计算"},
}, 
    queue.WithTaskPriority(models.PriorityHigh),
    queue.WithChatSystem("你是一个物理学专家"),
    queue.WithChatStreaming(true),
)
```

### 生成任务
用于文本生成和补全。

```go
task := queue.NewGenerateTask("codellama", "编写一个计算斐波那契数列的函数",
    queue.WithTaskPriority(models.PriorityNormal),
    queue.WithGenerateSystem("你是一个编程助手"),
    queue.WithGenerateTemplate("### 回复:\n{{ .Response }}"),
)
```

### 嵌入任务
用于创建文本嵌入向量。

```go
task := queue.NewEmbedTask("nomic-embed-text", "这是用于嵌入的示例文本",
    queue.WithTaskPriority(models.PriorityNormal),
    queue.WithEmbedTruncate(true),
)
```

## 优先级系统

系统支持四个优先级等级：

- **关键 (15)**: 最高优先级，最先处理
- **高 (10)**: 高优先级任务
- **普通 (5)**: 默认优先级等级
- **低 (1)**: 最低优先级任务

高优先级任务总是在低优先级任务之前处理。在同一优先级内，任务按照FIFO顺序处理。

## 配置

### 使用配置文件

创建 `config.yaml` 文件：

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

### 环境变量

```bash
export OLLAMA_HOST="http://localhost:11434"
export QUEUE_MAX_WORKERS=4
export QUEUE_STORAGE_PATH="./data"
export RETRY_MAX_RETRIES=3
export LOG_LEVEL="info"
```

## 库集成 (嵌入模式)

您也可以将队列管理器直接嵌入到应用程序中：

### 基本库使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/liliang-cn/ollama-queue/internal/models"
    "github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
    // 创建队列管理器
    qm, err := queue.NewQueueManagerWithOptions(
        queue.WithOllamaHost("http://localhost:11434"),
        queue.WithMaxWorkers(4),
        queue.WithStoragePath("./data"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer qm.Close()

    // 启动队列管理器
    ctx := context.Background()
    if err := qm.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // 创建并提交聊天任务
    task := queue.NewChatTask("llama2", []models.ChatMessage{
        {Role: "user", Content: "你好，你是谁？"},
    }, queue.WithTaskPriority(models.PriorityHigh))

    taskID, err := qm.SubmitTask(task)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("任务已提交，ID: %s\n", taskID)

    // 使用回调等待完成
    _, err = qm.SubmitTaskWithCallback(task, func(result *models.TaskResult) {
        if result.Success {
            fmt.Printf("任务完成成功: %v\n", result.Data)
        } else {
            fmt.Printf("任务失败: %s\n", result.Error)
        }
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### 在Web应用中使用

将Ollama Queue客户端集成到您的Web服务中：

```go
package main

import (
    "encoding/json"
    "net/http"
    "log"

    "github.com/gin-gonic/gin"
    "github.com/liliang-cn/ollama-queue/pkg/client"
    "github.com/liliang-cn/ollama-queue/internal/models"
    "github.com/liliang-cn/ollama-queue/pkg/queue"
)

type ChatRequest struct {
    Message string `json:"message"`
    Model   string `json:"model"`
}

var queueClient *client.Client

func main() {
    // 连接到队列服务器
    queueClient = client.New("localhost:7125")

    r := gin.Default()
    r.POST("/chat", handleChat)
    r.GET("/task/:id", handleTaskStatus)
    
    r.Run(":3000")
}

func handleChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    task := queue.NewChatTask(req.Model, []models.ChatMessage{
        {Role: "user", Content: req.Message},
    })

    taskID, err := queueClient.SubmitTask(task)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "task_id": taskID,
        "status": "submitted",
    })
}

func handleTaskStatus(c *gin.Context) {
    taskID := c.Param("id")
    
    task, err := queueClient.GetTask(taskID)
    if err != nil {
        c.JSON(404, gin.H{"error": "任务未找到"})
        return
    }

    c.JSON(200, gin.H{
        "task_id": task.ID,
        "status": task.Status,
        "result": task.Result,
        "error": task.Error,
    })
}
```

### 异步任务处理

创建任务处理器来处理后台操作：

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
            responses = append(responses, fmt.Sprintf("错误: %s", result.Error))
        }
    }

    return responses, nil
}

func (tp *TaskProcessor) Close() error {
    tp.wg.Wait()
    return tp.qm.Close()
}
```

### 配置选项

生产环境的完整配置设置：

```go
// 完整配置示例
qm, err := queue.NewQueueManagerWithOptions(
    // Ollama 配置
    queue.WithOllamaHost("http://localhost:11434"),
    queue.WithOllamaTimeout(5*time.Minute),
    
    // 队列配置
    queue.WithMaxWorkers(8),
    queue.WithStoragePath("./my_app_queue"),
    queue.WithCleanupInterval(time.Hour),
    queue.WithMaxCompletedTasks(1000),
    queue.WithSchedulingInterval(time.Second),
    
    // 重试配置
    queue.WithRetryConfig(models.RetryConfig{
        MaxRetries:    3,
        InitialDelay:  time.Second,
        MaxDelay:      30 * time.Second,
        BackoffFactor: 2.0,
    }),
    
    // 日志配置
    queue.WithLogLevel("info"),
    queue.WithLogFile("./logs/queue.log"),
)
```

## 高级用法

### 流式任务

```go
// 提交流式任务
streamChan, err := qm.SubmitStreamingTask(task)
if err != nil {
    log.Fatal(err)
}

// 处理流式输出
for chunk := range streamChan {
    if chunk.Error != nil {
        fmt.Printf("流式错误: %v\n", chunk.Error)
        break
    }
    
    if chunk.Done {
        fmt.Println("\n流式完成")
        break
    }
    
    fmt.Print(chunk.Data)
}
```

### 批量处理

```go
// 提交多个任务
tasks := []*models.Task{
    queue.NewChatTask("llama2", []models.ChatMessage{...}),
    queue.NewGenerateTask("codellama", "编写一个函数"),
    queue.NewEmbedTask("nomic-embed-text", "示例文本"),
}

// 同步批量处理
results, err := qm.SubmitBatchTasks(tasks)
if err != nil {
    log.Fatal(err)
}

// 处理结果
for i, result := range results {
    fmt.Printf("任务 %d: 成功=%v, 数据=%v\n", i, result.Success, result.Data)
}
```

### 事件监控

```go
// 订阅任务事件
eventChan, err := qm.Subscribe([]string{"task_completed", "task_failed"})
if err != nil {
    log.Fatal(err)
}

// 监控事件
go func() {
    for event := range eventChan {
        fmt.Printf("事件: %s, 任务: %s, 状态: %s\n", 
            event.Type, event.TaskID, event.Status)
    }
}()
```

## API参考

### QueueManager接口

```go
type QueueManagerInterface interface {
    // 生命周期管理
    Start(ctx context.Context) error
    Stop() error
    Close() error

    // 任务提交
    SubmitTask(task *models.Task) (string, error)
    SubmitTaskWithCallback(task *models.Task, callback models.TaskCallback) (string, error)
    SubmitTaskWithChannel(task *models.Task, resultChan chan *models.TaskResult) (string, error)
    SubmitStreamingTask(task *models.Task) (<-chan *models.StreamChunk, error)

    // 任务管理
    GetTask(taskID string) (*models.Task, error)
    GetTaskStatus(taskID string) (models.TaskStatus, error)
    CancelTask(taskID string) error
    UpdateTaskPriority(taskID string, priority models.Priority) error

    // 查询操作
    ListTasks(filter models.TaskFilter) ([]*models.Task, error)
    GetQueueStats() (*models.QueueStats, error)

    // 事件监控
    Subscribe(eventTypes []string) (<-chan *models.TaskEvent, error)
    Unsubscribe(eventChan <-chan *models.TaskEvent) error
}
```

## CLI命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `serve` | 启动带Web界面的队列服务器 | `ollama-queue serve --port 7125` |
| `submit` | 向服务器提交新任务 | `ollama-queue submit chat --model llama2 --messages "user:你好"` |
| `list` | 列出任务（支持过滤） | `ollama-queue list --status running --limit 10` |
| `status` | 显示任务状态或队列统计 | `ollama-queue status <task-id>` |
| `cancel` | 取消一个或多个任务 | `ollama-queue cancel <task-id1> <task-id2>` |
| `priority` | 更新任务优先级 | `ollama-queue priority <task-id> high` |

## 系统架构

系统采用灵活的客户端-服务器架构，支持多种使用模式：

### 服务器模式
- 独立HTTP服务器，提供REST API
- 实时WebSocket通信
- 内置Web监控界面
- 基于BadgerDB的持久化任务存储

### 客户端集成
- HTTP客户端库，便于编程访问
- CLI工具，用于命令行操作
- 直接库集成，用于嵌入式使用

## 性能特点

- **吞吐量**: 每秒处理数千个任务
- **延迟**: 亚毫秒级任务调度
- **内存**: 高效内存使用，可配置限制
- **存储**: 持久化存储，自动清理
- **并发**: 可配置工作池，优化资源利用

## 错误处理

系统提供全面的错误处理：

- **任务失败**: 自动重试，支持指数退避
- **连接错误**: 优雅降级和恢复
- **存储错误**: 数据一致性和损坏保护
- **资源限制**: 队列大小和内存限制，支持背压

## 贡献

1. Fork此仓库
2. 创建功能分支
3. 进行更改
4. 为新功能添加测试
5. 运行测试套件: `go test ./...`
6. 提交Pull Request

## 测试

运行所有测试：
```bash
go test ./...
```

运行测试并查看覆盖率：
```bash
go test ./... -cover
```

运行特定包的测试：
```bash
go test ./pkg/scheduler -v
go test ./pkg/storage -v
go test ./internal/models -v
```

## 许可证

本项目采用MIT许可证 - 请参阅 [LICENSE](LICENSE) 文件了解详情。

## 支持

- 📖 文档: [docs/](docs/) | [English](README.md)
- 🐛 问题: [GitHub Issues](https://github.com/liliang-cn/ollama-queue/issues)
- 💬 讨论: [GitHub Discussions](https://github.com/liliang-cn/ollama-queue/discussions)

## 相关项目

- [Ollama](https://github.com/ollama/ollama) - 本地运行大型语言模型
- [Ollama-Go](https://github.com/liliang-cn/ollama-go) - Ollama的Go客户端