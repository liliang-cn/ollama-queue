package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/storage"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared"
)

// RemoteScheduler implements remote task scheduling using OpenAI-compatible APIs
type RemoteScheduler struct {
	client            openai.Client
	hasClient         bool
	endpoints         []RemoteEndpoint
	mu                sync.RWMutex
	fallback          Scheduler // Fallback to local scheduler if remote fails
	maxLocalQueueSize int       // Maximum local queue size before using remote
	localFirstPolicy  bool      // Prefer local execution when possible
	storage           storage.Storage
}

// RemoteEndpoint represents a remote API endpoint configuration
type RemoteEndpoint struct {
	Name      string
	BaseURL   string
	APIKey    string
	Priority  int // Higher priority endpoints are tried first
	Available bool
	LastCheck time.Time
}

// RemoteSchedulerConfig contains configuration for remote scheduler
type RemoteSchedulerConfig struct {
	Endpoints           []RemoteEndpoint
	HealthCheckInterval time.Duration
	FallbackScheduler   Scheduler
	MaxLocalQueueSize   int  // Maximum local queue size before offloading to remote
	LocalFirstPolicy    bool // If true, prefer local execution unless queue is full
	Storage             storage.Storage
}

// NewRemoteScheduler creates a new remote scheduler with OpenAI-compatible endpoints
func NewRemoteScheduler(config RemoteSchedulerConfig) *RemoteScheduler {
	rs := &RemoteScheduler{
		endpoints:         config.Endpoints,
		fallback:          config.FallbackScheduler,
		maxLocalQueueSize: config.MaxLocalQueueSize,
		localFirstPolicy:  config.LocalFirstPolicy,
		storage:           config.Storage,
	}

	if rs.maxLocalQueueSize == 0 {
		rs.maxLocalQueueSize = 100 // Default to 100 tasks
	}

	if len(config.Endpoints) > 0 {
		rs.selectBestEndpoint()
	}

	if config.HealthCheckInterval > 0 {
		go rs.healthCheckLoop(config.HealthCheckInterval)
	}

	return rs
}

// AddTask delegates task to remote endpoint or fallback scheduler
func (rs *RemoteScheduler) AddTask(task *models.Task) error {
	rs.mu.RLock()
	useRemote := rs.shouldUseRemote(task)
	hasClient := rs.hasClient
	rs.mu.RUnlock()

	if useRemote && hasClient {
		if err := rs.scheduleRemotely(task); err == nil {
			return nil
		}
	}

	if rs.fallback != nil {
		return rs.fallback.AddTask(task)
	}

	return fmt.Errorf("no available scheduler for task %s", task.ID)
}

// GetNextTask retrieves next task from fallback scheduler
func (rs *RemoteScheduler) GetNextTask() *models.Task {
	if rs.fallback != nil {
		return rs.fallback.GetNextTask()
	}
	return nil
}

// RemoveTask removes task from fallback scheduler
func (rs *RemoteScheduler) RemoveTask(taskID string) error {
	if rs.fallback != nil {
		return rs.fallback.RemoveTask(taskID)
	}
	return nil
}

// UpdateTaskPriority updates task priority in fallback scheduler
func (rs *RemoteScheduler) UpdateTaskPriority(taskID string, priority models.Priority) error {
	if rs.fallback != nil {
		return rs.fallback.UpdateTaskPriority(taskID, priority)
	}
	return nil
}

// GetQueueLength returns queue length from fallback scheduler
func (rs *RemoteScheduler) GetQueueLength() int {
	if rs.fallback != nil {
		return rs.fallback.GetQueueLength()
	}
	return 0
}

// GetQueueStats returns queue statistics from fallback scheduler
func (rs *RemoteScheduler) GetQueueStats() map[models.Priority]int {
	if rs.fallback != nil {
		return rs.fallback.GetQueueStats()
	}
	return make(map[models.Priority]int)
}

