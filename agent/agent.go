package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"infinite-jest/llm"
	"io"
	"os"
	"strings"
)

type turn struct {
	persona string
	phase   string
	text    string
}

// Agent orchestrates a structured debate between one or more personas,
// calling the LLM for each turn and accumulating a transcript.
type Agent struct {
	client      *llm.Client
	transcript  []turn
	rounds      int
	maxHistory  int
	seed        string
	outputFile  string
	logFile     string
	verbose     bool
	fresh       bool
	personas    []Persona
	InputReader io.Reader
}

// New returns an Agent configured with the given LLM client, debate
// parameters, and personas.
func New(client *llm.Client, rounds, maxHistory int, seed, outputFile, logFile string, verbose, fresh bool, personas []Persona) *Agent {
	return &Agent{
		client:      client,
		rounds:      rounds,
		maxHistory:  maxHistory,
		seed:        seed,
		outputFile:  outputFile,
		logFile:     logFile,
		verbose:     verbose,
		fresh:       fresh,
		personas:    personas,
		InputReader: os.Stdin,
	}
}

func thinkingPath(outputFile string) string {
	if outputFile == "" {
		return ""
	}
	// Insert ".thinking" before the file extension so "output.md" → "output.thinking.md".
	dot := strings.LastIndex(outputFile, ".")
	if dot == -1 {
		return outputFile + ".thinking"
	}
	return outputFile[:dot] + ".thinking" + outputFile[dot:]
}

func loadDocument(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		fmt.Printf("warning: could not read %s: %s\n", path, err)
		return ""
	}
	return string(data)
}

func writeDocument(path, content string) {
	if path == "" {
		return
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("warning: failed to write document: %s\n", err)
	}
}

// buildMessages constructs the two-message prompt for one persona's turn.
// It includes up to maxHistory previous turns in the transcript excerpt.
func (a *Agent) buildMessages(persona Persona, existingThinking string, phase string) []llm.Message {
	var prompt strings.Builder

	if existingThinking != "" {
		prompt.WriteString("Context from previous session:\n")
		prompt.WriteString(existingThinking)
		prompt.WriteString("\n\n---\n\n")
	}

	prompt.WriteString("Topic: ")
	prompt.WriteString(a.seed)

	// Trim transcript to the most recent maxHistory turns to stay within context limits.
	start := 0
	if len(a.transcript) > a.maxHistory {
		start = len(a.transcript) - a.maxHistory
	}
	if start < len(a.transcript) {
		prompt.WriteString("\n\nDebate so far:\n")
		for _, t := range a.transcript[start:] {
			prompt.WriteString(fmt.Sprintf("\n**%s:** %s\n", t.persona, t.text))
		}
	}

	prompt.WriteString(fmt.Sprintf("\n\nYou are %s. %s Respond directly and in character. No preamble.", persona.Name, phase))

	return []llm.Message{
		{Role: "system", Content: persona.Prompt},
		{Role: "user", Content: prompt.String()},
	}
}

func (a *Agent) updateThinking(ctx context.Context, existingThinking string) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Topic: %s\n\nDebate transcript:\n", a.seed))
	for _, t := range a.transcript {
		content.WriteString(fmt.Sprintf("\n**%s:** %s\n", t.persona, t.text))
	}
	if existingThinking != "" {
		content.WriteString(fmt.Sprintf("\n\nPrevious thinking document:\n%s", existingThinking))
	}

	messages := []llm.Message{
		{Role: "system", Content: thinkingUpdatePrompt},
		{Role: "user", Content: content.String()},
	}

	result, err := a.client.Complete(ctx, messages)
	if err != nil {
		if existingThinking != "" {
			return existingThinking
		}
		return fmt.Sprintf("Topic: %s\n\n(thinking update failed)", a.seed)
	}
	return result
}

func (a *Agent) summarize(ctx context.Context, thinking string) string {
	userContent := fmt.Sprintf("Thinking document:\n\n%s\n\nOriginal question: %s", thinking, a.seed)

	messages := []llm.Message{
		{Role: "system", Content: summarizePrompt},
		{Role: "user", Content: userContent},
	}

	result, err := a.client.Complete(ctx, messages)
	if err != nil {
		return fmt.Sprintf("# Question\n\n%s\n\n---\n\n## Positions\n\n(summarize failed)\n\n---\n\n## Key Tensions\n\n(none)\n\n---\n\n## Points of Agreement\n\n(none)\n\n---\n\n## Next Round\n\n(none)\n", a.seed)
	}
	return result
}

func (a *Agent) flush(ctx context.Context, existingThinking string) {
	thinking := a.updateThinking(ctx, existingThinking)
	writeDocument(thinkingPath(a.outputFile), thinking)
	writeDocument(a.outputFile, a.summarize(ctx, thinking))
}

