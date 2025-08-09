package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
	"github.com/spf13/cobra"
)

var (
	taskModel    string
	taskPrompt   string
	taskPriority string
	taskSystem   string
	taskStream   bool
	taskMessages []string
	taskInput    string
	taskOutput   string
	batchFile    string
	batchSync    bool
)

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:   "submit [chat|generate|embed|batch]",
	Short: "Submit a new task to the queue",
	Long: `Submit a new task to the Ollama queue. You can submit chat, generate, embed, or batch tasks.

Examples:
  # Submit a chat task
  ollama-queue submit chat --model qwen3 --messages "user:Hello, how are you?"

  # Submit a generation task
  ollama-queue submit generate --model qwen3 --prompt "Write a Go function"

  # Submit an embedding task
  ollama-queue submit embed --model nomic-embed-text --input "Sample text"

  # Submit a batch of tasks from a file
  ollama-queue submit batch --file tasks.json --sync`,
	Args: cobra.ExactArgs(1),
	RunE: runSubmit,
}

// submitChatCmd represents the submit chat command
var submitChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Submit a chat task",
	RunE:  runSubmitChat,
}

// submitGenerateCmd represents the submit generate command
var submitGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Submit a generation task",
	RunE:  runSubmitGenerate,
}

// submitBatchCmd represents the submit batch command
var submitBatchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Submit a batch of tasks from a file",
	Long: `Submit multiple tasks from a JSON file. Each task in the file should specify its type (chat, generate, or embed).

Example batch file format:
[
  {
    "type": "chat",
    "model": "qwen3",
    "priority": "high",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "system": "You are a helpful assistant",
    "scheduled_at": "2024-12-25T10:00:00Z"
  },
  {
    "type": "generate",
    "model": "qwen3",
    "priority": 10,
    "prompt": "Write a Go function to reverse a string",
    "system": "You are a coding assistant"
  },
  {
    "type": "embed",
    "model": "nomic-embed-text",
    "priority": 1,
    "input": "Sample text for embedding"
  }
]

Priority values can be specified as:
- String: "low", "normal", "high", "critical"  
- Number: 1 (low), 5 (normal), 10 (high), 15 (critical)

Scheduling:
- Use "scheduled_at" field with RFC3339 format (e.g., "2024-12-25T10:00:00Z")
- Tasks will be queued but not executed until the scheduled time

Examples:
  # Submit batch synchronously (wait for all to complete)
  ollama-queue submit batch --file tasks.json --sync

  # Submit batch asynchronously (return immediately)
  ollama-queue submit batch --file tasks.json`,
	RunE: runSubmitBatch,
}

// submitEmbedCmd represents the submit embed command
var submitEmbedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Submit an embedding task",
	RunE:  runSubmitEmbed,
}

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.AddCommand(submitChatCmd)
	submitCmd.AddCommand(submitGenerateCmd)
	submitCmd.AddCommand(submitEmbedCmd)
	submitCmd.AddCommand(submitBatchCmd)

	// Common flags
	submitCmd.PersistentFlags().StringVar(&taskModel, "model", "", "Model to use (required)")
	submitCmd.PersistentFlags().StringVar(&taskPriority, "priority", "normal", "Task priority (low, normal, high, critical)")
	submitCmd.PersistentFlags().StringVar(&taskSystem, "system", "", "System message")
	submitCmd.PersistentFlags().BoolVar(&taskStream, "stream", false, "Enable streaming output")
	submitCmd.PersistentFlags().StringVar(&taskOutput, "output", "json", "Output format (json, text)")

	// Chat-specific flags
	submitChatCmd.Flags().StringSliceVar(&taskMessages, "messages", nil, "Chat messages in format 'role:content'")

	// Generate-specific flags
	submitGenerateCmd.Flags().StringVar(&taskPrompt, "prompt", "", "Generation prompt (required)")

	// Embed-specific flags
	submitEmbedCmd.Flags().StringVar(&taskInput, "input", "", "Input text for embedding (required)")

	// Batch-specific flags
	submitBatchCmd.Flags().StringVar(&batchFile, "file", "", "JSON file containing batch tasks (required)")
	submitBatchCmd.Flags().BoolVar(&batchSync, "sync", false, "Wait for all tasks to complete (synchronous mode)")

	// Mark required flags
	submitChatCmd.MarkFlagRequired("model")
	submitChatCmd.MarkFlagRequired("messages")
	submitGenerateCmd.MarkFlagRequired("model")
	submitGenerateCmd.MarkFlagRequired("prompt")
	submitEmbedCmd.MarkFlagRequired("model")
	submitEmbedCmd.MarkFlagRequired("input")
	submitBatchCmd.MarkFlagRequired("file")
}