// scheduleRemotely sends task to remote endpoint for execution
func (rs *RemoteScheduler) scheduleRemotely(task *models.Task) error {
	log.Printf("Attempting to schedule task %s remotely", task.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bestEndpoint := rs.selectBestEndpoint()
	if bestEndpoint == nil {
		return fmt.Errorf("no available remote endpoint")
	}
	task.ExecutedOn = bestEndpoint.Name

	var err error
	switch task.Type {
	case models.TaskTypeChat:
		err = rs.executeChatRemotely(ctx, task)
	case models.TaskTypeGenerate:
		err = rs.executeCompletionRemotely(ctx, task)
	case models.TaskTypeEmbed:
		err = rs.executeEmbeddingRemotely(ctx, task)
	default:
		err = fmt.Errorf("unsupported task type for remote execution: %s", task.Type)
	}

	if err != nil {
		log.Printf("Remote execution for task %s failed: %v. Falling back to local scheduler.", task.ID, err)
		task.Status = models.StatusPending // Reset status for local queue
		if rs.fallback != nil {
			return rs.fallback.AddTask(task)
		}
		return err
	}

	log.Printf("Task %s scheduled remotely successfully", task.ID)
	return nil
}

// executeChatRemotely executes chat task on remote endpoint
func (rs *RemoteScheduler) executeChatRemotely(ctx context.Context, task *models.Task) error {
	payload, ok := task.Payload.(models.ChatTaskPayload)
	if !ok {
		if mapPayload, ok := task.Payload.(map[string]any); ok {
			payload = rs.mapToChatPayload(mapPayload)
		} else {
			return fmt.Errorf("invalid chat payload")
		}
	}

	messages := make([]openai.ChatCompletionMessageParamUnion, len(payload.Messages))
	for i, msg := range payload.Messages {
		switch msg.Role {
		case "system":
			messages[i] = openai.SystemMessage(msg.Content)
		case "user":
			messages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(msg.Content)
		default:
			messages[i] = openai.UserMessage(msg.Content)
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(task.Model),
		Messages: messages,
	}

	if payload.Stream {
		stream := rs.client.Chat.Completions.NewStreaming(ctx, params)
		var fullContent string
		for stream.Next() {
			response := stream.Current()
			fullContent += response.Choices[0].Delta.Content
		}
		if stream.Err() != nil {
			return stream.Err()
		}

		task.Result = &models.TaskResult{
			TaskID:  task.ID,
			Success: true,
			Data:    fullContent,
		}
	} else {
		completion, err := rs.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return fmt.Errorf("remote chat execution failed: %w", err)
		}

		task.Result = &models.TaskResult{
			TaskID:  task.ID,
			Success: true,
			Data:    completion,
		}
	}

	task.Status = models.StatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	return rs.storage.SaveTask(task)
}

// executeCompletionRemotely executes text completion task on remote endpoint
func (rs *RemoteScheduler) executeCompletionRemotely(ctx context.Context, task *models.Task) error {
	payload, ok := task.Payload.(models.GenerateTaskPayload)
	if !ok {
		if mapPayload, ok := task.Payload.(map[string]any); ok {
			payload = rs.mapToGeneratePayload(mapPayload)
		} else {
			return fmt.Errorf("invalid generate payload")
		}
	}

	params := openai.CompletionNewParams{
		Model:  openai.CompletionNewParamsModel(task.Model),
		Prompt: openai.CompletionNewParamsPromptUnion{OfString: openai.String(payload.Prompt)},
	}

	if payload.Stream {
		stream := rs.client.Completions.NewStreaming(ctx, params)
		var fullContent string
		for stream.Next() {
			response := stream.Current()
			fullContent += response.Choices[0].Text
		}
		if stream.Err() != nil {
			return stream.Err()
		}

		task.Result = &models.TaskResult{
			TaskID:  task.ID,
			Success: true,
			Data:    fullContent,
		}
	} else {
		completion, err := rs.client.Completions.New(ctx, params)
		if err != nil {
			return fmt.Errorf("remote completion execution failed: %w", err)
		}

		task.Result = &models.TaskResult{
			TaskID:  task.ID,
			Success: true,
			Data:    completion,
		}
	}

	task.Status = models.StatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	return rs.storage.SaveTask(task)
}

// executeEmbeddingRemotely executes embedding task on remote endpoint
func (rs *RemoteScheduler) executeEmbeddingRemotely(ctx context.Context, task *models.Task) error {
	payload, ok := task.Payload.(models.EmbedTaskPayload)
	if !ok {
		if mapPayload, ok := task.Payload.(map[string]any); ok {
			payload = rs.mapToEmbedPayload(mapPayload)
		} else {
			return fmt.Errorf("invalid embed payload")
		}
	}

	inputStr, ok := payload.Input.(string)
	if !ok {
		return fmt.Errorf("embedding input must be a string")
	}

	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(task.Model),
		Input: openai.EmbeddingNewParamsInputUnion{OfString: openai.String(inputStr)},
	}

	embedding, err := rs.client.Embeddings.New(ctx, params)
	if err != nil {
		return fmt.Errorf("remote embedding execution failed: %w", err)
	}

	task.Result = &models.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Data:    embedding,
	}

	task.Status = models.StatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	return rs.storage.SaveTask(task)
}

