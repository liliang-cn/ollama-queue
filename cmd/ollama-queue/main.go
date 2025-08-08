package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/liliang-cn/ollama-queue/cmd/ollama-queue/cmd"
)

func main() {
	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	// Handle shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down...")
		cancel()
	}()
	
	// Execute the root command
	if err := cmd.Execute(ctx); err != nil {
		log.Fatal(err)
	}
}