func runSubmit(cmd *cobra.Command, args []string) error {
	taskType := args[0]
	
	switch taskType {
	case "chat":
		return runSubmitChat(cmd, args)
	case "generate":
		return runSubmitGenerate(cmd, args)
	case "embed":
		return runSubmitEmbed(cmd, args)
	case "batch":
		return runSubmitBatch(cmd, args)
	default:
		return fmt.Errorf("unsupported task type: %s", taskType)
	}
}

func runSubmitChat(cmd *cobra.Command, args []string) error {
	if len(taskMessages) == 0 {
		return fmt.Errorf("at least one message is required")
	}

	messages, err := parseMessages(taskMessages)
	if err != nil {
		return fmt.Errorf("failed to parse messages: %w", err)
	}

	priority, err := parsePriority(taskPriority)
	if err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

	taskOptions := []queue.TaskOption{
		queue.WithTaskPriority(priority),
	}
	
	if taskSystem != "" {
		taskOptions = append(taskOptions, queue.WithChatSystem(taskSystem))
	}
	
	if taskStream {
		taskOptions = append(taskOptions, queue.WithChatStreaming(true))
	}

	task := queue.NewChatTask(taskModel, messages, taskOptions...)

	return submitTask(cmd, task)
}

func runSubmitGenerate(cmd *cobra.Command, args []string) error {
	priority, err := parsePriority(taskPriority)
	if err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

	taskOptions := []queue.TaskOption{
		queue.WithTaskPriority(priority),
	}
	
	if taskSystem != "" {
		taskOptions = append(taskOptions, queue.WithGenerateSystem(taskSystem))
	}
	
	if taskStream {
		taskOptions = append(taskOptions, queue.WithGenerateStreaming(true))
	}

	task := queue.NewGenerateTask(taskModel, taskPrompt, taskOptions...)

	return submitTask(cmd, task)
}

func runSubmitEmbed(cmd *cobra.Command, args []string) error {
	priority, err := parsePriority(taskPriority)
	if err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

	task := queue.NewEmbedTask(taskModel, taskInput, queue.WithTaskPriority(priority))

	return submitTask(cmd, task)
}

func submitTask(cmd *cobra.Command, task *models.Task) error {
	// Load configuration
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create a new client
	cli := client.New(cfg.ListenAddr)

	// Submit task
	taskID, err := cli.SubmitTask(task)
	if err != nil {
		return fmt.Errorf("failed to submit task: %w", err)
	}

	// Output result
	if taskOutput == "json" {
		result := map[string]interface{}{
			"task_id": taskID,
			"status":  "submitted",
			"type":    task.Type,
			"model":   task.Model,
		}
		
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Task submitted successfully\n")
		fmt.Printf("Task ID: %s\n", taskID)
		fmt.Printf("Type: %s\n", task.Type)
		fmt.Printf("Model: %s\n", task.Model)
		fmt.Printf("Status: submitted\n")
	}

	return nil
}

func submitStreamingTask(ctx context.Context, qm *queue.QueueManager, task *models.Task) error {
	streamChan, err := qm.SubmitStreamingTask(task)
	if err != nil {
		return fmt.Errorf("failed to submit streaming task: %w", err)
	}

	if taskOutput == "json" {
		fmt.Printf("{\"task_id\":\"%s\",\"streaming\":true,\"chunks\":[\n", task.ID)
	} else {
		fmt.Printf("Streaming task %s:\n", task.ID)
	}

	first := true
	for chunk := range streamChan {
		if chunk.Error != nil {
			if taskOutput == "json" {
				fmt.Printf("%s{\"error\":\"%s\"}", 
					map[bool]string{true: "", false: ","}[first], chunk.Error.Error())
			} else {
				fmt.Printf("Error: %v\n", chunk.Error)
			}
			break
		}

		if chunk.Done {
			if taskOutput == "json" {
				fmt.Printf("%s{\"done\":true}", 
					map[bool]string{true: "", false: ","}[first])
			} else {
				fmt.Println("\nStreaming completed")
			}
			break
		}

		if chunk.Data != "" {
			if taskOutput == "json" {
				data, _ := json.Marshal(chunk.Data)
				fmt.Printf("%s{\"data\":%s}", 
					map[bool]string{true: "", false: ","}[first], string(data))
			} else {
				fmt.Print(chunk.Data)
			}
			first = false
		}
	}

	if taskOutput == "json" {
		fmt.Println("\n]}")
	}

	return nil
}

