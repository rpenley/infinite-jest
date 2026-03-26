# Infinite Jest

Infinite Jest is an intrusive thought masquerading as a code base. Multiple LLM personas argue a topic through structured rounds — opening statements, N rounds of back-and-forth, and closing statements — while a persistent thinking document accumulates context across sessions until a final summary of their arguments is produced.

Designed for use with local LLMs (Ollama). Do not point this at a paid API without a hard spending limit — it makes many sequential calls and is purposefully open ended about its goals.

There is also an interactive mode if you decide that having pointless arguments will real people isn't frustrating or challanging enough.

## Usage

```
infinite-jest [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--seed` | — | Topic or question for the debate |
| `--seed-file` | — | Path to a seed file (plain text or markdown with YAML frontmatter) |
| `--personas` | pragmatist | Comma-separated list: `pragmatist`, `sage`, `contrarian`, `idealist`, `realist`, `trickster` |
| `--rounds` | 1 | Number of back-and-forth debate rounds |
| `--output` | output.md | Markdown summary output path |
| `--log` | debate.log | Full transcript log path |
| `--model` | — | Model name (required) |
| `--backend` | ollama | LLM backend: `ollama` or `openai` |
| `--url` | http://localhost:11434 | LLM server base URL |
| `--max-history` | 20 | Max transcript turns passed to each LLM prompt |
| `--config` | ~/.config/infinite-jest/config.json | Config file path |
| `--verbose` | false | Print LLM responses to stdout in real time |
| `--interactive` | false | Join the debate as a participant |
| `--human-name` | You | Your display name in the debate (requires `--interactive`) |
| `--fresh` | false | Ignore any previous session state and start clean |

### Interactive Mode

`--interactive` inserts you into the persona rotation. You go first each round, before the LLM personas respond.

```
infinite-jest --interactive --personas pragmatist --human-name Alice
```

If no `--seed` is given, you will be prompted for the topic at startup:

```
debate topic: Is Go a good systems language?
```

Each time it is your turn, the full debate context is printed followed by a prompt:

```
--- your turn (opening) ---
Topic: Is Go a good systems language?

[Alice] enter your response (blank line to submit, /done to end):
```

Type your response across as many lines as you like, then press Enter on a blank line to submit. The LLM personas respond in turn, then it loops back to you. There are no fixed rounds — the debate continues until you type `/done` on its own line, at which point the session is synthesized and written to the output file.

`--interactive` requires at least one persona to be set explicitly via `--personas` (or in the config file). With no personas the flag is an error.

By default, a session picks up from where the previous one left off using the thinking document (`output.thinking.md` alongside the output file). Use `--fresh` to ignore that state and start a new conversation on a clean slate.

### Seed File Format

```markdown
---
question: Should we rewrite this in Rust?
personas:
  - pragmatist
  - idealist
---

Optional additional context goes here.
```

## Configuration

All flags can be set in a JSON config file. CLI flags override the file.

```json
{
	"backend": "ollama",
	"model": "qwen3:32b",
	"url": "http://localhost:11434",
	"rounds": 2,
	"max_history": 20,
	"personas": ["pragmatist", "contrarian"],
	"output_file": "output.md",
	"log_file": "debate.log"
}
```

See `examples/config.json` for a copy to start from.
