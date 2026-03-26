package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	ollama "github.com/ollama/ollama/api"
	openai "github.com/sashabaranov/go-openai"
)

// ErrContextExceeded is returned by Complete when the prompt exceeds the
// model's context window.
var ErrContextExceeded = errors.New("context window exceeded")

// Client routes LLM requests to either an Ollama or OpenAI backend.
type Client struct {
	backend string
	model   string
	ollama  *ollama.Client
	openai  *openai.Client
}

// New returns a Client for the given backend ("ollama" or "openai"),
// model name, and server base URL.
func New(backend, model, url string) (*Client, error) {
	c := &Client{backend: backend, model: model}
	switch backend {
	case "ollama":
		base, err := parseURL(url)
		if err != nil {
			return nil, fmt.Errorf("invalid url: %w", err)
		}
		c.ollama = ollama.NewClient(base, http.DefaultClient)
	case "openai":
		config := openai.DefaultConfig("none")
		config.BaseURL = url
		c.openai = openai.NewClientWithConfig(config)
	default:
		return nil, fmt.Errorf("unknown backend %q", backend)
	}
	return c, nil
}

// Complete sends messages to the LLM and returns the assistant response.
// Returns ErrContextExceeded if the context window is exceeded.
func (c *Client) Complete(ctx context.Context, messages []Message) (string, error) {
	switch c.backend {
	case "ollama":
		return c.completeOllama(ctx, messages)
	case "openai":
		return c.completeOpenAI(ctx, messages)
	default:
		return "", fmt.Errorf("unknown backend %q", c.backend)
	}
}

func (c *Client) completeOllama(ctx context.Context, messages []Message) (string, error) {
	ollamaMessages := make([]ollama.Message, len(messages))
	for i, m := range messages {
		ollamaMessages[i] = ollama.Message{Role: m.Role, Content: m.Content}
	}

	req := &ollama.ChatRequest{
		Model:    c.model,
		Messages: ollamaMessages,
	}

	var response string
	err := c.ollama.Chat(ctx, req, func(resp ollama.ChatResponse) error {
		response += resp.Message.Content
		return nil
	})
	if err != nil {
		// Ollama returns a plain-string error containing "context length exceeded"
		// when the prompt is too large for the model's context window.
		if strings.Contains(err.Error(), "context length exceeded") {
			return "", ErrContextExceeded
		}
		return "", fmt.Errorf("ollama chat: %w", err)
	}
	return response, nil
}

func (c *Client) completeOpenAI(ctx context.Context, messages []Message) (string, error) {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{Role: m.Role, Content: m.Content}
	}

	resp, err := c.openai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: openaiMessages,
	})
	if err != nil {
		var apiErr *openai.APIError
		if errors.As(err, &apiErr) && strings.Contains(apiErr.Message, "context_length_exceeded") {
			return "", ErrContextExceeded
		}
		return "", fmt.Errorf("openai chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	if resp.Choices[0].FinishReason == openai.FinishReasonLength {
		return resp.Choices[0].Message.Content, ErrContextExceeded
	}
	return resp.Choices[0].Message.Content, nil
}
