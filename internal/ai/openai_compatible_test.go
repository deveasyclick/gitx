package ai_test

import (
	"encoding/json"
	"testing"
)

func TestChatResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantText string
		wantOK   bool
	}{
		{
			name: "standard response",
			json: `{
				"id": "cmpl-1",
				"choices": [{"index": 0, "message": {"role": "assistant", "content": "hello"}}],
				"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
			}`,
			wantText: "hello",
			wantOK:   true,
		},
		{
			name: "multiple choices picks first",
			json: `{
				"id": "cmpl-2",
				"choices": [
					{"index": 0, "message": {"role": "assistant", "content": "first"}},
					{"index": 1, "message": {"role": "assistant", "content": "second"}}
				]
			}`,
			wantText: "first",
			wantOK:   true,
		},
		{
			name: "usage optional",
			json: `{
				"id": "cmpl-3",
				"choices": [{"index": 0, "message": {"role": "assistant", "content": "no usage data"}}]
			}`,
			wantText: "no usage data",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(tt.json), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(resp.Choices) == 0 {
				t.Fatal("no choices")
			}
			got := resp.Choices[0].Message.Content
			if got != tt.wantText {
				t.Errorf("content = %q, want %q", got, tt.wantText)
			}
		})
	}
}

func TestChatRequestMarshal(t *testing.T) {
	body := map[string]interface{}{
		"model": "gpt-5-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "you are helpful"},
			{"role": "user", "content": "hello"},
		},
		"temperature": 0.3,
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded["model"] != "gpt-5-mini" {
		t.Errorf("model = %v", decoded["model"])
	}
}
