package agent

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrHumanDone = errors.New("debate ended by human")

func readHumanTurn(reader *bufio.Reader, personaName, phaseLabel string) (string, error) {
	fmt.Printf("\n── %s ──\n[%s] > ", phaseLabel, personaName)
	os.Stdout.Sync()

	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", fmt.Errorf("reading input: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "/done" {
		return "", ErrHumanDone
	}
	return line, nil
}
