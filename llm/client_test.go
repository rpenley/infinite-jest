package llm

import (
	"context"
	"testing"
)

const (
	testServerURL = "http://192.168.42.252:11434"
	testModel     = "qwen3.5:35b"
)

func TestNew_Ollama(t *testing.T) {
	client, err := New("ollama", testModel, testServerURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestComplete_ReturnsResponse(t *testing.T) {
	client, err := New("ollama", testModel, testServerURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "Reply with exactly the word: hello"},
	}

	response, err := client.Complete(context.Background(), messages)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if response == "" {
		t.Error("expected non-empty response")
	}
	t.Logf("response: %s", response)
}
