package executor

import (
	"context"
	"fmt"

	"github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/ollama-queue/internal/models"
)

// Executor interface defines task execution operations
type Executor interface {
	Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error)
	ExecuteStream(ctx context.Context, task *models.Task) (<-chan *models.StreamChunk, error)
	CanExecute(taskType models.TaskType) bool
}

// OllamaExecutor implements task execution using Ollama client
type OllamaExecutor struct {
	client *ollama.Client
	config *models.Config
}

// NewOllamaExecutor creates a new Ollama executor
func NewOllamaExecutor(config *models.Config) (*OllamaExecutor, error) {
	clientOptions := []ollama.ClientOption{
		ollama.WithHost(config.OllamaHost),
	}
	
	client, err := ollama.NewClient(clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}
	
	return &OllamaExecutor{
		client: client,
		config: config,
	}, nil
}

// Execute executes a task synchronously
func (e *OllamaExecutor) Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error) {
	task.ExecutedOn = "local"
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.OllamaTimeout)
	defer cancel()
	
	result := &models.TaskResult{
		TaskID:  task.ID,
		Success: false,
	}
	
	switch task.Type {
	case models.TaskTypeChat:
		data, err := e.executeChat(timeoutCtx, task)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Data = data
		result.Success = true
		
	case models.TaskTypeGenerate:
		data, err := e.executeGenerate(timeoutCtx, task)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Data = data
		result.Success = true
		
	case models.TaskTypeEmbed:
		data, err := e.executeEmbed(timeoutCtx, task)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Data = data
		result.Success = true
		
	default:
		err := fmt.Errorf("unsupported task type: %s", task.Type)
		result.Error = err.Error()
		return result, err
	}
	
	return result, nil
}

// ExecuteStream executes a task with streaming output
func (e *OllamaExecutor) ExecuteStream(ctx context.Context, task *models.Task) (<-chan *models.StreamChunk, error) {
	outputChan := make(chan *models.StreamChunk, 10)
	
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.OllamaTimeout)
	
	go func() {
		defer close(outputChan)
		defer cancel()
		
		switch task.Type {
		case models.TaskTypeChat:
			e.executeChatStream(timeoutCtx, task, outputChan)
		case models.TaskTypeGenerate:
			e.executeGenerateStream(timeoutCtx, task, outputChan)
		default:
			outputChan <- &models.StreamChunk{
				Error: fmt.Errorf("streaming not supported for task type: %s", task.Type),
				Done:  true,
			}
		}
	}()
	
	return outputChan, nil
}

// CanExecute checks if the executor can handle a specific task type
func (e *OllamaExecutor) CanExecute(taskType models.TaskType) bool {
	switch taskType {
	case models.TaskTypeChat, models.TaskTypeGenerate, models.TaskTypeEmbed:
		return true
	default:
		return false
	}
}

// Private methods for specific task types

func (e *OllamaExecutor) executeChat(ctx context.Context, task *models.Task) (any, error) {
	payload, ok := task.Payload.(map[string]any)
	if !ok {
		// Try to convert from ChatTaskPayload
		if chatPayload, ok := task.Payload.(models.ChatTaskPayload); ok {
			return e.executeChatWithPayload(ctx, task, &chatPayload)
		}
		return nil, fmt.Errorf("invalid chat task payload")
	}
	
	// Convert map to ChatTaskPayload
	chatPayload, err := e.mapToChatPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chat payload: %w", err)
	}
	
	return e.executeChatWithPayload(ctx, task, chatPayload)
}

