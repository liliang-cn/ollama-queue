# Cron Task Scheduling

This example demonstrates how to use the cron task scheduling functionality in ollama-queue, which allows you to create recurring tasks based on cron expressions.

## Overview

Cron tasks in ollama-queue allow you to:
- Schedule recurring AI model tasks using standard cron expressions
- Manage task lifecycles (enable/disable, update, remove)
- Track execution history and statistics
- Set task priorities and metadata
- Support all task types (chat, generate, embed)

## Cron Expression Format

Ollama-queue uses the standard 5-field cron format:

```
* * * * *
│ │ │ │ │
│ │ │ │ └── Day of Week (0-6, 0=Sunday)
│ │ │ └──── Month (1-12)
│ │ └────── Day of Month (1-31)
│ └──────── Hour (0-23)
└────────── Minute (0-59)
```

### Supported Syntax

- `*` - Any value
- `*/n` - Every n units (e.g., `*/15` = every 15 minutes)
- `n-m` - Range (e.g., `1-5` = Monday through Friday)
- `n,m,o` - List (e.g., `1,3,5` = Monday, Wednesday, Friday)

### Example Expressions

| Expression | Description |
|------------|-------------|
| `*/5 * * * *` | Every 5 minutes |
| `0 * * * *` | Every hour at minute 0 |
| `0 9 * * 1-5` | 9 AM Monday through Friday |
| `0 0 * * 0` | Midnight every Sunday |
| `0 12 1 * *` | Noon on the 1st of every month |
| `30 14 * * 6` | 2:30 PM every Saturday |

## Running the Example

1. Start the ollama-queue server:
```bash
ollama-queue serve
```

2. Run the cron example:
```bash
cd examples/cron
go run main.go
```

## CLI Commands

### Add a Cron Task

```bash
# Basic chat task every hour
ollama-queue cron add \
  --name "hourly-check" \
  --cron "0 * * * *" \
  --type chat \
  --model qwen3 \
  --prompt "Provide an hourly status update"

# Daily briefing on weekdays
ollama-queue cron add \
  --name "daily-briefing" \
  --cron "0 9 * * 1-5" \
  --type chat \
  --model qwen3 \
  --system "You are a professional assistant" \
  --prompt "Provide today's briefing" \
  --priority high

# Generate task with scheduling
ollama-queue cron add \
  --name "weekly-report" \
  --cron "0 17 * * 5" \
  --type generate \
  --model qwen3 \
  --prompt "Generate weekly performance report" \
  --priority critical \
  --metadata '{"department":"analytics","format":"markdown"}'
```

### List Cron Tasks

```bash
# List all cron tasks
ollama-queue cron list

# List only enabled tasks
ollama-queue cron list --enabled

# List with pagination
ollama-queue cron list --limit 10 --offset 0
```

### Manage Cron Tasks

```bash
# Show detailed task information
ollama-queue cron show [cron-id]

# Enable/disable tasks
ollama-queue cron enable [cron-id]
ollama-queue cron disable [cron-id]

# Update a cron task
ollama-queue cron update [cron-id] --cron "0 8 * * 1-5" --name "morning-update"

# Remove a cron task
ollama-queue cron remove [cron-id]
```

### Test Cron Expressions

```bash
# Test a cron expression and see next execution times
ollama-queue cron test "0 9 * * 1-5"
ollama-queue cron test "*/15 * * * *" --count 10
```

## Programmatic Usage

### Creating Cron Tasks

```go
package main

import (
    "github.com/liliang-cn/ollama-queue/internal/models"
    "github.com/liliang-cn/ollama-queue/pkg/client"
)

func main() {
    cli := client.New("localhost:8080")
    
    // Create a cron task
    cronTask := &models.CronTask{
        Name:     "Daily Summary",
        CronExpr: "0 18 * * 1-5", // 6 PM weekdays
        Enabled:  true,
        TaskTemplate: models.TaskTemplate{
            Type:     models.TaskTypeChat,
            Model:    "qwen3",
            Priority: models.PriorityHigh,
            Payload: []models.ChatMessage{
                {
                    Role:    "user",
                    Content: "Provide end-of-day summary",
                },
            },
        },
        Metadata: map[string]interface{}{
            "team": "operations",
        },
    }
    
    cronID, err := cli.AddCronTask(cronTask)
    if err != nil {
        panic(err)
    }
    
    println("Created cron task:", cronID)
}
```

### Managing Cron Tasks

```go
// List cron tasks
enabled := true
filter := models.CronFilter{
    Enabled: &enabled, // Only enabled tasks
    Limit:   10,
}
tasks, err := cli.ListCronTasks(filter)

// Get specific cron task
task, err := cli.GetCronTask(cronID)

// Update cron task
task.CronExpr = "0 19 * * 1-5" // Change to 7 PM
err = cli.UpdateCronTask(task)

// Enable/disable
err = cli.EnableCronTask(cronID)
err = cli.DisableCronTask(cronID)

// Remove
err = cli.RemoveCronTask(cronID)
```

