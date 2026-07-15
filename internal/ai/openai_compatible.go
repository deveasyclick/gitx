package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// chatMessage represents a message in the OpenAI-compatible chat API.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest is the request body for /v1/chat/completions.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// chatResponse is the response body from /v1/chat/completions.
type chatResponse struct {
	ID      string        `json:"id"`
	Choices []chatChoice  `json:"choices"`
	Usage   *chatUsage    `json:"usage,omitempty"`
	Error   *chatAPIError `json:"error,omitempty"`
}

type chatChoice struct {
	Index   int         `json:"index"`
	Message chatMessage `json:"message"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// openAICompatibleClient handles HTTP communication with OpenAI-compatible APIs.
type openAICompatibleClient struct {
	baseURL    string
	apiPath    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type openAICompatibleConfig struct {
	baseURL string
	apiPath string // API path, e.g. "/v1/chat/completions" or "/chat/completions"
	apiKey  string
	model   string
}

func newOpenAICompatibleClient(cfg openAICompatibleConfig) *openAICompatibleClient {
	apiPath := cfg.apiPath
	if apiPath == "" {
		apiPath = "/v1/chat/completions" // default for OpenAI
	}
	return &openAICompatibleClient{
		baseURL: cfg.baseURL,
		apiPath: apiPath,
		apiKey:  cfg.apiKey,
		model:   cfg.model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *openAICompatibleClient) Name() string {
	return c.model // overridden by wrapping provider
}

func (c *openAICompatibleClient) Generate(ctx context.Context, req Request) (Response, error) {
	body := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
		Temperature: 0.3,
	}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.apiPath, bytes.NewReader(payload))
	if err != nil {
		return Response{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return Response{}, fmt.Errorf("parse response: %w", err)
	}

	if chatResp.Error != nil {
		return Response{}, fmt.Errorf("API error: %s (%s)", chatResp.Error.Message, chatResp.Error.Type)
	}

	if len(chatResp.Choices) == 0 {
		return Response{}, fmt.Errorf("no choices in response")
	}

	usage := TokenUsage{}
	if chatResp.Usage != nil {
		usage = TokenUsage{
			InputTokens:  chatResp.Usage.PromptTokens,
			OutputTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:  chatResp.Usage.TotalTokens,
		}
	}

	return Response{
		Text:  chatResp.Choices[0].Message.Content,
		Usage: usage,
	}, nil
}