func (e *OllamaExecutor) executeChatWithPayload(ctx context.Context, task *models.Task, payload *models.ChatTaskPayload) (any, error) {
	// Convert messages
	var messages []ollama.Message
	for _, msg := range payload.Messages {
		messages = append(messages, ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Prepare options
	var options []func(*ollama.ChatRequest)
	if payload.System != "" {
		options = append(options, ollama.WithChatSystem(payload.System))
	}
	
	// Execute chat
	response, err := ollama.Chat(ctx, task.Model, messages, options...)
	if err != nil {
		return nil, fmt.Errorf("chat execution failed: %w", err)
	}
	
	return response, nil
}

func (e *OllamaExecutor) executeGenerate(ctx context.Context, task *models.Task) (any, error) {
	payload, ok := task.Payload.(map[string]any)
	if !ok {
		// Try to convert from GenerateTaskPayload
		if genPayload, ok := task.Payload.(models.GenerateTaskPayload); ok {
			return e.executeGenerateWithPayload(ctx, task, &genPayload)
		}
		return nil, fmt.Errorf("invalid generate task payload")
	}
	
	// Convert map to GenerateTaskPayload
	genPayload, err := e.mapToGeneratePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generate payload: %w", err)
	}
	
	return e.executeGenerateWithPayload(ctx, task, genPayload)
}

func (e *OllamaExecutor) executeGenerateWithPayload(ctx context.Context, task *models.Task, payload *models.GenerateTaskPayload) (any, error) {
	// Prepare options
	var options []func(*ollama.GenerateRequest)
	if payload.System != "" {
		options = append(options, ollama.WithSystem(payload.System))
	}
	if payload.Template != "" {
		options = append(options, ollama.WithGenerateTemplate(payload.Template))
	}
	if payload.Raw {
		options = append(options, ollama.WithRaw())
	}
	
	// Execute generation
	response, err := ollama.Generate(ctx, task.Model, payload.Prompt, options...)
	if err != nil {
		return nil, fmt.Errorf("generate execution failed: %w", err)
	}
	
	return response, nil
}

func (e *OllamaExecutor) executeEmbed(ctx context.Context, task *models.Task) (any, error) {
	payload, ok := task.Payload.(map[string]any)
	if !ok {
		// Try to convert from EmbedTaskPayload
		if embedPayload, ok := task.Payload.(models.EmbedTaskPayload); ok {
			return e.executeEmbedWithPayload(ctx, task, &embedPayload)
		}
		return nil, fmt.Errorf("invalid embed task payload")
	}
	
	// Convert map to EmbedTaskPayload
	embedPayload, err := e.mapToEmbedPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embed payload: %w", err)
	}
	
	return e.executeEmbedWithPayload(ctx, task, embedPayload)
}

func (e *OllamaExecutor) executeEmbedWithPayload(ctx context.Context, task *models.Task, payload *models.EmbedTaskPayload) (any, error) {
	// Prepare options
	var options []func(*ollama.EmbedRequest)
	if payload.Truncate {
		options = append(options, ollama.WithTruncate(payload.Truncate))
	}
	
	// Execute embedding
	response, err := ollama.Embed(ctx, task.Model, payload.Input, options...)
	if err != nil {
		return nil, fmt.Errorf("embed execution failed: %w", err)
	}
	
	return response, nil
}

// Streaming execution methods

func (e *OllamaExecutor) executeChatStream(ctx context.Context, task *models.Task, outputChan chan<- *models.StreamChunk) {
	payload, ok := task.Payload.(map[string]any)
	if !ok {
		if chatPayload, ok := task.Payload.(models.ChatTaskPayload); ok {
			e.executeChatStreamWithPayload(ctx, task, &chatPayload, outputChan)
			return
		}
		outputChan <- &models.StreamChunk{
			Error: fmt.Errorf("invalid chat task payload"),
			Done:  true,
		}
		return
	}
	
	chatPayload, err := e.mapToChatPayload(payload)
	if err != nil {
		outputChan <- &models.StreamChunk{
			Error: fmt.Errorf("failed to parse chat payload: %w", err),
			Done:  true,
		}
		return
	}
	
	e.executeChatStreamWithPayload(ctx, task, chatPayload, outputChan)
}

func (e *OllamaExecutor) executeChatStreamWithPayload(ctx context.Context, task *models.Task, payload *models.ChatTaskPayload, outputChan chan<- *models.StreamChunk) {
	// Convert messages
	var messages []ollama.Message
	for _, msg := range payload.Messages {
		messages = append(messages, ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Prepare options
	var options []func(*ollama.ChatRequest)
	if payload.System != "" {
		options = append(options, ollama.WithChatSystem(payload.System))
	}
	
	// Execute streaming chat
	responseChan, errorChan := ollama.ChatStream(ctx, task.Model, messages, options...)
	
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				outputChan <- &models.StreamChunk{Done: true}
				return
			}
			
			if response.Message.Content != "" {
				outputChan <- &models.StreamChunk{Data: response.Message.Content}
			}
			
			if response.Done {
				outputChan <- &models.StreamChunk{Done: true}
				return
			}
			
		case err, ok := <-errorChan:
			if !ok {
				outputChan <- &models.StreamChunk{Done: true}
				return
			}
			
			outputChan <- &models.StreamChunk{
				Error: err,
				Done:  true,
			}
			return
			
		case <-ctx.Done():
			outputChan <- &models.StreamChunk{
				Error: ctx.Err(),
				Done:  true,
			}
			return
		}
	}
}

