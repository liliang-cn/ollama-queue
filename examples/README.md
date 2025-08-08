# Usage Examples

This directory contains practical examples demonstrating how to use the Ollama Queue library.

## Examples

### 1. Basic Usage (`examples/basic/`)
Demonstrates fundamental operations:
- Creating and configuring a queue manager
- Submitting different task types (chat, generate, embed)
- Using callbacks and channels for results
- Checking queue statistics

Run with:
```bash
cd examples/basic
go run main.go
```

### 2. Streaming (`examples/streaming/`)
Shows how to work with streaming tasks:
- Real-time streaming for chat and generation tasks
- Handling streaming data and errors
- Managing multiple concurrent streams

Run with:
```bash
cd examples/streaming
go run main.go
```

### 3. Batch Processing (`examples/batch/`)
Demonstrates batch operations:
- Synchronous and asynchronous batch processing
- Priority-based task scheduling
- Event monitoring and queue statistics
- Mixed priority batch handling

Run with:
```bash
cd examples/batch
go run main.go
```

## Prerequisites

Before running the examples:

1. **Install Ollama**: Make sure Ollama is installed and running
   ```bash
   # Install Ollama (if not already installed)
   curl -fsSL https://ollama.ai/install.sh | sh
   
   # Start Ollama
   ollama serve
   ```

2. **Download Required Models**: The examples use these models:
   ```bash
   ollama pull llama2
   ollama pull codellama
   ollama pull nomic-embed-text
   ```

3. **Build the Project**: From the root directory:
   ```bash
   go mod tidy
   go build ./...
   ```

## Running the CLI Examples

### Submit Tasks via CLI

```bash
# Submit a chat task
./ollama-queue submit chat --model llama2 --messages "user:Hello, how are you?" --priority high

# Submit a generation task with streaming
./ollama-queue submit generate --model codellama --prompt "Write a Go function" --stream

# Submit an embedding task
./ollama-queue submit embed --model nomic-embed-text --input "Sample text to embed"
```

### Monitor Tasks

```bash
# List all tasks
./ollama-queue list

# List only running tasks
./ollama-queue list --status running

# Show queue statistics
./ollama-queue status

# Show specific task status
./ollama-queue status <task-id>
```

### Manage Tasks

```bash
# Cancel a task
./ollama-queue cancel <task-id>

# Update task priority
./ollama-queue priority <task-id> high
```

## Configuration

You can customize the queue behavior using configuration files or environment variables:

### Configuration File (`config.yaml`)
```yaml
ollama:
  host: "http://localhost:11434"
  timeout: "5m"

queue:
  max_workers: 4
  storage_path: "./data"
  cleanup_interval: "1h"

retry_config:
  max_retries: 3
  initial_delay: "1s"
  backoff_factor: 2.0
```

### Environment Variables
```bash
export OLLAMA_HOST="http://localhost:11434"
export QUEUE_MAX_WORKERS=4
export QUEUE_STORAGE_PATH="./data"
export RETRY_MAX_RETRIES=3
```

## Troubleshooting

### Common Issues

1. **"Connection refused" errors**: Ensure Ollama is running (`ollama serve`)

2. **"Model not found" errors**: Download required models:
   ```bash
   ollama pull llama2
   ollama pull codellama
   ollama pull nomic-embed-text
   ```

3. **Permission errors**: Ensure the data directory is writable:
   ```bash
   mkdir -p ./data
   chmod 755 ./data
   ```

4. **Port conflicts**: If port 11434 is in use, configure a different port:
   ```bash
   export OLLAMA_HOST="http://localhost:11435"
   ollama serve --port 11435
   ```

### Debug Mode

Enable debug logging for detailed information:
```bash
export LOG_LEVEL=debug
./ollama-queue status
```

Or in code:
```go
qm, err := queue.NewQueueManagerWithOptions(
    queue.WithLogLevel("debug"),
)
```