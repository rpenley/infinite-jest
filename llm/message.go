package llm

import "net/url"

// Message is a single role/content pair in an LLM conversation.
type Message struct {
	Role    string
	Content string
}

func parseURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}
