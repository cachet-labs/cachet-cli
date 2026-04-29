package llm

// Adapter is the only boundary between cachet and any LLM provider.
// All SDK imports must stay inside this package.
type Adapter interface {
	Ask(prompt string) (string, error)
}
