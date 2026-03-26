package agent

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

func TestReadHumanTurn_SingleLine(t *testing.T) {
	input := bufio.NewReader(strings.NewReader("hello world\n"))
	response, err := readHumanTurn(input, "You", "opening")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "hello world" {
		t.Errorf("got %q, want %q", response, "hello world")
	}
}

func TestReadHumanTurn_EmptyInput(t *testing.T) {
	input := bufio.NewReader(strings.NewReader("\n"))
	response, err := readHumanTurn(input, "You", "closing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "" {
		t.Errorf("expected empty response, got %q", response)
	}
}

func TestReadHumanTurn_Done(t *testing.T) {
	input := bufio.NewReader(strings.NewReader("/done\n"))
	_, err := readHumanTurn(input, "You", "round 1")
	if !errors.Is(err, ErrHumanDone) {
		t.Errorf("expected ErrHumanDone, got %v", err)
	}
}
