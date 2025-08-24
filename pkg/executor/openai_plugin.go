package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	
	"github.com/liliang-cn/ollama-queue/pkg/models"
)

// OpenAIPlugin implements the GenericExecutor interface for OpenAI API
type OpenAIPlugin struct {
	client   *http.Client
	metadata *ExecutorMetadata
	apiKey   string
	baseURL  string
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"` // defaults to https://api.openai.com/v1
	Timeout time.Duration `json:"timeout"`
}

// NewOpenAIPlugin creates a new OpenAI executor plugin
func NewOpenAIPlugin(config OpenAIConfig) (*OpenAIPlugin, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	plugin := &OpenAIPlugin{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		apiKey:  config.APIKey,
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		metadata: &ExecutorMetadata{
			Name:        "OpenAI Executor",
			Type:        models.ExecutorTypeOpenAI,
			Description: "Executes tasks using OpenAI API",
			Version:     "1.0.0",
			Actions: []ExecutorAction{
				models.ActionChat,
				models.ActionGenerate,
			},
			Config: map[string]any{
				"base_url": config.BaseURL,
				"timeout":  config.Timeout.String(),
			},
		},
	}
	
	return plugin, nil
}

// GetMetadata returns information about this executor
func (p *OpenAIPlugin) GetMetadata() *ExecutorMetadata {
	return p.metadata
}

// CanExecute checks if the executor can handle a specific action
func (p *OpenAIPlugin) CanExecute(action ExecutorAction) bool {
	switch action {
	case models.ActionChat, models.ActionGenerate:
		return true
	default:
		return false
	}
}

// Execute executes a task synchronously
func (p *OpenAIPlugin) Execute(ctx context.Context, task *GenericTask) (*TaskResult, error) {
	result := &TaskResult{
		TaskID:  task.ID,
		Success: false,
	}
	
	switch task.Action {
	case models.ActionChat:
		data, err := p.executeChat(ctx, task)
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Data = data
		result.Success = true
		
	case models.ActionGenerate:
		data, err := p.executeGenerate(ctx, task)
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
func (p *OpenAIPlugin) ExecuteStream(ctx context.Context, task *GenericTask) (<-chan *StreamChunk, error) {
	outputChan := make(chan *StreamChunk, 10)
	
	go func() {
		defer close(outputChan)
		
		switch task.Action {
		case models.ActionChat:
			p.executeChatStream(ctx, task, outputChan)
		case models.ActionGenerate:
			p.executeGenerateStream(ctx, task, outputChan)
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
func (p *OpenAIPlugin) Validate(action ExecutorAction, payload map[string]any) error {
	switch action {
	case models.ActionChat:
		return p.validateChatPayload(payload)
	case models.ActionGenerate:
		return p.validateGeneratePayload(payload)
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}
}

// Close cleans up executor resources
func (p *OpenAIPlugin) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// OpenAI API structures
type OpenAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenAIChatMessage `json:"messages"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Choices []OpenAIChatChoice    `json:"choices"`
	Usage   *OpenAIUsage          `json:"usage,omitempty"`
}

type OpenAIChatChoice struct {
	Index   int               `json:"index"`
	Message OpenAIChatMessage `json:"message"`
	Delta   *OpenAIChatMessage `json:"delta,omitempty"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Private execution methods
func (p *OpenAIPlugin) executeChat(ctx context.Context, task *GenericTask) (any, error) {
	messages, err := p.extractOpenAIMessages(task.Payload)
	if err != nil {
		return nil, err
	}
	
	model := task.Model
	if model == "" {
		model = "gpt-3.5-turbo" // Default model
	}
	
	reqBody := OpenAIChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}
	
	// Add optional parameters
	if temp, ok := task.Payload["temperature"].(float64); ok {
		reqBody.Temperature = &temp
	}
	if maxTokens, ok := task.Payload["max_tokens"].(float64); ok {
		mt := int(maxTokens)
		reqBody.MaxTokens = &mt
	}
	
	return p.makeOpenAIRequest(ctx, "/chat/completions", reqBody)
}

func (p *OpenAIPlugin) executeGenerate(ctx context.Context, task *GenericTask) (any, error) {
	// For generate, convert to chat format
	prompt, ok := task.Payload["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid prompt in payload")
	}
	
	messages := []OpenAIChatMessage{
		{Role: "user", Content: prompt},
	}
	
	// Add system message if present
	if system, ok := task.Payload["system"].(string); ok && system != "" {
		messages = append([]OpenAIChatMessage{{Role: "system", Content: system}}, messages...)
	}
	
	model := task.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	
	reqBody := OpenAIChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}
	
	return p.makeOpenAIRequest(ctx, "/chat/completions", reqBody)
}

func (p *OpenAIPlugin) executeChatStream(ctx context.Context, task *GenericTask, outputChan chan<- *StreamChunk) {
	messages, err := p.extractOpenAIMessages(task.Payload)
	if err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	
	model := task.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	
	reqBody := OpenAIChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}
	
	p.streamOpenAIRequest(ctx, "/chat/completions", reqBody, outputChan)
}

func (p *OpenAIPlugin) executeGenerateStream(ctx context.Context, task *GenericTask, outputChan chan<- *StreamChunk) {
	prompt, ok := task.Payload["prompt"].(string)
	if !ok {
		outputChan <- &StreamChunk{Error: fmt.Errorf("missing or invalid prompt"), Done: true}
		return
	}
	
	messages := []OpenAIChatMessage{
		{Role: "user", Content: prompt},
	}
	
	if system, ok := task.Payload["system"].(string); ok && system != "" {
		messages = append([]OpenAIChatMessage{{Role: "system", Content: system}}, messages...)
	}
	
	model := task.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	
	reqBody := OpenAIChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}
	
	p.streamOpenAIRequest(ctx, "/chat/completions", reqBody, outputChan)
}

