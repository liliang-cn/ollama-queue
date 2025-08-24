package executor

import (
	"context"
	"fmt"

	"github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/ollama-queue/pkg/models"
)

// OllamaPlugin implements the GenericExecutor interface for Ollama
type OllamaPlugin struct {
	client   *ollama.Client
	metadata *ExecutorMetadata
	config   *models.Config
}

// NewOllamaPlugin creates a new Ollama executor plugin
func NewOllamaPlugin(config *models.Config) (*OllamaPlugin, error) {
	if config == nil {
		config = models.DefaultConfig()
	}
	
	clientOptions := []ollama.ClientOption{
		ollama.WithHost(config.OllamaHost),
	}
	
	client, err := ollama.NewClient(clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}
	
	plugin := &OllamaPlugin{
		client: client,
		config: config,
		metadata: &ExecutorMetadata{
			Name:        "Ollama Executor",
			Type:        models.ExecutorTypeOllama,
			Description: "Executes tasks using Ollama language models",
			Version:     "1.0.0",
			Actions: []ExecutorAction{
				models.ActionChat,
				models.ActionGenerate,
				models.ActionEmbed,
			},
			Config: map[string]any{
				"host":    config.OllamaHost,
				"timeout": config.OllamaTimeout.String(),
			},
		},
	}
	
	return plugin, nil
}

// GetMetadata returns information about this executor
func (p *OllamaPlugin) GetMetadata() *ExecutorMetadata {
	return p.metadata
}

// CanExecute checks if the executor can handle a specific action
func (p *OllamaPlugin) CanExecute(action ExecutorAction) bool {
	switch action {
	case models.ActionChat, models.ActionGenerate, models.ActionEmbed:
		return true
	default:
		return false
	}
}

// Execute executes a task synchronously
func (p *OllamaPlugin) Execute(ctx context.Context, task *GenericTask) (*TaskResult, error) {
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.OllamaTimeout)
	defer cancel()
	
	result := &TaskResult{
		TaskID:  task.ID,
		Success: false,
	}
	
	switch task.Action {
	case models.ActionChat:
		data, err := p.executeChat(timeoutCtx, task)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Data = data
		result.Success = true
		
	case models.ActionGenerate:
		data, err := p.executeGenerate(timeoutCtx, task)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Data = data
		result.Success = true
		
	case models.ActionEmbed:
		data, err := p.executeEmbed(timeoutCtx, task)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Data = data
		result.Success = true
		
	default:
		err := fmt.Errorf("unsupported action: %s", task.Action)
		result.Error = err.Error()
		return result, err
	}
	
	return result, nil
}

// ExecuteStream executes a task with streaming output
func (p *OllamaPlugin) ExecuteStream(ctx context.Context, task *GenericTask) (<-chan *StreamChunk, error) {
	outputChan := make(chan *StreamChunk, 10)
	
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.OllamaTimeout)
	
	go func() {
		defer close(outputChan)
		defer cancel()
		
		switch task.Action {
		case models.ActionChat:
			p.executeChatStream(timeoutCtx, task, outputChan)
		case models.ActionGenerate:
			p.executeGenerateStream(timeoutCtx, task, outputChan)
		default:
			outputChan <- &StreamChunk{
				Error: fmt.Errorf("streaming not supported for action: %s", task.Action),
				Done:  true,
			}
		}
	}()
	
	return outputChan, nil
}

// Validate validates that a task payload is correct for this executor
func (p *OllamaPlugin) Validate(action ExecutorAction, payload map[string]any) error {
	switch action {
	case models.ActionChat:
		return p.validateChatPayload(payload)
	case models.ActionGenerate:
		return p.validateGeneratePayload(payload)
	case models.ActionEmbed:
		return p.validateEmbedPayload(payload)
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}
}

// Close cleans up executor resources
func (p *OllamaPlugin) Close() error {
	// Ollama client doesn't need explicit cleanup
	return nil
}

// Private execution methods

func (p *OllamaPlugin) executeChat(ctx context.Context, task *GenericTask) (any, error) {
	messages, err := p.extractMessages(task.Payload)
	if err != nil {
		return nil, err
	}
	
	// Prepare options
	var options []func(*ollama.ChatRequest)
	if system, ok := task.Payload["system"].(string); ok && system != "" {
		options = append(options, ollama.WithChatSystem(system))
	}
	
	// Execute chat
	response, err := ollama.Chat(ctx, task.Model, messages, options...)
	if err != nil {
		return nil, fmt.Errorf("chat execution failed: %w", err)
	}
	
	return response, nil
}

func (p *OllamaPlugin) executeGenerate(ctx context.Context, task *GenericTask) (any, error) {
	prompt, ok := task.Payload["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid prompt in payload")
	}
	
	// Prepare options
	var options []func(*ollama.GenerateRequest)
	if system, ok := task.Payload["system"].(string); ok && system != "" {
		options = append(options, ollama.WithSystem(system))
	}
	if template, ok := task.Payload["template"].(string); ok && template != "" {
		options = append(options, ollama.WithGenerateTemplate(template))
	}
	if raw, ok := task.Payload["raw"].(bool); ok && raw {
		options = append(options, ollama.WithRaw())
	}
	
	// Execute generation
	response, err := ollama.Generate(ctx, task.Model, prompt, options...)
	if err != nil {
		return nil, fmt.Errorf("generate execution failed: %w", err)
	}
	
	return response, nil
}

