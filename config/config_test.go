package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Backend != "ollama" {
		t.Errorf("Backend: got %q, want %q", cfg.Backend, "ollama")
	}
	if cfg.URL != "http://localhost:11434" {
		t.Errorf("URL: got %q, want %q", cfg.URL, "http://localhost:11434")
	}
	if cfg.MaxHistory != 20 {
		t.Errorf("MaxHistory: got %d, want 20", cfg.MaxHistory)
	}
	if cfg.Rounds != 1 {
		t.Errorf("Rounds: got %d, want 1", cfg.Rounds)
	}
	if cfg.Model != "" {
		t.Errorf("Model should be empty by default, got %q", cfg.Model)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	defaults := Defaults()
	if cfg.Backend != defaults.Backend {
		t.Errorf("Backend: got %q, want %q", cfg.Backend, defaults.Backend)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data, _ := json.Marshal(Config{
		Backend:    "openai",
		Model:      "gpt-4",
		URL:        "http://example.com",
		Rounds:     5,
		MaxHistory: 10,
		Seed:       "hello",
		OutputFile: "/tmp/out.md",
	})
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Backend != "openai" {
		t.Errorf("Backend: got %q, want %q", cfg.Backend, "openai")
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Model: got %q, want %q", cfg.Model, "gpt-4")
	}
	if cfg.Rounds != 5 {
		t.Errorf("Rounds: got %d, want 5", cfg.Rounds)
	}
	if cfg.Seed != "hello" {
		t.Errorf("Seed: got %q, want %q", cfg.Seed, "hello")
	}
	if cfg.OutputFile != "/tmp/out.md" {
		t.Errorf("OutputFile: got %q, want %q", cfg.OutputFile, "/tmp/out.md")
	}
}