// Helper methods
func (p *OpenAIPlugin) makeOpenAIRequest(ctx context.Context, endpoint string, reqBody any) (any, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
	}
	
	var response OpenAIChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return response, nil
}

func (p *OpenAIPlugin) streamOpenAIRequest(ctx context.Context, endpoint string, reqBody any, outputChan chan<- *StreamChunk) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	resp, err := p.client.Do(req)
	if err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		outputChan <- &StreamChunk{
			Error: fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body)),
			Done: true,
		}
		return
	}
	
	// Parse Server-Sent Events
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				outputChan <- &StreamChunk{Done: true}
				return
			}
			
			var streamResp OpenAIChatResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err == nil {
				if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta != nil {
					if streamResp.Choices[0].Delta.Content != "" {
						outputChan <- &StreamChunk{Data: streamResp.Choices[0].Delta.Content}
					}
				}
			}
		}
		
		select {
		case <-ctx.Done():
			outputChan <- &StreamChunk{Error: ctx.Err(), Done: true}
			return
		default:
		}
	}
	
	if err := scanner.Err(); err != nil {
		outputChan <- &StreamChunk{Error: err, Done: true}
	}
}

func (p *OpenAIPlugin) validateChatPayload(payload map[string]any) error {
	if _, ok := payload["messages"]; !ok {
		return fmt.Errorf("missing messages in chat payload")
	}
	
	_, err := p.extractOpenAIMessages(payload)
	return err
}

func (p *OpenAIPlugin) validateGeneratePayload(payload map[string]any) error {
	if _, ok := payload["prompt"].(string); !ok {
		return fmt.Errorf("missing or invalid prompt in generate payload")
	}
	return nil
}

func (p *OpenAIPlugin) extractOpenAIMessages(payload map[string]any) ([]OpenAIChatMessage, error) {
	messagesAny, ok := payload["messages"]
	if !ok {
		return nil, fmt.Errorf("missing messages in payload")
	}
	
	messagesSlice, ok := messagesAny.([]any)
	if !ok {
		// Handle []models.ChatMessage
		if chatMessages, ok := messagesAny.([]models.ChatMessage); ok {
			var messages []OpenAIChatMessage
			for _, msg := range chatMessages {
				messages = append(messages, OpenAIChatMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
			return messages, nil
		}
		return nil, fmt.Errorf("messages should be an array")
	}
	
	var messages []OpenAIChatMessage
	for _, msgAny := range messagesSlice {
		if msgMap, ok := msgAny.(map[string]any); ok {
			msg := OpenAIChatMessage{}
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