package models

// ChatMessage represents a single message in a chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatTaskPayload represents the payload for chat tasks
type ChatTaskPayload struct {
	Messages []ChatMessage `json:"messages"`
	System   string        `json:"system,omitempty"`
	Stream   bool          `json:"stream,omitempty"`
}

// GenerateTaskPayload represents the payload for generation tasks
type GenerateTaskPayload struct {
	Prompt   string `json:"prompt"`
	System   string `json:"system,omitempty"`
	Template string `json:"template,omitempty"`
	Raw      bool   `json:"raw,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
}

// EmbedTaskPayload represents the payload for embedding tasks
type EmbedTaskPayload struct {
	Input    any  `json:"input"`
	Truncate bool `json:"truncate,omitempty"`
}

// TaskCallback is a function type for handling task completion
type TaskCallback func(result *TaskResult)

// BatchCallback is a function type for handling batch task completion
type BatchCallback func(results []*TaskResult)