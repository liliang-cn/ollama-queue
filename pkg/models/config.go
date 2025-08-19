package models

import "time"

// Config represents the configuration for the queue manager
type Config struct {
	// Ollama configuration
	OllamaHost    string        `json:"ollama_host" yaml:"ollama_host"`
	OllamaTimeout time.Duration `json:"ollama_timeout" yaml:"ollama_timeout"`

	// Queue configuration
	MaxWorkers        int           `json:"max_workers" yaml:"max_workers"`
	StoragePath       string        `json:"storage_path" yaml:"storage_path"`
	CleanupInterval   time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
	MaxCompletedTasks int           `json:"max_completed_tasks" yaml:"max_completed_tasks"`

	// Scheduling configuration
	SchedulingInterval time.Duration `json:"scheduling_interval" yaml:"scheduling_interval"`
	BatchSize         int           `json:"batch_size" yaml:"batch_size"`

	// Retry configuration
	RetryConfig RetryConfig `json:"retry_config" yaml:"retry_config"`

	// Remote scheduling configuration
	RemoteScheduling RemoteSchedulingConfig `json:"remote_scheduling" yaml:"remote_scheduling"`

	// Logging configuration
	LogLevel string `json:"log_level" yaml:"log_level"`
	LogFile  string `json:"log_file" yaml:"log_file"`

	// Server configuration
	ListenAddr string `json:"listen_addr" yaml:"listen_addr"`
}

// RetryConfig represents retry configuration for failed tasks
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries" yaml:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay" yaml:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay" yaml:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor" yaml:"backoff_factor"`
}

// RemoteSchedulingConfig represents configuration for remote scheduling
type RemoteSchedulingConfig struct {
	Enabled             bool                  `json:"enabled" yaml:"enabled"`
	Endpoints           []RemoteEndpointConfig `json:"endpoints" yaml:"endpoints"`
	HealthCheckInterval time.Duration         `json:"health_check_interval" yaml:"health_check_interval"`
	FallbackToLocal     bool                  `json:"fallback_to_local" yaml:"fallback_to_local"`
	MaxLocalQueueSize   int                   `json:"max_local_queue_size" yaml:"max_local_queue_size"`
	LocalFirstPolicy    bool                  `json:"local_first_policy" yaml:"local_first_policy"`
}

// RemoteEndpointConfig represents a remote endpoint configuration
type RemoteEndpointConfig struct {
	Name     string `json:"name" yaml:"name"`
	BaseURL  string `json:"base_url" yaml:"base_url"`
	APIKey   string `json:"api_key" yaml:"api_key"`
	Priority int    `json:"priority" yaml:"priority"`
	Enabled  bool   `json:"enabled" yaml:"enabled"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		OllamaHost:    "http://localhost:11434",
		OllamaTimeout: 5 * time.Minute,
		
		MaxWorkers:        4,
		StoragePath:       "./data",
		CleanupInterval:   1 * time.Hour,
		MaxCompletedTasks: 1000,
		
		SchedulingInterval: 1 * time.Second,
		BatchSize:         10,
		
		RetryConfig: RetryConfig{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
		
		RemoteScheduling: RemoteSchedulingConfig{
			Enabled:             false,
			Endpoints:           []RemoteEndpointConfig{},
			HealthCheckInterval: 30 * time.Second,
			FallbackToLocal:     true,
			MaxLocalQueueSize:   100,  // Default: use remote when local queue has 100+ tasks
			LocalFirstPolicy:    true, // Default: prefer local execution
		},
		
		LogLevel: "info",
		LogFile:  "",

		ListenAddr: "localhost:7125",
	}
}