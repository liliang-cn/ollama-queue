// Package main demonstrates client-server usage of the ollama-queue system
package main

import (
	"fmt"
	"log"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// Connect to the queue server (make sure to run `ollama-queue serve` first)
	cli := client.New("localhost:7125")

	// Example 1: Submit a chat task via HTTP client
	fmt.Println("=== Example 1: Chat Task via HTTP Client ===")
	chatTask := queue.NewChatTask("qwen3", []models.ChatMessage{
		{Role: "user", Content: "What is the capital of Japan?"},
	}, queue.WithTaskPriority(models.PriorityHigh))

	taskID, err := cli.SubmitTask(chatTask)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Chat task submitted with ID: %s\n", taskID)

	// Example 2: Submit a generation task
	fmt.Println("\n=== Example 2: Generation Task ===")
	genTask := queue.NewGenerateTask(
		"qwen3",
		"Write a Python function to sort a list using quicksort",
		queue.WithTaskPriority(models.PriorityNormal),
	)

	genTaskID, err := cli.SubmitTask(genTask)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generation task submitted with ID: %s\n", genTaskID)

	// Example 3: Submit an embedding task
	fmt.Println("\n=== Example 3: Embedding Task ===")
	embedTask := queue.NewEmbedTask(
		"nomic-embed-text",
		"Client-server architecture for distributed AI task processing",
		queue.WithTaskPriority(models.PriorityLow),
	)

	embedTaskID, err := cli.SubmitTask(embedTask)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Embedding task submitted with ID: %s\n", embedTaskID)

	// Example 4: List all tasks
	fmt.Println("\n=== Example 4: List All Tasks ===")
	tasks, err := cli.ListTasks(models.TaskFilter{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d tasks:\n", len(tasks))
	for _, task := range tasks {
		fmt.Printf("  Task %s: %s (%s) - Priority: %d\n",
			task.ID[:8], task.Type, task.Status, task.Priority)
	}

	// Example 5: Get specific task details
	fmt.Println("\n=== Example 5: Get Task Details ===")
	task, err := cli.GetTask(taskID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task Details:\n")
	fmt.Printf("  ID: %s\n", task.ID)
	fmt.Printf("  Type: %s\n", task.Type)
	fmt.Printf("  Status: %s\n", task.Status)
	fmt.Printf("  Priority: %d\n", task.Priority)
	fmt.Printf("  Model: %s\n", task.Model)
	fmt.Printf("  Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))

	// Example 6: Update task priority
	fmt.Println("\n=== Example 6: Update Task Priority ===")
	err = cli.UpdateTaskPriority(genTaskID, models.PriorityCritical)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated task %s priority to Critical\n", genTaskID[:8])

	// Example 7: Get queue statistics
	fmt.Println("\n=== Example 7: Queue Statistics ===")
	stats, err := cli.GetQueueStats()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Queue Statistics:\n")
	fmt.Printf("  Total Tasks: %d\n", stats.TotalTasks)
	fmt.Printf("  Pending: %d\n", stats.PendingTasks)
	fmt.Printf("  Running: %d\n", stats.RunningTasks)
	fmt.Printf("  Completed: %d\n", stats.CompletedTasks)
	fmt.Printf("  Failed: %d\n", stats.FailedTasks)
	fmt.Printf("  Cancelled: %d\n", stats.CancelledTasks)

	// Example 8: Cancel a task (optional, comment out if you want to see results)
	// fmt.Println("\n=== Example 8: Cancel Task ===")
	// err = cli.CancelTask(embedTaskID)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// fmt.Printf("Cancelled task %s\n", embedTaskID[:8])

	fmt.Println("\n=== Client-Server Example completed ===")
	fmt.Println("Check the web interface at http://localhost:7125 to monitor tasks in real-time!")
}
