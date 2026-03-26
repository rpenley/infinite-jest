package agent

import (
	"context"
	"infinite-jest/llm"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testServerURL = "http://192.168.42.252:11434"
	testModel     = "qwen3.5:35b"
)

func testPersonas() []Persona {
	return []Persona{catalog["pragmatist"], catalog["contrarian"]}
}

func TestRun_OneRound(t *testing.T) {
	client, err := llm.New("ollama", testModel, testServerURL)
	if err != nil {
		t.Fatalf("llm.New: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "session.md")
	a := New(client, 1, 20, "Is testing worth the effort?", outputPath, "", false, false, testPersonas())

	if err := a.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Question") {
		t.Error("output missing '# Question' header")
	}
	t.Logf("output.md:\n%s", content)

	thinkingData, err := os.ReadFile(thinkingPath(outputPath))
	if err != nil {
		t.Errorf("thinking file not created: %v", err)
	} else if len(thinkingData) == 0 {
		t.Error("thinking file is empty")
	}
}

func TestRun_PersonaRotation(t *testing.T) {
	client, err := llm.New("ollama", testModel, testServerURL)
	if err != nil {
		t.Fatalf("llm.New: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "session.md")
	personas := []Persona{catalog["sage"], catalog["trickster"]}
	a := New(client, 2, 20, "Does consciousness require a body?", outputPath, "", false, false, personas)

	if err := a.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 2 rounds, 2 personas: 2 opening + 4 rounds + 2 closing = 8 turns
	if len(a.transcript) != 8 {
		t.Errorf("expected 8 transcript turns, got %d", len(a.transcript))
	}
	// Opening: turns 0-1
	if a.transcript[0].persona != "Sage" || a.transcript[0].phase != "opening" {
		t.Errorf("turn 0: expected Sage/opening, got %s/%s", a.transcript[0].persona, a.transcript[0].phase)
	}
	if a.transcript[1].persona != "Trickster" || a.transcript[1].phase != "opening" {
		t.Errorf("turn 1: expected Trickster/opening, got %s/%s", a.transcript[1].persona, a.transcript[1].phase)
	}
	// Round 1: turns 2-3
	if a.transcript[2].persona != "Sage" || a.transcript[2].phase != "round 1" {
		t.Errorf("turn 2: expected Sage/round 1, got %s/%s", a.transcript[2].persona, a.transcript[2].phase)
	}
	if a.transcript[3].persona != "Trickster" || a.transcript[3].phase != "round 1" {
		t.Errorf("turn 3: expected Trickster/round 1, got %s/%s", a.transcript[3].persona, a.transcript[3].phase)
	}
	// Round 2: turns 4-5
	if a.transcript[4].persona != "Sage" || a.transcript[4].phase != "round 2" {
		t.Errorf("turn 4: expected Sage/round 2, got %s/%s", a.transcript[4].persona, a.transcript[4].phase)
	}
	if a.transcript[5].persona != "Trickster" || a.transcript[5].phase != "round 2" {
		t.Errorf("turn 5: expected Trickster/round 2, got %s/%s", a.transcript[5].persona, a.transcript[5].phase)
	}
	// Closing: turns 6-7
	if a.transcript[6].persona != "Sage" || a.transcript[6].phase != "closing" {
		t.Errorf("turn 6: expected Sage/closing, got %s/%s", a.transcript[6].persona, a.transcript[6].phase)
	}
	if a.transcript[7].persona != "Trickster" || a.transcript[7].phase != "closing" {
		t.Errorf("turn 7: expected Trickster/closing, got %s/%s", a.transcript[7].persona, a.transcript[7].phase)
	}

	t.Logf("transcript:\n")
	for i, turn := range a.transcript {
		t.Logf("[%d] %s: %s\n", i, turn.persona, turn.text[:min(80, len(turn.text))])
	}
}

func TestRun_DocumentPersistence(t *testing.T) {
	client, err := llm.New("ollama", testModel, testServerURL)
	if err != nil {
		t.Fatalf("llm.New: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "session.md")
	thinkingFile := thinkingPath(outputPath)

	agent1 := New(client, 2, 20, "What is 1 + 1?", outputPath, "", false, false, testPersonas())
	if err := agent1.Run(context.Background()); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	firstThinking, err := os.ReadFile(thinkingFile)
	if err != nil {
		t.Fatalf("thinking file not created after first run: %v", err)
	}
	t.Logf("thinking after first run:\n%s", firstThinking)

	firstOutput, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file not created after first run: %v", err)
	}
	if !strings.Contains(string(firstOutput), "# Question") {
		t.Error("first run output missing '# Question'")
	}

	agent2 := New(client, 2, 20, "What is 1 + 1?", outputPath, "", false, false, testPersonas())
	if err := agent2.Run(context.Background()); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	secondOutput, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file not readable after second run: %v", err)
	}
	if !strings.Contains(string(secondOutput), "## Positions") {
		t.Error("second run output missing '## Positions'")
	}
	if !strings.Contains(string(secondOutput), "## Next Round") {
		t.Error("second run output missing '## Next Round'")
	}
}
