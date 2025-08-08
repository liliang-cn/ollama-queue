// Package main demonstrates basic usage of the ollama-queue library
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// Create queue manager with custom configuration
	qm, err := queue.NewQueueManagerWithOptions(
		queue.WithOllamaHost("http://localhost:11434"),
		queue.WithMaxWorkers(4),
		queue.WithStoragePath("./data"),
		queue.WithLogLevel("debug"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer qm.Close()

	// Start the queue manager
	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Example 1: Basic chat task
	fmt.Println("=== Example 1: Basic Chat Task ===")
	chatTask := queue.NewChatTask("llama2", []models.ChatMessage{
		{Role: "user", Content: "What is the capital of France?"},
	})

	taskID, err := qm.SubmitTask(chatTask)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Chat task submitted with ID: %s\n", taskID)

	// Example 2: High priority generation task with callback
	fmt.Println("\n=== Example 2: High Priority Generation Task ===")
	genTask := queue.NewGenerateTask(
		"codellama",
		"Write a Go function to calculate the factorial of a number",
		queue.WithTaskPriority(models.PriorityHigh),
		queue.WithGenerateSystem("You are an expert Go programmer"),
	)

	_, err = qm.SubmitTaskWithCallback(genTask, func(result *models.TaskResult) {
		if result.Success {
			fmt.Printf("Generation completed:\n%v\n", result.Data)
		} else {
			fmt.Printf("Generation failed: %s\n", result.Error)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	// Example 3: Embedding task with channel
	fmt.Println("\n=== Example 3: Embedding Task ===")
	embedTask := queue.NewEmbedTask(
		"nomic-embed-text",
		"This is a sample text for embedding to demonstrate the queue system",
		queue.WithTaskPriority(models.PriorityNormal),
	)

	resultChan := make(chan *models.TaskResult, 1)
	embedTaskID, err := qm.SubmitTaskWithChannel(embedTask, resultChan)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Embedding task submitted with ID: %s\n", embedTaskID)

	// Wait for embedding result
	result := <-resultChan
	if result.Success {
		fmt.Printf("Embedding completed, vector length: %d\n", len(result.Data.([]float64)))
	} else {
		fmt.Printf("Embedding failed: %s\n", result.Error)
	}

	// Example 4: Check queue status
	fmt.Println("\n=== Example 4: Queue Statistics ===")
	stats, err := qm.GetQueueStats()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Queue Statistics:\n")
	fmt.Printf("  Pending: %d\n", stats.PendingTasks)
	fmt.Printf("  Running: %d\n", stats.RunningTasks)
	fmt.Printf("  Completed: %d\n", stats.CompletedTasks)
	fmt.Printf("  Failed: %d\n", stats.FailedTasks)
	fmt.Printf("  Active Workers: %d\n", stats.WorkersActive)
	fmt.Printf("  Idle Workers: %d\n", stats.WorkersIdle)

	fmt.Println("\n=== Example completed ===")
}