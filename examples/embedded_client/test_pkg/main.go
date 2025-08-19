package main

import (
	"fmt"
	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
)

func main() {
	fmt.Println("Testing package imports...")
	
	// Test models package
	config := models.DefaultConfig()
	fmt.Printf("Default config loaded: %s\n", config.ListenAddr)
	
	// Test client package 
	cli := client.New("localhost:8080")
	fmt.Printf("Client created successfully: %t\n", cli != nil)
	
	fmt.Println("All imports working correctly!")
}