## Task Types and Payloads

### Chat Tasks

```go
cronTask := &models.CronTask{
    Name:     "Daily Chat Task",
    CronExpr: "0 9 * * *",
    Enabled:  true,
    TaskTemplate: models.TaskTemplate{
        Type:     models.TaskTypeChat,
        Model:    "qwen3",
        Priority: models.PriorityNormal,
        Payload: []models.ChatMessage{
            {
                Role:    "system",
                Content: "You are a helpful assistant",
            },
            {
                Role:    "user",
                Content: "Provide daily insights",
            },
        },
        Options: map[string]interface{}{
            "stream": false,
        },
    },
}
```

### Generate Tasks

```go
cronTask := &models.CronTask{
    Name:     "Report Generator",
    CronExpr: "0 18 * * 5",
    Enabled:  true,
    TaskTemplate: models.TaskTemplate{
        Type:     models.TaskTypeGenerate,
        Model:    "qwen3",
        Priority: models.PriorityNormal,
        Payload: map[string]interface{}{
            "prompt": "Generate a status report",
            "system": "You are a report generator",
        },
        Options: map[string]interface{}{
            "stream": false,
        },
    },
}
```

### Embed Tasks

```go
cronTask := &models.CronTask{
    Name:     "Content Embedder", 
    CronExpr: "0 */6 * * *",
    Enabled:  true,
    TaskTemplate: models.TaskTemplate{
        Type:     models.TaskTypeEmbed,
        Model:    "qwen3",
        Priority: models.PriorityNormal,
        Payload: map[string]interface{}{
            "input": "Text to embed",
        },
    },
}
```

## Best Practices

### 1. Appropriate Frequencies
- Avoid overly frequent tasks (< 1 minute intervals) unless necessary
- Consider server load and model response times
- Use higher priorities for critical recurring tasks

### 2. Error Handling
- Cron tasks inherit the same retry logic as regular tasks
- Failed cron executions will be retried according to task configuration
- Monitor task failure rates for recurring issues

### 3. Resource Management
- Disable unused cron tasks rather than deleting them
- Use metadata to organize and categorize cron tasks
- Set appropriate priorities to avoid resource conflicts

### 4. Monitoring
- Regularly review cron task execution statistics
- Monitor `RunCount` and `LastRun` timestamps
- Set up alerting for failed recurring tasks

### 5. Updates and Maintenance
- Update cron expressions during low-usage periods
- Test new cron expressions before deploying to production
- Keep task templates simple and focused

## Common Use Cases

### 1. Automated Reporting
```bash
# Daily sales report at 8 AM
ollama-queue cron add --name "daily-sales" --cron "0 8 * * *" \
  --type generate --prompt "Generate daily sales summary"

# Weekly performance report on Fridays
ollama-queue cron add --name "weekly-performance" --cron "0 17 * * 5" \
  --type chat --prompt "Create weekly performance analysis"
```

### 2. Health Monitoring
```bash
# System health check every 30 minutes
ollama-queue cron add --name "health-check" --cron "*/30 * * * *" \
  --type generate --prompt "Check system health status"

# Database maintenance reminder weekly
ollama-queue cron add --name "db-maintenance" --cron "0 2 * * 0" \
  --type chat --prompt "Generate database maintenance checklist"
```

### 3. Content Generation
```bash
# Morning newsletter content
ollama-queue cron add --name "newsletter" --cron "0 6 * * 1-5" \
  --type chat --prompt "Generate morning newsletter content"

# Social media content ideas
ollama-queue cron add --name "social-content" --cron "0 10 * * *" \
  --type generate --prompt "Generate social media content ideas"
```

### 4. Data Processing
```bash
# Process daily logs
ollama-queue cron add --name "log-analysis" --cron "0 23 * * *" \
  --type generate --prompt "Analyze today's application logs"

# Generate embeddings for new content
ollama-queue cron add --name "content-embedding" --cron "0 */6 * * *" \
  --type embed --input "New content for embedding"
```

## Troubleshooting

### Invalid Cron Expression
```
Error: invalid cron expression: cron expression must have 5 fields, got 4
```
Solution: Ensure your cron expression has exactly 5 fields (minute, hour, day, month, day-of-week).

### Task Not Executing
1. Check if the cron task is enabled: `ollama-queue cron show [id]`
2. Verify the cron expression: `ollama-queue cron test "[expression]"`
3. Check server logs for execution errors
4. Ensure the target model is available

### Performance Issues
1. Review the frequency of your cron tasks
2. Check if multiple tasks are competing for resources
3. Adjust task priorities appropriately
4. Monitor server resource usage

## Architecture Notes

- Cron scheduling runs independently of the main task queue
- Cron tasks create regular tasks when triggered
- The scheduler checks for due tasks every minute
- Cron task state is persisted in the same database as regular tasks
- Task execution follows the same priority and retry logic as manual submissions