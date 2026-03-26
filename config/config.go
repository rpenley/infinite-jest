package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all runtime configuration for an infinite-jest session.
// Values are loaded from a JSON file and may be overridden by CLI flags.
type Config struct {
	Backend       string   `json:"backend"`
	Model         string   `json:"model"`
	URL           string   `json:"url"`
	Rounds        int      `json:"rounds"`
	MaxHistory    int      `json:"max_history"`
	Seed          string   `json:"seed"`
	OutputFile    string   `json:"output_file"`
	Personas      []string `json:"personas"`
	LogFile       string   `json:"log_file"`
}

// Defaults returns a Config with sensible starting values: ollama backend,
// localhost Ollama URL, 20-turn history window, and 1 debate round.
func Defaults() Config {
	return Config{
		Backend:    "ollama",
		URL:        "http://localhost:11434",
		MaxHistory: 20,
		Rounds:     1,
	}
}

// DefaultPath returns the platform-appropriate config file path:
// ~/.config/infinite-jest/config.json, with a fallback to the working directory.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "infinite-jest.json"
	}
	return filepath.Join(home, ".config", "infinite-jest", "config.json")
}

// Load reads a JSON config file at path, merging it into Defaults.
// A missing file is not an error — callers receive the defaults unchanged.
func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
