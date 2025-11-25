package llm

import "time"

type Client interface {
	Chat(messages []*ChatMessage, options *ChatOptions) (*ChatResponse, error)
	ChatStream(messages []*ChatMessage, options *ChatOptions, handler func(*ChatResponse) error) error
	FunctionCall(messages []*ChatMessage, tools []*ToolDefinition) (*FunctionCallResponse, error)
	GetModelInfo() *ModelInfo
}

type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatOptions struct {
	Temperature float64
	MaxTokens   int
	TopP        float64
}

type ChatResponse struct {
	Message      *ChatMessage
	Usage        *Usage
	FinishReason string
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type FunctionCallResponse struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ModelInfo struct {
	Name         string
	Provider     string
	MaxTokens    int
	SupportsTool bool
}
