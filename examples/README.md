# Ollama Queue Usage Examples

This directory contains practical examples demonstrating how to use the Ollama Queue system in both client-server and embedded library modes.

## Examples

### 1. Basic Library Usage (`examples/basic/`)
Demonstrates fundamental embedded library operations:
- Creating and configuring a queue manager
- Submitting different task types (chat, generate, embed)
- Using callbacks and channels for results
- Checking queue statistics

Run with:
```bash
cd examples/basic
go run main.go
```

### 2. Client-Server Usage (`examples/client_server/`)
Shows how to use the client-server architecture:
- Connecting to a running queue server
- Submitting tasks via HTTP client
- Monitoring tasks and queue status
- Real-time web interface interaction

**Prerequisites**: Start the server first:
```bash
# Terminal 1: Start the server
ollama-queue serve

# Terminal 2: Run the client example
cd examples/client_server
go run main.go
```

### 3. Streaming (`examples/streaming/`)
Shows how to work with streaming tasks:
- Real-time streaming for chat and generation tasks
- Handling streaming data and errors
- Managing multiple concurrent streams

Run with:
```bash
cd examples/streaming
go run main.go
```

### 4. Batch Processing (`examples/batch/`)
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

### 5. Library Integration (`examples/library_integration/`)
Advanced embedded usage examples:
- Complex task workflows
- Event monitoring and callbacks
- Production-ready configurations
- Error handling and recovery

Run with:
```bash
cd examples/library_integration  
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

## Usage Modes

### Client-Server Mode (Recommended)

The client-server mode provides better scalability and allows multiple clients to share a single queue server.

1. **Start the server**:
   ```bash
   # Start server with default settings
   ollama-queue serve
   
   # Or with custom port and data directory
   ollama-queue serve --port 9090 --data-dir ./my-queue-data
   ```

2. **Use CLI client**:
   ```bash
   # Submit tasks
   ollama-queue submit chat --model llama2 --messages "user:Hello!"
   ollama-queue submit generate --model codellama --prompt "Write a function"
   
   # Monitor tasks
   ollama-queue list
   ollama-queue status
   
   # Manage tasks
   ollama-queue cancel <task-id>
   ollama-queue priority <task-id> high
   ```

3. **Use HTTP client in your applications**:
   ```go
   import "github.com/liliang-cn/ollama-queue/pkg/client"
   
   cli := client.New("localhost:7125")
   taskID, err := cli.SubmitTask(task)
   ```

4. **Access web interface**: Visit `http://localhost:7125` for real-time monitoring

### Embedded Library Mode

For applications that need direct integration without a separate server:

```go
import "github.com/liliang-cn/ollama-queue/pkg/queue"

qm, err := queue.NewQueueManagerWithOptions(...)
defer qm.Close()
qm.Start(ctx)
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