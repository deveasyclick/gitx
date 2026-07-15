package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatibleClient_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/chat/completions") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test-key" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}

		// Decode and verify body
		var body chatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Model != "gpt-5-mini" {
			t.Errorf("model = %q", body.Model)
		}
		if len(body.Messages) != 2 {
			t.Fatalf("messages = %d", len(body.Messages))
		}
		if body.Messages[0].Role != "system" || body.Messages[0].Content != "sys prompt" {
			t.Errorf("system message = %+v", body.Messages[0])
		}
		if body.Messages[1].Role != "user" || body.Messages[1].Content != "user prompt" {
			t.Errorf("user message = %+v", body.Messages[1])
		}
		if body.MaxTokens != 0 {
			t.Errorf("max_tokens = %d, want 0 (unset)", body.MaxTokens)
		}
		if body.Temperature != 0.3 {
			t.Errorf("temperature = %f", body.Temperature)
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			ID: "cmpl-1",
			Choices: []chatChoice{
				{Index: 0, Message: chatMessage{Role: "assistant", Content: "feat: add login"}},
			},
			Usage: &chatUsage{
				PromptTokens:     50,
				CompletionTokens: 10,
				TotalTokens:      60,
			},
		})
	}))
	defer server.Close()

	client := newOpenAICompatibleClient(openAICompatibleConfig{
		baseURL: server.URL,
		apiKey:  "sk-test-key",
		model:   "gpt-5-mini",
	})

	resp, err := client.Generate(context.Background(), Request{
		SystemPrompt: "sys prompt",
		UserPrompt:   "user prompt",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if resp.Text != "feat: add login" {
		t.Errorf("Text = %q, want %q", resp.Text, "feat: add login")
	}
	if resp.Usage.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 10 {
		t.Errorf("OutputTokens = %d, want 10", resp.Usage.OutputTokens)
	}
	if resp.Usage.TotalTokens != 60 {
		t.Errorf("TotalTokens = %d, want 60", resp.Usage.TotalTokens)
	}
}

func TestOpenAICompatibleClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Incorrect API key", "type": "auth_error"}}`))
	}))
	defer server.Close()

	client := newOpenAICompatibleClient(openAICompatibleConfig{
		baseURL: server.URL,
		apiKey:  "sk-bad-key",
		model:   "gpt-5-mini",
	})

	_, err := client.Generate(context.Background(), Request{
		UserPrompt: "hello",
	})
	if err == nil {
		t.Fatal("expected error for API error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestOpenAICompatibleClient_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			ID:      "cmpl-2",
			Choices: []chatChoice{},
		})
	}))
	defer server.Close()

	client := newOpenAICompatibleClient(openAICompatibleConfig{
		baseURL: server.URL,
		apiKey:  "sk-test-key",
		model:   "gpt-5-mini",
	})

	_, err := client.Generate(context.Background(), Request{
		UserPrompt: "hello",
	})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestOpenAICompatibleClient_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't respond — let context cancel
		select {}
	}))
	defer server.Close()

	client := newOpenAICompatibleClient(openAICompatibleConfig{
		baseURL: server.URL,
		apiKey:  "sk-test-key",
		model:   "gpt-5-mini",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.Generate(ctx, Request{
		UserPrompt: "hello",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestOpenAICompatibleClient_CustomMaxTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.MaxTokens != 4096 {
			t.Errorf("max_tokens = %d, want 4096", body.MaxTokens)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []chatChoice{
				{Message: chatMessage{Role: "assistant", Content: "ok"}},
			},
		})
	}))
	defer server.Close()

	client := newOpenAICompatibleClient(openAICompatibleConfig{
		baseURL: server.URL,
		apiKey:  "sk-test-key",
		model:   "gpt-5-mini",
	})

	_, err := client.Generate(context.Background(), Request{
		UserPrompt: "hello",
		MaxTokens:  4096,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestNamedProvider(t *testing.T) {
	inner := &mockProvider{name: "inner", text: "hello"}
	p := &namedProvider{name: "openai", delegate: inner}

	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}

	resp, err := p.Generate(context.Background(), Request{UserPrompt: "test"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Text != "hello" {
		t.Errorf("Text = %q, want %q", resp.Text, "hello")
	}
}

// mockProvider for testing namedProvider.
type mockProvider struct {
	name string
	text string
	err  error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Generate(_ context.Context, _ Request) (Response, error) {
	if m.err != nil {
		return Response{}, m.err
	}
	return Response{Text: m.text}, nil
}
