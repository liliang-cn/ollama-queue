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
	// Configure with local-first policy
	config := &models.Config{
		OllamaHost:    "http://localhost:11434",
		OllamaTimeout: 5 * time.Minute,

		MaxWorkers:        2, // Limited workers to demonstrate queue filling
		StoragePath:       "./data",
		CleanupInterval:   1 * time.Hour,
		MaxCompletedTasks: 1000,

		SchedulingInterval: 1 * time.Second,
		BatchSize:          10,

		RetryConfig: models.RetryConfig{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},

		// Local-first remote scheduling
		RemoteScheduling: models.RemoteSchedulingConfig{
			Enabled:             true,
			MaxLocalQueueSize:   5,    // Small queue to demonstrate overflow to remote
			LocalFirstPolicy:    true, // Prefer local execution
			HealthCheckInterval: 30 * time.Second,
			FallbackToLocal:     true,
			Endpoints: []models.RemoteEndpointConfig{
				{
					Name:     "OpenAI",
					Priority: 10,
					Enabled:  true,
				},
				{
					Name:     "LocalAI",
					BaseURL:  "http://localhost:8080/v1",
					APIKey:   "",
					Priority: 5,
					Enabled:  true,
				},
			},
		},

		LogLevel:   "info",
		ListenAddr: "localhost:7125",
	}

	qm, err := queue.NewQueueManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer qm.Stop()

	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Local-First Scheduling Demo ===")
	fmt.Println("Submitting tasks to demonstrate local-first policy:")
	fmt.Println("- Normal/Low priority: stays local")
	fmt.Println("- High priority + queue full: goes remote")
	fmt.Println("- Critical priority: always goes to best endpoint")
	fmt.Println()

	// Submit multiple normal priority tasks to fill local queue
	fmt.Println("1. Submitting normal priority tasks (should stay local)...")
	for i := 0; i < 3; i++ {
		task := &models.Task{
			Type:     models.TaskTypeGenerate,
			Priority: models.PriorityNormal,
			Model:    "qwen3",
			Payload: models.GenerateTaskPayload{
				Prompt: fmt.Sprintf("Write a short poem about task %d.", i+1),
				Stream: false,
			},
			RemoteExecution: false,
			MaxRetries:      2,
		}

		taskID, err := qm.SubmitTask(task)
		if err != nil {
			log.Printf("Failed to submit normal task %d: %v", i+1, err)
		} else {
			fmt.Printf("  Submitted normal task %d: %s\n", i+1, taskID)
		}
	}

	// Wait a bit and check queue status
	time.Sleep(2 * time.Second)
	stats, _ := qm.GetQueueStats()
	fmt.Printf("\nQueue status after normal tasks: %d pending, %d running\n", stats.PendingTasks, stats.RunningTasks)

	// Now submit high priority tasks (should go remote if queue is full)
	fmt.Println("\n2. Submitting high priority tasks (may go remote if queue full)...")
	for i := 0; i < 3; i++ {
		task := &models.Task{
			Type:     models.TaskTypeChat,
			Priority: models.PriorityHigh,
			Model:    "gpt-3.5-turbo",
			Payload: models.ChatTaskPayload{
				Messages: []models.ChatMessage{
					{Role: "user", Content: fmt.Sprintf("Explain concept %d briefly.", i+1)},
				},
				Stream: false,
			},
			RemoteExecution: false, // Not explicitly requested, depends on queue status
			MaxRetries:      2,
		}

		taskID, err := qm.SubmitTask(task)
		if err != nil {
			log.Printf("Failed to submit high priority task %d: %v", i+1, err)
		} else {
			fmt.Printf("  Submitted high priority task %d: %s\n", i+1, taskID)
		}
	}

	// Submit a critical priority task (should always go to best endpoint)
	fmt.Println("\n3. Submitting critical priority task (always goes to best endpoint)...")
	criticalTask := &models.Task{
		Type:     models.TaskTypeChat,
		Priority: models.PriorityCritical,
		Model:    "gpt-4",
		Payload: models.ChatTaskPayload{
			Messages: []models.ChatMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "What is the meaning of life?"},
			},
			Stream: false,
		},
		RemoteExecution: false, // Will go remote due to critical priority
		MaxRetries:      1,
	}

	criticalTaskID, err := qm.SubmitTask(criticalTask)
	if err != nil {
		log.Printf("Failed to submit critical task: %v", err)
	} else {
		fmt.Printf("  Submitted critical task: %s\n", criticalTaskID)
	}

	// Submit a task with explicit remote execution
	fmt.Println("\n4. Submitting task with explicit remote execution...")
	remoteTask := &models.Task{
		Type:     models.TaskTypeGenerate,
		Priority: models.PriorityNormal,
		Model:    "gpt-3.5-turbo",
		Payload: models.GenerateTaskPayload{
			Prompt: "Write a haiku about remote computing.",
			Stream: false,
		},
		RemoteExecution: true, // Explicitly request remote
		MaxRetries:      2,
	}

	remoteTaskID, err := qm.SubmitTask(remoteTask)
	if err != nil {
		log.Printf("Failed to submit explicit remote task: %v", err)
	} else {
		fmt.Printf("  Submitted explicit remote task: %s\n", remoteTaskID)
	}

	// Final queue status
	fmt.Println("\n=== Final Queue Status ===")
	finalStats, _ := qm.GetQueueStats()
	fmt.Printf("Pending: %d, Running: %d, Completed: %d, Failed: %d\n",
		finalStats.PendingTasks, finalStats.RunningTasks,
		finalStats.CompletedTasks, finalStats.FailedTasks)

	fmt.Printf("\nPriority distribution:\n")
	for priority, count := range finalStats.QueuesByPriority {
		if count > 0 {
			fmt.Printf("  Priority %d: %d tasks\n", priority, count)
		}
	}

	fmt.Println("\n=== Policy Summary ===")
	fmt.Println("Local-First Policy Rules:")
	fmt.Println("✓ Normal/Low priority → Local (unless queue full)")
	fmt.Println("✓ High priority + queue full → Remote")
	fmt.Println("✓ Critical priority → Always best endpoint (remote)")
	fmt.Println("✓ Explicit remote flag → Always remote")
	fmt.Println("✓ Remote failure → Fallback to local")

	// Wait for some tasks to complete
	time.Sleep(10 * time.Second)
}
