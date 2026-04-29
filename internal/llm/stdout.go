package llm

import "fmt"

// StdoutAdapter is the no-config fallback: it prints the prompt to stdout and
// returns an empty response. The caller should not print anything further.
type StdoutAdapter struct{}

func (s *StdoutAdapter) Ask(prompt string) (string, error) {
	fmt.Print(prompt)
	return "", nil
}
