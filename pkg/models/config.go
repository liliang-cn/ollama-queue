package models

import "time"

// Config represents basic configuration needed for executors
type Config struct {
	// Ollama configuration
	OllamaHost    string        `json:"ollama_host" yaml:"ollama_host"`
	OllamaTimeout time.Duration `json:"ollama_timeout" yaml:"ollama_timeout"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		OllamaHost:    "http://localhost:11434",
		OllamaTimeout: 5 * time.Minute,
	}
}