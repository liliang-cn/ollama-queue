package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// Configure remote scheduling with OpenAI-compatible endpoints
	config := &models.Config{
		OllamaHost:    "http://localhost:11434",
		OllamaTimeout: 5 * time.Minute,
		
		MaxWorkers:        4,
		StoragePath:       "./data",
		CleanupInterval:   1 * time.Hour,
		MaxCompletedTasks: 1000,
		
		SchedulingInterval: 1 * time.Second,
		BatchSize:         10,
		
		RetryConfig: models.RetryConfig{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
		
		// Enable remote scheduling
		RemoteScheduling: models.RemoteSchedulingConfig{
			Enabled: true,
			Endpoints: []models.RemoteEndpointConfig{
				{
					Name:     "OpenAI",
					BaseURL:  "https://api.openai.com/v1",
					APIKey:   "sk-your-openai-api-key",
					Priority: 10, // Higher priority
					Enabled:  true,
				},
				{
					Name:     "DeepSeek",
					BaseURL:  "https://api.deepseek.com/v1",
					APIKey:   "sk-your-deepseek-api-key",
					Priority: 8,
					Enabled:  true,
				},
				{
					Name:     "LocalAI",
					BaseURL:  "http://localhost:8080/v1",
					APIKey:   "", // No API key needed for local
					Priority: 5,
					Enabled:  true,
				},
			},
			HealthCheckInterval: 30 * time.Second,
			FallbackToLocal:     true, // Fallback to local Ollama if all remotes fail
		},
		
		LogLevel:   "info",
		ListenAddr: "localhost:7125",
	}
	
	// Create queue manager
	qm, err := queue.NewQueueManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer qm.Stop()
	
	// Start queue manager
	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		log.Fatal(err)
	}
	
	// Example 1: Submit a high-priority task for remote execution
	task1 := &models.Task{
		Type:     models.TaskTypeChat,
		Priority: models.PriorityHigh, // High priority tasks go to remote endpoints
		Model:    "gpt-4",
		Payload: models.ChatTaskPayload{
			Messages: []models.ChatMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "Explain quantum computing in simple terms."},
			},
			Stream: false,
		},
		RemoteExecution: true, // Explicitly request remote execution
		MaxRetries:      2,
	}
	
	taskID1, err := qm.SubmitTask(task1)
	if err != nil {
		log.Printf("Failed to submit task 1: %v", err)
	} else {
		fmt.Printf("Submitted remote task: %s\n", taskID1)
	}
	
	// Example 2: Submit a normal priority task (will use local Ollama)
	task2 := &models.Task{
		Type:     models.TaskTypeGenerate,
		Priority: models.PriorityNormal,
		Model:    "llama2",
		Payload: models.GenerateTaskPayload{
			Prompt: "Write a haiku about programming.",
			Stream: false,
		},
		RemoteExecution: false, // Use local execution
		MaxRetries:      3,
	}
	
	taskID2, err := qm.SubmitTask(task2)
	if err != nil {
		log.Printf("Failed to submit task 2: %v", err)
	} else {
		fmt.Printf("Submitted local task: %s\n", taskID2)
	}
	
	// Example 3: Submit an embedding task for remote execution
	task3 := &models.Task{
		Type:     models.TaskTypeEmbed,
		Priority: models.PriorityCritical, // Critical tasks always go remote if available
		Model:    "text-embedding-ada-002",
		Payload: models.EmbedTaskPayload{
			Input:    "Machine learning is a subset of artificial intelligence.",
			Truncate: false,
		},
		RemoteExecution: true,
		MaxRetries:      1,
	}
	
	taskID3, err := qm.SubmitTask(task3)
	if err != nil {
		log.Printf("Failed to submit task 3: %v", err)
	} else {
		fmt.Printf("Submitted embedding task: %s\n", taskID3)
	}
	
	// Wait for tasks to complete
	time.Sleep(10 * time.Second)
	
	// Check task statuses
	for _, taskID := range []string{taskID1, taskID2, taskID3} {
		task, err := qm.GetTask(taskID)
			if err != nil {
				log.Printf("Failed to get status for task %s: %v", taskID, err)
			} else {
				fmt.Printf("Task %s status: %v\n", taskID, task.Status)
				if task.Status == models.StatusCompleted {
					fmt.Printf("  Result: %v\n", task.Result)
				} else if task.Status == models.StatusFailed {
					fmt.Printf("  Error: %s\n", task.Error)
				}
			}
	}
	
	// Get queue statistics
	stats, _ := qm.GetQueueStats()
	fmt.Printf("\nQueue Statistics:\n")
	fmt.Printf("  Pending: %d\n", stats.PendingTasks)
	fmt.Printf("  Running: %d\n", stats.RunningTasks)
	fmt.Printf("  Completed: %d\n", stats.CompletedTasks)
	fmt.Printf("  Failed: %d\n", stats.FailedTasks)
}