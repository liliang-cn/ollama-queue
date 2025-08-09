# Remote Scheduling with OpenAI-Compatible APIs

The ollama-queue system now supports remote scheduling, allowing you to distribute tasks to OpenAI-compatible API endpoints while maintaining local Ollama as a fallback option.

## Features

- **Multiple Endpoint Support**: Configure multiple OpenAI-compatible endpoints with different priorities
- **Automatic Failover**: Automatically fallback to lower-priority endpoints or local Ollama if primary endpoints fail
- **Health Monitoring**: Periodic health checks ensure only available endpoints are used
- **Priority-Based Routing**: High-priority tasks can be automatically routed to remote endpoints
- **OpenAI Compatibility**: Works with any OpenAI-compatible API (OpenAI, DeepSeek, LocalAI, vLLM, etc.)

## Configuration

### Basic Setup

Enable remote scheduling in your configuration:

```yaml
remote_scheduling:
  enabled: true
  health_check_interval: "30s"
  fallback_to_local: true
  endpoints:
    - name: "OpenAI"
      base_url: "https://api.openai.com/v1"
      api_key: "sk-your-api-key"
      priority: 10
      enabled: true
```

### Environment Variables

Store API keys securely using environment variables:

```bash
export OPENAI_API_KEY="sk-your-openai-api-key"
export DEEPSEEK_API_KEY="sk-your-deepseek-api-key"
```

Then reference them in your config:

```yaml
api_key: "${OPENAI_API_KEY}"
```

## Task Routing

Tasks are routed to remote endpoints based on:

1. **Explicit Remote Flag**: Tasks with `RemoteExecution: true`
2. **Priority Level**: High and Critical priority tasks are candidates for remote execution
3. **Endpoint Availability**: Only healthy endpoints are considered

### Example Task Submission

```go
task := &models.Task{
    Type:     models.TaskTypeChat,
    Priority: models.PriorityHigh,
    Model:    "gpt-4",
    Payload: models.ChatTaskPayload{
        Messages: []models.ChatMessage{
            {Role: "user", Content: "Hello, world!"},
        },
    },
    RemoteExecution: true, // Explicitly request remote execution
}
```

## Supported Endpoints

Any OpenAI-compatible API can be used:

- **OpenAI**: GPT-3.5, GPT-4, Embeddings
- **DeepSeek**: DeepSeek models via OpenAI-compatible API
- **LocalAI**: Self-hosted OpenAI-compatible server
- **vLLM**: High-performance inference server
- **LM Studio**: Local model server with OpenAI compatibility
- **Ollama Web API**: Ollama instances exposed via OpenAI-compatible wrapper

## Health Monitoring

The system performs periodic health checks on all configured endpoints:

- Endpoints are marked as available/unavailable based on health check results
- Failed endpoints are automatically retried after the health check interval
- The best available endpoint (highest priority) is always selected

## Fallback Behavior

When `fallback_to_local: true`:

1. System first attempts remote endpoints by priority
2. If all remote endpoints fail, task is queued locally
3. Local Ollama executor handles the task

## API Compatibility

The remote scheduler supports:

- **Chat Completions**: `/v1/chat/completions`
- **Text Completions**: `/v1/completions`
- **Embeddings**: `/v1/embeddings`

Both streaming and non-streaming modes are supported.

## Example Use Cases

### 1. Cost Optimization
Route simple tasks to cheaper endpoints while sending complex tasks to more capable models:

```yaml
endpoints:
  - name: "GPT-4"
    priority: 10
    # For complex tasks
  - name: "GPT-3.5"
    priority: 5
    # For simple tasks
```

### 2. Load Balancing
Distribute load across multiple endpoints:

```yaml
endpoints:
  - name: "Primary"
    priority: 10
  - name: "Secondary"
    priority: 9
  - name: "Tertiary"
    priority: 8
```

### 3. Hybrid Cloud/Local
Use cloud APIs with local fallback:

```yaml
endpoints:
  - name: "CloudAPI"
    base_url: "https://api.example.com/v1"
    priority: 10
fallback_to_local: true  # Use local Ollama as backup
```

## Monitoring

Monitor remote scheduling through:

- Queue statistics showing remote vs local task distribution
- Health check logs showing endpoint availability
- Task results indicating which endpoint processed each task

## Troubleshooting

### Common Issues

1. **Authentication Errors**: Verify API keys are correct
2. **Network Issues**: Check firewall and proxy settings
3. **Model Not Found**: Ensure the model name is supported by the endpoint
4. **Rate Limiting**: Configure multiple endpoints to distribute load

### Debug Logging

Enable debug logging to see detailed remote scheduling information:

```yaml
log_level: "debug"
```

## Security Considerations

- Store API keys in environment variables or secure vaults
- Use HTTPS for all remote endpoints
- Implement rate limiting to prevent abuse
- Monitor API usage and costs