# Client-Server Architecture

The ollama-queue project now uses a separated client-server architecture for better deployment flexibility and clearer separation of concerns.

## Architecture Overview

```
┌─────────────────────┐    HTTP API    ┌─────────────────────┐
│   ollama-queue      │ ────────────── │ ollama-queue-server │
│   (Client CLI)      │                │   (Server Daemon)   │
└─────────────────────┘                └─────────────────────┘
         │                                       │
         │                                       │
         ▼                                       ▼
   ┌──────────────┐                     ┌──────────────┐
   │   Commands   │                     │   Services   │
   │              │                     │              │
   │ • submit     │                     │ • Queue Mgmt │
   │ • cancel     │                     │ • Cron Tasks │
   │ • status     │                     │ • Web UI     │
   │ • list       │                     │ • WebSocket  │
   │ • priority   │                     │ • HTTP API   │
   │ • cron       │                     │ • Storage    │
   └──────────────┘                     └──────────────┘
```

## Components

### 🖥️ Server (`ollama-queue-server`)

**Purpose**: Long-running daemon that manages the task queue and provides HTTP API.

**Features**:
- Task queue management
- Cron-based recurring tasks
- Web UI for monitoring
- WebSocket real-time updates
- HTTP REST API
- Persistent storage (BadgerDB)
- Remote task scheduling

**Usage**:
```bash
# Start the server
./ollama-queue-server --port 8080 --host localhost

# With custom configuration
./ollama-queue-server --data-dir ./mydata --ollama-host http://remote:11434
```

**Deployment**:
- Can run as a system service
- Suitable for Docker containers
- Persistent data storage
- Production-ready with graceful shutdown

### 💻 Client (`ollama-queue`)

**Purpose**: Command-line interface for interacting with the server.

**Features**:
- Task submission and management
- Cron task scheduling
- Queue monitoring and statistics
- Lightweight and portable
- No persistent storage needed

**Usage**:
```bash
# Submit a task
./ollama-queue submit --type chat --model qwen3 --prompt "Hello"

# Manage cron tasks
./ollama-queue cron add --name "hourly" --cron "0 * * * *" --type chat --prompt "Status update"

# Monitor queue
./ollama-queue status
./ollama-queue list
```

## Benefits of Separation

### 🎯 **Clear Responsibilities**
- **Server**: Focus on queue management, persistence, and API
- **Client**: Focus on user interaction and command-line experience

### 🚀 **Better Deployment**
- **Server**: Deploy once, run continuously as a service
- **Client**: Install on developer machines, CI/CD systems, etc.

### 📦 **Smaller Binaries**
- **Server**: ~15MB (includes web UI, storage, full queue management)
- **Client**: ~8MB (lightweight, API client only)

### 🔒 **Security Separation**
- **Server**: Requires persistent file system access, network ports
- **Client**: Minimal permissions, just HTTP client capabilities

### 🏗️ **Development Benefits**
- Independent testing of server and client
- Different release cycles if needed
- Clear API contract between components

## Migration from Combined Binary

If you were using the old combined binary:

### Old Way:
```bash
# Start server (blocking)
ollama-queue serve --port 8080

# In another terminal, submit tasks
ollama-queue submit --type chat --prompt "Hello"
```

### New Way:
```bash
# Start server (background/service)
ollama-queue-server --port 8080 &

# Submit tasks from same or different machine
ollama-queue submit --type chat --prompt "Hello"
```

## Configuration

Both binaries share the same configuration system but with different defaults:

### Server Configuration:
```yaml
# ~/.ollama-queue/config.yaml
server:
  host: "0.0.0.0"      # Listen on all interfaces
  port: "8080"
  
storage:
  path: "./data"
  
ollama:
  host: "http://localhost:11434"
```

### Client Configuration:
```yaml
# ~/.ollama-queue/config.yaml  
client:
  server_url: "http://localhost:8080"  # Auto-detected from config
```

## Production Deployment

### Server as System Service:
```bash
# Install server binary
sudo cp ollama-queue-server /usr/local/bin/

# Create systemd service
sudo tee /etc/systemd/system/ollama-queue.service > /dev/null <<EOF
[Unit]
Description=Ollama Queue Server
After=network.target

[Service]
Type=simple
User=ollama-queue
WorkingDirectory=/var/lib/ollama-queue
ExecStart=/usr/local/bin/ollama-queue-server --data-dir /var/lib/ollama-queue
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable ollama-queue
sudo systemctl start ollama-queue
```

### Client Installation:
```bash
# Install client on dev machines
cp ollama-queue /usr/local/bin/

# Or use from CI/CD
./ollama-queue submit --type generate --prompt "Build report"
```

## Docker Deployment

### Server:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ollama-queue-server ./cmd/ollama-queue-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ollama-queue-server .
EXPOSE 8080
CMD ["./ollama-queue-server", "--host", "0.0.0.0"]
```

```bash
docker run -d -p 8080:8080 -v ollama-data:/data ollama-queue-server
```

### Client (for CI/CD):
```dockerfile
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY ollama-queue /usr/local/bin/
ENTRYPOINT ["ollama-queue"]
```

This architecture provides much better flexibility for different deployment scenarios while maintaining all the existing functionality!