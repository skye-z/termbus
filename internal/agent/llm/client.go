package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type AnthropicClient struct {
	apiKey string
	model  string
	client *http.Client
}

type OllamaClient struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOpenAI(apiKey, model, baseURL string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *OpenAIClient) Chat(messages []*ChatMessage, options *ChatOptions) (*ChatResponse, error) {
	if options == nil {
		options = &ChatOptions{Temperature: 0.7, MaxTokens: 4096}
	}

	reqBody := map[string]interface{}{
		"model":       c.model,
		"messages":    convertToOpenAIMessages(messages),
		"temperature": options.Temperature,
		"max_tokens":  options.MaxTokens,
		"top_p":       options.TopP,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	return &ChatResponse{
		Message: &ChatMessage{
			Role:    "assistant",
			Content: result.Choices[0].Message.Content,
		},
		Usage: &Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
		FinishReason: result.Choices[0].FinishReason,
	}, nil
}

func (c *OpenAIClient) ChatStream(messages []*ChatMessage, options *ChatOptions, handler func(*ChatResponse) error) error {
	return fmt.Errorf("streaming not implemented")
}

func (c *OpenAIClient) FunctionCall(messages []*ChatMessage, tools []*ToolDefinition) (*FunctionCallResponse, error) {
	return nil, fmt.Errorf("function calling not implemented")
}

func (c *OpenAIClient) GetModelInfo() *ModelInfo {
	return &ModelInfo{
		Name:         c.model,
		Provider:     "openai",
		MaxTokens:    128000,
		SupportsTool: true,
	}
}

func NewAnthropic(apiKey, model string) *AnthropicClient {
	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *AnthropicClient) Chat(messages []*ChatMessage, options *ChatOptions) (*ChatResponse, error) {
	if options == nil {
		options = &ChatOptions{Temperature: 0.7, MaxTokens: 4096}
	}

	reqBody := map[string]interface{}{
		"model":      c.model,
		"messages":   convertToAnthropicMessages(messages),
		"max_tokens": options.MaxTokens,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	content := ""
	for _, c := range result.Content {
		content += c.Text
	}

	return &ChatResponse{
		Message: &ChatMessage{Role: "assistant", Content: content},
		Usage:   &Usage{PromptTokens: result.Usage.InputTokens, CompletionTokens: result.Usage.OutputTokens},
	}, nil
}

func (c *AnthropicClient) ChatStream(messages []*ChatMessage, options *ChatOptions, handler func(*ChatResponse) error) error {
	return fmt.Errorf("streaming not implemented")
}

func (c *AnthropicClient) FunctionCall(messages []*ChatMessage, tools []*ToolDefinition) (*FunctionCallResponse, error) {
	return nil, fmt.Errorf("function calling not implemented")
}

func (c *AnthropicClient) GetModelInfo() *ModelInfo {
	return &ModelInfo{Name: c.model, Provider: "anthropic", MaxTokens: 200000, SupportsTool: true}
}

func NewOllama(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OllamaClient) Chat(messages []*ChatMessage, options *ChatOptions) (*ChatResponse, error) {
	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": convertToOllamaMessages(messages),
		"stream":   false,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Message: &ChatMessage{Role: "assistant", Content: result.Message.Content},
	}, nil
}

func (c *OllamaClient) ChatStream(messages []*ChatMessage, options *ChatOptions, handler func(*ChatResponse) error) error {
	return fmt.Errorf("streaming not implemented")
}

func (c *OllamaClient) FunctionCall(messages []*ChatMessage, tools []*ToolDefinition) (*FunctionCallResponse, error) {
	return nil, fmt.Errorf("function calling not supported")
}

func (c *OllamaClient) GetModelInfo() *ModelInfo {
	return &ModelInfo{Name: c.model, Provider: "ollama", MaxTokens: 8192, SupportsTool: false}
}

func convertToOpenAIMessages(messages []*ChatMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		result[i] = map[string]interface{}{"role": m.Role, "content": m.Content}
	}
	return result
}

func convertToAnthropicMessages(messages []*ChatMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		result[i] = map[string]interface{}{"role": m.Role, "content": m.Content}
	}
	return result
}

func convertToOllamaMessages(messages []*ChatMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		result[i] = map[string]interface{}{"role": m.Role, "content": m.Content}
	}
	return result
}
