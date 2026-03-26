package main

import (
	"context"
	"flag"
	"fmt"
	"infinite-jest/agent"
	"infinite-jest/config"
	"infinite-jest/llm"
	"os"
	"strings"
	"time"
)

func main() {
	configPath   := flag.String("config", config.DefaultPath(), "path to config file")
	backendFlag  := flag.String("backend", "", "llm backend: ollama or openai (overrides config)")
	modelFlag    := flag.String("model", "", "model name (overrides config)")
	urlFlag      := flag.String("url", "", "base url for llm server (overrides config)")
	roundsFlag   := flag.Int("rounds", 0, "number of debate rounds, 0=use config default (overrides config)")
	maxHistory   := flag.Int("max-history", 0, "max transcript turns passed to each LLM call (overrides config)")
	seedFlag     := flag.String("seed", "", "seed question (overrides config)")
	seedFileFlag := flag.String("seed-file", "", "path to seed file (plain text or markdown with YAML frontmatter)")
	outputFlag   := flag.String("output", "", "markdown output file path (overrides config)")
	personasFlag := flag.String("personas", "", "comma-separated persona names, e.g. sage,contrarian (overrides config)")
	logFlag         := flag.String("log", "", "transcript log file path (default: ./debate.log)")
	verboseFlag     := flag.Bool("verbose", false, "print LLM responses to terminal in real time")
	interactiveFlag := flag.Bool("interactive", false, "join the debate as a participant")
	humanNameFlag   := flag.String("human-name", "You", "your name in the debate (requires --interactive)")
	freshFlag       := flag.Bool("fresh", false, "ignore any previous session state and start clean")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	if *backendFlag != "" {
		cfg.Backend = *backendFlag
	}
	if *modelFlag != "" {
		cfg.Model = *modelFlag
	}
	if *urlFlag != "" {
		cfg.URL = *urlFlag
	}
	if *roundsFlag != 0 {
		cfg.Rounds = *roundsFlag
	}
	if *maxHistory != 0 {
		cfg.MaxHistory = *maxHistory
	}

	if *seedFileFlag != "" {
		question, body, filePersonas, err := config.ParseSeedFile(*seedFileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
		if question != "" {
			cfg.Seed = question
		}
		if body != "" {
			if cfg.Seed != "" {
				cfg.Seed = cfg.Seed + "\n\n" + body
			} else {
				cfg.Seed = body
			}
		}
		if len(filePersonas) > 0 {
			cfg.Personas = filePersonas
		}
	}

	if *seedFlag != "" {
		cfg.Seed = *seedFlag
	}
	if *outputFlag != "" {
		cfg.OutputFile = *outputFlag
	} else if *interactiveFlag {
		cfg.OutputFile = time.Now().Format("debate-20060102-150405.md")
	}
	if cfg.OutputFile == "" {
		cfg.OutputFile = "output.md"
	}
	if *personasFlag != "" {
		cfg.Personas = strings.Split(*personasFlag, ",")
	}
	if *logFlag != "" {
		cfg.LogFile = *logFlag
	}
	if cfg.LogFile == "" {
		cfg.LogFile = "debate.log"
	}

	if cfg.Model == "" {
		fmt.Fprintln(os.Stderr, "error: model is required (set in config or --model flag)")
		os.Exit(1)
	}

	// Resolve personas — default to pragmatist
	var personas []agent.Persona
	if len(cfg.Personas) == 0 {
		if *interactiveFlag {
			fmt.Fprintln(os.Stderr, "error: --interactive requires at least one persona (use --personas, e.g. --personas pragmatist)")
			os.Exit(1)
		}
		personas = agent.DefaultPersonas()
	} else {
		personas, err = agent.LookupPersonas(cfg.Personas)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
	}

	if *interactiveFlag {
		if cfg.Seed == "" {
			fmt.Print("debate topic: ")
			os.Stdout.Sync()
			var buf [4096]byte
			n, _ := os.Stdin.Read(buf[:])
			cfg.Seed = strings.TrimSpace(string(buf[:n]))
			if cfg.Seed == "" {
				fmt.Fprintln(os.Stderr, "error: debate topic is required")
				os.Exit(1)
			}
		}
		personas = append([]agent.Persona{{Name: *humanNameFlag, IsHuman: true}}, personas...)
	}

	client, err := llm.New(cfg.Backend, cfg.Model, cfg.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	a := agent.New(client, cfg.Rounds, cfg.MaxHistory, cfg.Seed, cfg.OutputFile, cfg.LogFile, *verboseFlag, *freshFlag, personas)
	if err := a.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
