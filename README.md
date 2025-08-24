# Ollama Queue

一个**高性能通用任务队列系统**，支持多种执行器插件，包括Ollama模型、OpenAI API、脚本执行等。

**📖 [中文文档](README_zh.md)** | **🌟 [English](README.md)**

## ✨ 特性

- 🚀 **通用架构**: 支持多种执行器类型 (Ollama, OpenAI, Script等)
- 🏗️ **客户端-服务器分离**: 独立的server和client二进制文件
- 🌐 **实时Web UI**: 浏览器管理界面
- ⏰ **Cron调度**: Unix风格定时任务
- 📋 **优先级调度**: 四级优先级 (1, 5, 10, 15)
- 💾 **持久化存储**: BadgerDB崩溃恢复
- 📊 **实时监控**: 任务状态追踪
- 📚 **通用架构**: 可作为Go库使用

## 🚀 快速开始

### 安装
```bash
go get github.com/liliang-cn/ollama-queue
```

### 服务器模式
```bash
# 启动服务器 (默认端口7125)
ollama-queue-server

# 访问Web界面
open http://localhost:7125
```

### 客户端使用
```bash
# 提交Ollama任务
ollama-queue submit chat --model qwen3 --messages "user:Hello!"

# 提交脚本任务 (新功能)
ollama-queue submit generic --executor script --action execute --command "python script.py"

# 定时任务 (支持所有执行器)
ollama-queue cron add --name "Daily Report" --schedule "0 9 * * 1-5" --executor ollama --action generate --model qwen3 --prompt "Generate daily business report"

# 脚本定时任务
ollama-queue cron add --name "Backup" --schedule "0 2 * * *" --executor script --action execute --command "python backup.py"

# OpenAI定时任务  
ollama-queue cron add --name "Analysis" --schedule "0 */6 * * *" --executor openai --action chat --model gpt-4 --prompt "Analyze system metrics"

# 查看队列状态
ollama-queue status
```

## 📚 作为库使用

### HTTP客户端模式 (推荐)
```go
import (
    "github.com/liliang-cn/ollama-queue/pkg/client"
    "github.com/liliang-cn/ollama-queue/pkg/models"
)

// 连接到运行中的服务器
cli := client.New("localhost:7125")

// 提交任务到Ollama
task := models.CreateOllamaTask(
    models.ActionChat,
    "qwen3", 
    map[string]interface{}{
        "messages": []models.ChatMessage{
            {Role: "user", Content: "Hello!"},
        },
    },
)
taskID, _ := cli.SubmitGenericTask(task)

// 提交脚本执行任务
scriptTask := models.CreateGenericTask(
    models.ExecutorTypeScript,
    models.ActionExecute,
    map[string]interface{}{
        "command": "python analyze.py",
    },
)
cli.SubmitGenericTask(scriptTask)
```

### 嵌入式使用
```go
import "github.com/liliang-cn/ollama-queue/pkg/executor"

// 创建执行器注册表
registry := executor.NewExecutorRegistry()

// 注册自定义执行器
scriptPlugin, _ := executor.NewScriptPlugin(executor.ScriptConfig{
    WorkingDir: "./scripts",
})
registry.Register(models.ExecutorTypeScript, scriptPlugin)

// 提交和执行任务
task := models.CreateGenericTask(models.ExecutorTypeScript, models.ActionExecute, payload)
result, _ := registry.ExecuteTask(ctx, task)
```

## 🎯 支持的执行器

| 执行器 | 动作 | 描述 |
|-------|------|------|
| `ollama` | `chat`, `generate`, `embed` | 本地Ollama模型 |
| `openai` | `chat`, `generate` | OpenAI API兼容服务 |
| `script` | `execute` | 执行脚本和命令 |

## 📝 示例

查看 `examples/` 目录获取完整示例：
- `examples/universal_queue/` - 通用队列库使用示例
- `examples/basic/` - 基本任务提交
- `examples/cron/` - 定时任务示例

## 🧪 测试

```bash
go test ./...
go run examples/universal_queue/main.go
```

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**ollama-queue**: 通用任务调度平台，支持多种执行器插件 🚀