func parseMessages(messages []string) ([]models.ChatMessage, error) {
	var result []models.ChatMessage

	for _, msg := range messages {
		parts := strings.SplitN(msg, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid message format: %s (expected 'role:content')", msg)
		}

		result = append(result, models.ChatMessage{
			Role:    parts[0],
			Content: parts[1],
		})
	}

	return result, nil
}

// BatchTaskSpec represents a task specification in a batch file
type BatchTaskSpec struct {
	Type        string                 `json:"type"`
	Model       string                 `json:"model"`
	Priority    interface{}            `json:"priority,omitempty"`  // Can be string or int
	System      string                 `json:"system,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	ScheduledAt string                 `json:"scheduled_at,omitempty"`  // RFC3339 format
	Messages    []models.ChatMessage   `json:"messages,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	Input       interface{}            `json:"input,omitempty"`
}

func runSubmitBatch(cmd *cobra.Command, args []string) error {
	// Read the batch file
	file, err := os.Open(batchFile)
	if err != nil {
		return fmt.Errorf("failed to open batch file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read batch file: %w", err)
	}

	var batchSpecs []BatchTaskSpec
	if err := json.Unmarshal(data, &batchSpecs); err != nil {
		return fmt.Errorf("failed to parse batch file: %w", err)
	}

	if len(batchSpecs) == 0 {
		return fmt.Errorf("batch file contains no tasks")
	}

	// Convert specs to tasks
	tasks := make([]*models.Task, 0, len(batchSpecs))
	for i, spec := range batchSpecs {
		task, err := createTaskFromSpec(spec)
		if err != nil {
			return fmt.Errorf("invalid task at index %d: %w", i, err)
		}
		tasks = append(tasks, task)
	}

	// Load configuration
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create a new client
	cli := client.New(cfg.ListenAddr)

	if batchSync {
		return submitBatchSync(cmd, cli, tasks)
	} else {
		return submitBatchAsync(cmd, cli, tasks)
	}
}

func createTaskFromSpec(spec BatchTaskSpec) (*models.Task, error) {
	// Parse priority
	priority := models.PriorityNormal
	if spec.Priority != nil {
		var priorityStr string
		switch v := spec.Priority.(type) {
		case string:
			priorityStr = v
		case int:
			priorityStr = fmt.Sprintf("%d", v)
		case float64: // JSON numbers are parsed as float64
			priorityStr = fmt.Sprintf("%.0f", v)
		default:
			return nil, fmt.Errorf("invalid priority type: %T (must be string or number)", spec.Priority)
		}
		
		if priorityStr != "" {
			p, err := parsePriority(priorityStr)
			if err != nil {
				return nil, fmt.Errorf("invalid priority: %w", err)
			}
			priority = p
		}
	}

	// Parse scheduled time if provided
	var scheduledAt *time.Time
	if spec.ScheduledAt != "" {
		parsed, err := time.Parse(time.RFC3339, spec.ScheduledAt)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_at format: %w (expected RFC3339 format like '2024-12-25T10:00:00Z')", err)
		}
		scheduledAt = &parsed
	}

	var task *models.Task
	var taskOptions []queue.TaskOption

	// Add common options
	taskOptions = append(taskOptions, queue.WithTaskPriority(priority))
	
	if scheduledAt != nil {
		taskOptions = append(taskOptions, queue.WithTaskScheduledAt(*scheduledAt))
	}
	
	if spec.System != "" {
		switch spec.Type {
		case "chat":
			taskOptions = append(taskOptions, queue.WithChatSystem(spec.System))
		case "generate":
			taskOptions = append(taskOptions, queue.WithGenerateSystem(spec.System))
		}
	}

	if spec.Stream {
		switch spec.Type {
		case "chat":
			taskOptions = append(taskOptions, queue.WithChatStreaming(true))
		case "generate":
			taskOptions = append(taskOptions, queue.WithGenerateStreaming(true))
		}
	}

	// Create task based on type
	switch spec.Type {
	case "chat":
		if len(spec.Messages) == 0 {
			return nil, fmt.Errorf("chat task requires messages")
		}
		task = queue.NewChatTask(spec.Model, spec.Messages, taskOptions...)
		
	case "generate":
		if spec.Prompt == "" {
			return nil, fmt.Errorf("generate task requires prompt")
		}
		task = queue.NewGenerateTask(spec.Model, spec.Prompt, taskOptions...)
		
	case "embed":
		if spec.Input == nil {
			return nil, fmt.Errorf("embed task requires input")
		}
		task = queue.NewEmbedTask(spec.Model, spec.Input, taskOptions...)
		
	default:
		return nil, fmt.Errorf("unsupported task type: %s", spec.Type)
	}

	return task, nil
}

