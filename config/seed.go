package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// seedFrontmatter holds the YAML frontmatter fields recognized in seed files.
type seedFrontmatter struct {
	Question string   `yaml:"question"`
	Personas []string `yaml:"personas"`
}

// ParseSeedFile reads a seed file and returns the question, optional body
// text, and persona list. Files without YAML frontmatter (--- delimited)
// are returned as plain text in question with no body or personas.
func ParseSeedFile(path string) (question, body string, personas []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, fmt.Errorf("read seed file: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "---") {
		return strings.TrimSpace(content), "", nil, nil
	}

	rest := content[3:]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return strings.TrimSpace(content), "", nil, nil
	}

	frontmatterRaw := strings.TrimSpace(rest[:end])
	body = strings.TrimSpace(rest[end+4:])

	var fm seedFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatterRaw), &fm); err != nil {
		return "", "", nil, fmt.Errorf("parse seed file frontmatter: %w", err)
	}

	return fm.Question, body, fm.Personas, nil
}
