package main

import (
	"context"
	"log"

	"github.com/liliang-cn/ollama-queue/cmd/ollama-queue/cmd"
)

func main() {
	// Execute the root command
	if err := cmd.Execute(context.Background()); err != nil {
		log.Fatal(err)
	}
}