package main

import (
	"fmt"
	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
	"github.com/liliang-cn/ollama-queue/pkg/client"
)

func main() {
	fmt.Println("Testing package imports...")
	
	// Test models package
	config := models.DefaultConfig()
	fmt.Printf("Default config loaded: %+v\n", config.ListenAddr)
	
	// Test client package 
	cli := client.New("localhost:8080")
	fmt.Printf("Client created successfully: %+v\n", cli != nil)
	
	// Test queue package
	manager, err := queue.NewQueueManager(config)
	if err != nil {
		fmt.Printf("QueueManager creation failed (expected without storage): %v\n", err)
	} else {
		fmt.Printf("QueueManager created successfully: %+v\n", manager != nil)
	}
	
	fmt.Println("All imports working correctly!")
}