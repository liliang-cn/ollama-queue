package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
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
)

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:   "submit [chat|generate|embed]",
	Short: "Submit a new task to the queue",
	Long: `Submit a new task to the Ollama queue. You can submit chat, generate, or embed tasks.

Examples:
  # Submit a chat task
  ollama-queue submit chat --model llama2 --messages "user:Hello, how are you?"

  # Submit a generation task
  ollama-queue submit generate --model codellama --prompt "Write a Go function"

  # Submit an embedding task
  ollama-queue submit embed --model nomic-embed-text --input "Sample text"`,
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

	// Mark required flags
	submitChatCmd.MarkFlagRequired("model")
	submitChatCmd.MarkFlagRequired("messages")
	submitGenerateCmd.MarkFlagRequired("model")
	submitGenerateCmd.MarkFlagRequired("prompt")
	submitEmbedCmd.MarkFlagRequired("model")
	submitEmbedCmd.MarkFlagRequired("input")
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

	// Override with CLI flags
	if ollamaHost != "" {
		cfg.OllamaHost = ollamaHost
	}
	if dataDir != "" {
		cfg.StoragePath = dataDir
	}

	// Create queue manager
	qm, err := queue.NewQueueManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create queue manager: %w", err)
	}
	defer qm.Close()

	ctx := cmd.Context()

	// Start queue manager
	if err := qm.Start(ctx); err != nil {
		return fmt.Errorf("failed to start queue manager: %w", err)
	}

	// Submit task
	if taskStream && (task.Type == models.TaskTypeChat || task.Type == models.TaskTypeGenerate) {
		return submitStreamingTask(ctx, qm, task)
	}

	taskID, err := qm.SubmitTask(task)
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

func parsePriority(priority string) (models.Priority, error) {
	switch strings.ToLower(priority) {
	case "low":
		return models.PriorityLow, nil
	case "normal":
		return models.PriorityNormal, nil
	case "high":
		return models.PriorityHigh, nil
	case "critical":
		return models.PriorityCritical, nil
	default:
		// Try to parse as integer
		if p, err := strconv.Atoi(priority); err == nil {
			return models.Priority(p), nil
		}
		return 0, fmt.Errorf("invalid priority: %s (valid values: low, normal, high, critical, or integer)", priority)
	}
}