func (e *OllamaExecutor) executeGenerateStream(ctx context.Context, task *models.Task, outputChan chan<- *models.StreamChunk) {
	payload, ok := task.Payload.(map[string]any)
	if !ok {
		if genPayload, ok := task.Payload.(models.GenerateTaskPayload); ok {
			e.executeGenerateStreamWithPayload(ctx, task, &genPayload, outputChan)
			return
		}
		outputChan <- &models.StreamChunk{
			Error: fmt.Errorf("invalid generate task payload"),
			Done:  true,
		}
		return
	}
	
	genPayload, err := e.mapToGeneratePayload(payload)
	if err != nil {
		outputChan <- &models.StreamChunk{
			Error: fmt.Errorf("failed to parse generate payload: %w", err),
			Done:  true,
		}
		return
	}
	
	e.executeGenerateStreamWithPayload(ctx, task, genPayload, outputChan)
}

func (e *OllamaExecutor) executeGenerateStreamWithPayload(ctx context.Context, task *models.Task, payload *models.GenerateTaskPayload, outputChan chan<- *models.StreamChunk) {
	// Prepare options
	var options []func(*ollama.GenerateRequest)
	if payload.System != "" {
		options = append(options, ollama.WithSystem(payload.System))
	}
	if payload.Template != "" {
		options = append(options, ollama.WithGenerateTemplate(payload.Template))
	}
	if payload.Raw {
		options = append(options, ollama.WithRaw())
	}
	
	// Execute streaming generation
	responseChan, errorChan := ollama.GenerateStream(ctx, task.Model, payload.Prompt, options...)
	
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				outputChan <- &models.StreamChunk{Done: true}
				return
			}
			
			if response.Response != "" {
				outputChan <- &models.StreamChunk{Data: response.Response}
			}
			
			if response.Done {
				outputChan <- &models.StreamChunk{Done: true}
				return
			}
			
		case err, ok := <-errorChan:
			if !ok {
				outputChan <- &models.StreamChunk{Done: true}
				return
			}
			
			outputChan <- &models.StreamChunk{
				Error: err,
				Done:  true,
			}
			return
			
		case <-ctx.Done():
			outputChan <- &models.StreamChunk{
				Error: ctx.Err(),
				Done:  true,
			}
			return
		}
	}
}

// Helper methods for payload conversion

func (e *OllamaExecutor) mapToChatPayload(payload map[string]any) (*models.ChatTaskPayload, error) {
	chatPayload := &models.ChatTaskPayload{}
	
	// Extract messages
	if messagesAny, ok := payload["messages"]; ok {
		if messagesSlice, ok := messagesAny.([]any); ok {
			for _, msgAny := range messagesSlice {
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
	}
	
	// Extract system
	if system, ok := payload["system"].(string); ok {
		chatPayload.System = system
	}
	
	// Extract stream
	if stream, ok := payload["stream"].(bool); ok {
		chatPayload.Stream = stream
	}
	
	return chatPayload, nil
}

func (e *OllamaExecutor) mapToGeneratePayload(payload map[string]any) (*models.GenerateTaskPayload, error) {
	genPayload := &models.GenerateTaskPayload{}
	
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
	
	return genPayload, nil
}

func (e *OllamaExecutor) mapToEmbedPayload(payload map[string]any) (*models.EmbedTaskPayload, error) {
	embedPayload := &models.EmbedTaskPayload{}
	
	if input, ok := payload["input"]; ok {
		embedPayload.Input = input
	}
	
	if truncate, ok := payload["truncate"].(bool); ok {
		embedPayload.Truncate = truncate
	}
	
	return embedPayload, nil
}