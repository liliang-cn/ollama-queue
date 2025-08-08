package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"gopkg.in/yaml.v3"
)

// ConfigLoader provides methods to load configuration from various sources
type ConfigLoader struct {
	searchPaths []string
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	homeDir, _ := os.UserHomeDir()
	
	return &ConfigLoader{
		searchPaths: []string{
			"./config.yaml",
			"./config.yml",
			"./config.json",
			filepath.Join(homeDir, ".ollama-queue", "config.yaml"),
			filepath.Join(homeDir, ".ollama-queue", "config.yml"),
			filepath.Join(homeDir, ".ollama-queue", "config.json"),
			"/etc/ollama-queue/config.yaml",
			"/etc/ollama-queue/config.yml",
			"/etc/ollama-queue/config.json",
		},
	}
}

// LoadConfig loads configuration from file and environment variables
func (cl *ConfigLoader) LoadConfig() (*models.Config, error) {
	config := models.DefaultConfig()
	
	// Try to load from file
	if err := cl.loadFromFile(config); err != nil {
		// If no config file found, use defaults with environment overrides
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}
	
	// Override with environment variables
	cl.loadFromEnv(config)
	
	// Validate configuration
	if err := cl.validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// LoadConfigFromFile loads configuration from a specific file
func (cl *ConfigLoader) LoadConfigFromFile(filePath string) (*models.Config, error) {
	config := models.DefaultConfig()
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, config)
	case ".json":
		err = json.Unmarshal(data, config)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Override with environment variables
	cl.loadFromEnv(config)
	
	// Validate configuration
	if err := cl.validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// SaveConfigToFile saves configuration to a file
func (cl *ConfigLoader) SaveConfigToFile(config *models.Config, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	ext := filepath.Ext(filePath)
	var data []byte
	var err error
	
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

func (cl *ConfigLoader) loadFromFile(config *models.Config) error {
	for _, path := range cl.searchPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		
		ext := filepath.Ext(path)
		switch ext {
		case ".yaml", ".yml":
			err = yaml.Unmarshal(data, config)
		case ".json":
			err = json.Unmarshal(data, config)
		default:
			continue
		}
		
		if err != nil {
			return fmt.Errorf("failed to parse config file %s: %w", path, err)
		}
		
		return nil
	}
	
	return os.ErrNotExist
}

func (cl *ConfigLoader) loadFromEnv(config *models.Config) {
	// Ollama configuration
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		config.OllamaHost = host
	}
	
	if timeout := os.Getenv("OLLAMA_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.OllamaTimeout = d
		}
	}
	
	// Queue configuration
	if workers := os.Getenv("QUEUE_MAX_WORKERS"); workers != "" {
		if w, err := parseInt(workers); err == nil {
			config.MaxWorkers = w
		}
	}
	
	if path := os.Getenv("QUEUE_STORAGE_PATH"); path != "" {
		config.StoragePath = path
	}
	
	if interval := os.Getenv("QUEUE_CLEANUP_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.CleanupInterval = d
		}
	}
	
	if maxTasks := os.Getenv("QUEUE_MAX_COMPLETED_TASKS"); maxTasks != "" {
		if m, err := parseInt(maxTasks); err == nil {
			config.MaxCompletedTasks = m
		}
	}
	
	// Scheduler configuration
	if interval := os.Getenv("SCHEDULER_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.SchedulingInterval = d
		}
	}
	
	if batchSize := os.Getenv("SCHEDULER_BATCH_SIZE"); batchSize != "" {
		if b, err := parseInt(batchSize); err == nil {
			config.BatchSize = b
		}
	}
	
	// Retry configuration
	if maxRetries := os.Getenv("RETRY_MAX_RETRIES"); maxRetries != "" {
		if m, err := parseInt(maxRetries); err == nil {
			config.RetryConfig.MaxRetries = m
		}
	}
	
	if initialDelay := os.Getenv("RETRY_INITIAL_DELAY"); initialDelay != "" {
		if d, err := time.ParseDuration(initialDelay); err == nil {
			config.RetryConfig.InitialDelay = d
		}
	}
	
	if maxDelay := os.Getenv("RETRY_MAX_DELAY"); maxDelay != "" {
		if d, err := time.ParseDuration(maxDelay); err == nil {
			config.RetryConfig.MaxDelay = d
		}
	}
	
	if backoffFactor := os.Getenv("RETRY_BACKOFF_FACTOR"); backoffFactor != "" {
		if f, err := parseFloat64(backoffFactor); err == nil {
			config.RetryConfig.BackoffFactor = f
		}
	}
	
	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}
	
	if file := os.Getenv("LOG_FILE"); file != "" {
		config.LogFile = file
	}
}

func (cl *ConfigLoader) validateConfig(config *models.Config) error {
	if config.OllamaHost == "" {
		return fmt.Errorf("ollama_host is required")
	}
	
	if config.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be greater than 0")
	}
	
	if config.StoragePath == "" {
		return fmt.Errorf("storage_path is required")
	}
	
	if config.RetryConfig.MaxRetries < 0 {
		return fmt.Errorf("retry_config.max_retries must be non-negative")
	}
	
	if config.RetryConfig.InitialDelay <= 0 {
		return fmt.Errorf("retry_config.initial_delay must be positive")
	}
	
	if config.RetryConfig.MaxDelay <= 0 {
		return fmt.Errorf("retry_config.max_delay must be positive")
	}
	
	if config.RetryConfig.BackoffFactor <= 1.0 {
		return fmt.Errorf("retry_config.backoff_factor must be greater than 1.0")
	}
	
	return nil
}

// Helper functions
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func parseFloat64(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}