func submitBatchSync(cmd *cobra.Command, cli *client.Client, tasks []*models.Task) error {
	fmt.Printf("Submitting batch of %d tasks synchronously...\n", len(tasks))
	
	startTime := time.Now()
	taskIDs := make([]string, len(tasks))
	
	// Submit all tasks
	for i, task := range tasks {
		taskID, err := cli.SubmitTask(task)
		if err != nil {
			return fmt.Errorf("failed to submit task %d: %w", i, err)
		}
		taskIDs[i] = taskID
	}

	// Wait for completion and track progress
	completed := make(map[string]bool)
	var mu sync.Mutex
	
	fmt.Printf("Tasks submitted. Waiting for completion...\n")
	
	for len(completed) < len(taskIDs) {
		time.Sleep(500 * time.Millisecond)
		
		mu.Lock()
		for i, taskID := range taskIDs {
			if completed[taskID] {
				continue
			}
			
			task, err := cli.GetTask(taskID)
			if err != nil {
				fmt.Printf("Warning: Failed to get status for task %d (%s): %v\n", i+1, safeTaskIDShort(taskID), err)
				continue
			}
			
			if task.Status == models.StatusCompleted || task.Status == models.StatusFailed {
				completed[taskID] = true
				status := "✓"
				if task.Status == models.StatusFailed {
					status = "✗"
				}
				fmt.Printf("[%s] Task %d (%s) - %s: %s\n", status, i+1, safeTaskIDShort(taskID), task.Type, task.Status)
			}
		}
		mu.Unlock()
	}
	
	duration := time.Since(startTime)
	
	// Output final results
	if taskOutput == "json" {
		results := make([]map[string]interface{}, len(taskIDs))
		for i, taskID := range taskIDs {
			task, _ := cli.GetTask(taskID)
			results[i] = map[string]interface{}{
				"task_id": taskID,
				"type":    task.Type,
				"model":   task.Model,
				"status":  task.Status,
			}
			if task.Status == models.StatusCompleted && task.Result != nil {
				results[i]["result"] = task.Result
			}
			if task.Status == models.StatusFailed && task.Error != "" {
				results[i]["error"] = task.Error
			}
		}
		
		output := map[string]interface{}{
			"batch_size":     len(taskIDs),
			"duration_ms":    duration.Milliseconds(),
			"tasks":          results,
		}
		
		jsonOutput, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Printf("\nBatch completed in %v\n", duration)
		fmt.Printf("Total tasks: %d\n", len(taskIDs))
		
		// Count results
		successCount := 0
		failedCount := 0
		for _, taskID := range taskIDs {
			task, _ := cli.GetTask(taskID)
			if task.Status == models.StatusCompleted {
				successCount++
			} else if task.Status == models.StatusFailed {
				failedCount++
			}
		}
		
		fmt.Printf("Successful: %d\n", successCount)
		fmt.Printf("Failed: %d\n", failedCount)
		fmt.Printf("Success rate: %.1f%%\n", float64(successCount)/float64(len(taskIDs))*100)
	}

	return nil
}

func submitBatchAsync(cmd *cobra.Command, cli *client.Client, tasks []*models.Task) error {
	fmt.Printf("Submitting batch of %d tasks asynchronously...\n", len(tasks))
	
	taskIDs := make([]string, len(tasks))
	
	// Submit all tasks
	for i, task := range tasks {
		taskID, err := cli.SubmitTask(task)
		if err != nil {
			return fmt.Errorf("failed to submit task %d: %w", i, err)
		}
		taskIDs[i] = taskID
	}

	// Output result
	if taskOutput == "json" {
		results := make([]map[string]interface{}, len(taskIDs))
		for i, taskID := range taskIDs {
			results[i] = map[string]interface{}{
				"task_id": taskID,
				"type":    tasks[i].Type,
				"model":   tasks[i].Model,
				"status":  "submitted",
			}
		}
		
		output := map[string]interface{}{
			"batch_size": len(taskIDs),
			"status":     "submitted",
			"tasks":      results,
		}
		
		jsonOutput, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Printf("Batch submitted successfully\n")
		fmt.Printf("Total tasks: %d\n", len(taskIDs))
		fmt.Printf("Task IDs:\n")
		for i, taskID := range taskIDs {
			fmt.Printf("  %d. %s (%s)\n", i+1, taskID, tasks[i].Type)
		}
		fmt.Printf("\nUse 'ollama-queue status <task-id>' to check individual task status\n")
		fmt.Printf("Use 'ollama-queue list' to see all tasks\n")
	}

	return nil
}