// shouldUseRemote determines whether to use remote scheduling based on local-first policy
func (rs *RemoteScheduler) shouldUseRemote(task *models.Task) bool {
	if !rs.isRemoteTaskSupported(task) {
		return false
	}

	if !rs.localFirstPolicy {
		return task.Priority >= models.PriorityHigh || task.RemoteExecution
	}

	if task.RemoteExecution {
		return true
	}

	if task.Priority >= models.PriorityCritical {
		return true
	}

	localQueueSize := 0
	if rs.fallback != nil {
		localQueueSize = rs.fallback.GetQueueLength()
	}

	if localQueueSize >= rs.maxLocalQueueSize && task.Priority >= models.PriorityHigh {
		return true
	}

	return false
}

// isRemoteTaskSupported checks if task can be executed remotely
func (rs *RemoteScheduler) isRemoteTaskSupported(task *models.Task) bool {
	switch task.Type {
	case models.TaskTypeChat, models.TaskTypeGenerate, models.TaskTypeEmbed:
		return true
	default:
		return false
	}
}

// selectBestEndpoint selects the best available endpoint
func (rs *RemoteScheduler) selectBestEndpoint() *RemoteEndpoint {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	var bestEndpoint *RemoteEndpoint
	for i := range rs.endpoints {
		ep := &rs.endpoints[i]
		if ep.Available {
			if bestEndpoint == nil || ep.Priority > bestEndpoint.Priority {
				bestEndpoint = ep
			}
		}
	}

	if bestEndpoint != nil {
		options := []option.RequestOption{
			option.WithBaseURL(bestEndpoint.BaseURL),
		}
		if bestEndpoint.APIKey != "" {
			options = append(options, option.WithAPIKey(bestEndpoint.APIKey))
		}
		rs.client = openai.NewClient(options...)
		rs.hasClient = true
	}
	return bestEndpoint
}

// healthCheckLoop periodically checks endpoint health
func (rs *RemoteScheduler) healthCheckLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rs.checkEndpointHealth()
	}
}

// checkEndpointHealth checks the health of all endpoints
func (rs *RemoteScheduler) checkEndpointHealth() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	for i := range rs.endpoints {
		ep := &rs.endpoints[i]

		options := []option.RequestOption{
			option.WithBaseURL(ep.BaseURL),
		}
		if ep.APIKey != "" {
			options = append(options, option.WithAPIKey(ep.APIKey))
		}
		client := openai.NewClient(options...)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := client.Models.List(ctx)
		cancel()

		ep.Available = err == nil
		ep.LastCheck = time.Now()
	}

	rs.selectBestEndpoint()
}

// Helper methods for payload conversion
func (rs *RemoteScheduler) mapToChatPayload(payload map[string]any) models.ChatTaskPayload {
	chatPayload := models.ChatTaskPayload{}

	if messages, ok := payload["messages"].([]any); ok {
		for _, msgAny := range messages {
			if msgMap, ok := msgAny.(map[string]any); ok {
				msg := models.ChatMessage{}
				if role, ok := msgMap["role"].(string); ok {
					msg.Role = role
				}
				if content, ok := msgMap["content"].(string); ok {
					msg.Content = content
				}
				chatPayload.Messages = append(chatPayload.Messages, msg)
			}
		}
	}

	if system, ok := payload["system"].(string); ok {
		chatPayload.System = system
	}

	if stream, ok := payload["stream"].(bool); ok {
		chatPayload.Stream = stream
	}

	return chatPayload
}

func (rs *RemoteScheduler) mapToGeneratePayload(payload map[string]any) models.GenerateTaskPayload {
	genPayload := models.GenerateTaskPayload{}

	if prompt, ok := payload["prompt"].(string); ok {
		genPayload.Prompt = prompt
	}

	if system, ok := payload["system"].(string); ok {
		genPayload.System = system
	}

	if template, ok := payload["template"].(string); ok {
		genPayload.Template = template
	}

	if raw, ok := payload["raw"].(bool); ok {
		genPayload.Raw = raw
	}

	if stream, ok := payload["stream"].(bool); ok {
		genPayload.Stream = stream
	}

	return genPayload
}

func (rs *RemoteScheduler) mapToEmbedPayload(payload map[string]any) models.EmbedTaskPayload {
	embedPayload := models.EmbedTaskPayload{}

	if input, ok := payload["input"]; ok {
		embedPayload.Input = input
	}

	if truncate, ok := payload["truncate"].(bool); ok {
		embedPayload.Truncate = truncate
	}

	return embedPayload
}