func (p *OllamaPlugin) executeEmbed(ctx context.Context, task *GenericTask) (any, error) {
	input, ok := task.Payload["input"]
	if !ok {
		return nil, fmt.Errorf("missing input in payload")
	}
	
	// Prepare options
	var options []func(*ollama.EmbedRequest)
	if truncate, ok := task.Payload["truncate"].(bool); ok && truncate {
		options = append(options, ollama.WithTruncate(truncate))
	}
	
	// Execute embedding
	response, err := ollama.Embed(ctx, task.Model, input, options...)
	if err != nil {
		return nil, fmt.Errorf("embed execution failed: %w", err)
	}
	
	return response, nil
}

// Streaming execution methods

func (p *OllamaPlugin) executeChatStream(ctx context.Context, task *GenericTask, outputChan chan<- *StreamChunk) {
	messages, err := p.extractMessages(task.Payload)
	if err != nil {
		outputChan <- &StreamChunk{
			Error: err,
			Done:  true,
		}
		return
	}
	
	// Prepare options
	var options []func(*ollama.ChatRequest)
	if system, ok := task.Payload["system"].(string); ok && system != "" {
		options = append(options, ollama.WithChatSystem(system))
	}
	
	// Execute streaming chat
	responseChan, errorChan := ollama.ChatStream(ctx, task.Model, messages, options...)
	
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				outputChan <- &StreamChunk{Done: true}
				return
			}
			
			if response.Message.Content != "" {
				outputChan <- &StreamChunk{Data: response.Message.Content}
			}
			
			if response.Done {
				outputChan <- &StreamChunk{Done: true}
				return
			}
			
		case err, ok := <-errorChan:
			if !ok {
				outputChan <- &StreamChunk{Done: true}
				return
			}
			
			outputChan <- &StreamChunk{
				Error: err,
				Done:  true,
			}
			return
			
		case <-ctx.Done():
			outputChan <- &StreamChunk{
				Error: ctx.Err(),
				Done:  true,
			}
			return
		}
	}
}

func (p *OllamaPlugin) executeGenerateStream(ctx context.Context, task *GenericTask, outputChan chan<- *StreamChunk) {
	prompt, ok := task.Payload["prompt"].(string)
	if !ok {
		outputChan <- &StreamChunk{
			Error: fmt.Errorf("missing or invalid prompt in payload"),
			Done:  true,
		}
		return
	}
	
	// Prepare options
	var options []func(*ollama.GenerateRequest)
	if system, ok := task.Payload["system"].(string); ok && system != "" {
		options = append(options, ollama.WithSystem(system))
	}
	if template, ok := task.Payload["template"].(string); ok && template != "" {
		options = append(options, ollama.WithGenerateTemplate(template))
	}
	if raw, ok := task.Payload["raw"].(bool); ok && raw {
		options = append(options, ollama.WithRaw())
	}
	
	// Execute streaming generation
	responseChan, errorChan := ollama.GenerateStream(ctx, task.Model, prompt, options...)
	
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				outputChan <- &StreamChunk{Done: true}
				return
			}
			
			if response.Response != "" {
				outputChan <- &StreamChunk{Data: response.Response}
			}
			
			if response.Done {
				outputChan <- &StreamChunk{Done: true}
				return
			}
			
		case err, ok := <-errorChan:
			if !ok {
				outputChan <- &StreamChunk{Done: true}
				return
			}
			
			outputChan <- &StreamChunk{
				Error: err,
				Done:  true,
			}
			return
			
		case <-ctx.Done():
			outputChan <- &StreamChunk{
				Error: ctx.Err(),
				Done:  true,
			}
			return
		}
	}
}

// Helper methods for payload validation and extraction

func (p *OllamaPlugin) validateChatPayload(payload map[string]any) error {
	if _, ok := payload["messages"]; !ok {
		return fmt.Errorf("missing messages in chat payload")
	}
	
	_, err := p.extractMessages(payload)
	return err
}

func (p *OllamaPlugin) validateGeneratePayload(payload map[string]any) error {
	if _, ok := payload["prompt"].(string); !ok {
		return fmt.Errorf("missing or invalid prompt in generate payload")
	}
	return nil
}

func (p *OllamaPlugin) validateEmbedPayload(payload map[string]any) error {
	if _, ok := payload["input"]; !ok {
		return fmt.Errorf("missing input in embed payload")
	}
	return nil
}

func (p *OllamaPlugin) extractMessages(payload map[string]any) ([]ollama.Message, error) {
	messagesAny, ok := payload["messages"]
	if !ok {
		return nil, fmt.Errorf("missing messages in payload")
	}
	
	messagesSlice, ok := messagesAny.([]any)
	if !ok {
		// Try to handle []models.ChatMessage
		if chatMessages, ok := messagesAny.([]models.ChatMessage); ok {
			var messages []ollama.Message
			for _, msg := range chatMessages {
				messages = append(messages, ollama.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
			return messages, nil
		}
		return nil, fmt.Errorf("messages should be an array")
	}
	
	var messages []ollama.Message
	for _, msgAny := range messagesSlice {
		if msgMap, ok := msgAny.(map[string]any); ok {
			msg := ollama.Message{}
			if role, ok := msgMap["role"].(string); ok {
				msg.Role = role
			}
			if content, ok := msgMap["content"].(string); ok {
				msg.Content = content
			}
			messages = append(messages, msg)
		}
	}
	
	if len(messages) == 0 {
		return nil, fmt.Errorf("no valid messages found")
	}
	
	return messages, nil
}