// Run executes a full debate session: opening statements, N rounds of
// back-and-forth, then closing statements. Writes a thinking document
// and a markdown summary to the configured output file.
func (a *Agent) Run(ctx context.Context) error {
	if len(a.personas) == 0 {
		return fmt.Errorf("at least one persona is required")
	}

	hasHuman := false
	for _, p := range a.personas {
		if p.IsHuman {
			hasHuman = true
			break
		}
	}
	if hasHuman {
		fmt.Print("\033[2J\033[H")
	}

	var existingThinking string
	if !a.fresh {
		existingThinking = loadDocument(thinkingPath(a.outputFile))
	}

	// Print header
	fmt.Printf("infinite-jest\n")
	if a.seed != "" {
		preview := a.seed
		if len(preview) > 80 {
			preview = preview[:77] + "..."
		}
		fmt.Printf("topic:    %s\n", preview)
	}
	names := make([]string, len(a.personas))
	for i, p := range a.personas {
		names[i] = p.Name
	}

	rounds := a.rounds
	if rounds < 1 {
		rounds = 1
	}
	fmt.Printf("personas: %s\n", strings.Join(names, ", "))
	if hasHuman {
		fmt.Printf("rounds:   open-ended (/done to finish)\n\n")
	} else {
		fmt.Printf("rounds:   %d\n\n", rounds)
	}

	// Open log file for appending
	var logWriter *os.File
	if a.logFile != "" {
		f, err := os.OpenFile(a.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("warning: could not open log file %s: %s\n", a.logFile, err)
		} else {
			logWriter = f
			defer logWriter.Close()
		}
	}

	inputReader := bufio.NewReader(a.InputReader)

	runTurn := func(persona Persona, phaseLabel, phaseInstruction string) error {
		messages := a.buildMessages(persona, existingThinking, phaseInstruction)
		var response string
		var err error
		if persona.IsHuman {
			response, err = readHumanTurn(inputReader, persona.Name, phaseLabel)
			if errors.Is(err, ErrHumanDone) {
				return ErrHumanDone
			}
			if err != nil {
				return fmt.Errorf("human input: %w", err)
			}
		} else if hasHuman {
			thinkingMsg := fmt.Sprintf("[%s is thinking...]", persona.Name)
			fmt.Print(thinkingMsg)
			os.Stdout.Sync()
			response, err = a.client.Complete(ctx, messages)
			if errors.Is(err, llm.ErrContextExceeded) {
				fmt.Printf("\r\033[K[%s] (context window exceeded — ending here)\n", persona.Name)
				fmt.Printf("\nsynthesizing... ")
				a.flush(ctx, existingThinking)
				fmt.Printf("done\noutput: %s\n", a.outputFile)
				if a.logFile != "" {
					fmt.Printf("log:    %s\n", a.logFile)
				}
				return llm.ErrContextExceeded
			}
			if err != nil {
				fmt.Printf("\r\033[K")
				return fmt.Errorf("llm complete: %w", err)
			}
			fmt.Printf("\r\033[K[%s] %s\n", persona.Name, response)
		} else {
			fmt.Printf("  %s... ", persona.Name)
			response, err = a.client.Complete(ctx, messages)
			if errors.Is(err, llm.ErrContextExceeded) {
				fmt.Printf("\ncontext window exceeded, stopping early\n")
				fmt.Printf("synthesizing... ")
				a.flush(ctx, existingThinking)
				fmt.Printf("done\n")
				fmt.Printf("output: %s\n", a.outputFile)
				if a.logFile != "" {
					fmt.Printf("log:    %s\n", a.logFile)
				}
				return llm.ErrContextExceeded
			}
			if err != nil {
				fmt.Printf("\n")
				return fmt.Errorf("llm complete: %w", err)
			}
			fmt.Printf("done\n")
			if a.verbose {
				fmt.Printf("\n%s\n\n", response)
			}
		}
		if logWriter != nil {
			fmt.Fprintf(logWriter, "=== [%s] %s ===\n%s\n\n", phaseLabel, persona.Name, response)
		}
		a.transcript = append(a.transcript, turn{persona: persona.Name, phase: phaseLabel, text: response})
		return nil
	}

	done := false

	// Opening statements
	if !hasHuman {
		fmt.Printf("opening statements\n")
	}
	for _, persona := range a.personas {
		if err := runTurn(persona, "opening", "This is your opening statement — lay out your position clearly."); err != nil {
			if errors.Is(err, ErrHumanDone) {
				done = true
				break
			}
			if errors.Is(err, llm.ErrContextExceeded) {
				return nil
			}
			return err
		}
	}

	if hasHuman {
		// Open-ended rounds — loop until the human types /done
		for round := 1; !done; round++ {
			for _, persona := range a.personas {
				if err := runTurn(persona, fmt.Sprintf("round %d", round), "Continue the debate."); err != nil {
					if errors.Is(err, ErrHumanDone) {
						done = true
						break
					}
					if errors.Is(err, llm.ErrContextExceeded) {
						return nil
					}
					return err
				}
			}
		}
	} else {
		// Fixed rounds
		for round := 1; round <= rounds; round++ {
			fmt.Printf("\nround %d of %d\n", round, rounds)
			for _, persona := range a.personas {
				instruction := fmt.Sprintf("This is round %d of %d.", round, rounds)
				if err := runTurn(persona, fmt.Sprintf("round %d", round), instruction); err != nil {
					if errors.Is(err, llm.ErrContextExceeded) {
						return nil
					}
					return err
				}
			}
		}

		// Closing statements
		fmt.Printf("\nclosing statements\n")
		for _, persona := range a.personas {
			if err := runTurn(persona, "closing", "This is your closing statement — make your final case."); err != nil {
				if errors.Is(err, llm.ErrContextExceeded) {
					return nil
				}
				return err
			}
		}
	}

	fmt.Printf("\nsynthesizing... ")
	a.flush(ctx, existingThinking)
	fmt.Printf("done\n")
	fmt.Printf("output: %s\n", a.outputFile)
	if a.logFile != "" {
		fmt.Printf("log:    %s\n", a.logFile)
	}
	return nil
}
