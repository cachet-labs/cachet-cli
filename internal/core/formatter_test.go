package core

import (
	"os"
	"testing"
	"time"
)

func TestBuildPromptGolden(t *testing.T) {
	f := &Failure{
		Request:  Request{Method: "POST", URL: "/pay", Body: `{"amount":100}`},
		Response: Response{Status: 500, Body: `{"error":"timeout"}`},
		Error:    ErrorInfo{Type: "timeout", Message: "upstream service timed out after 30s"},
	}
	cases := []*Case{
		{
			Fingerprint: "POST:/pay:500:timeout",
			RootCause:   "Connection pool limit exceeded",
			Fix:         "Set MAX_CONNECTIONS=20 and added retry with backoff",
			CreatedAt:   time.Now(),
		},
	}

	got := BuildPrompt(f, cases)

	const golden = "testdata/formatter_golden.txt"

	// Set UPDATE_GOLDEN=1 to regenerate the golden file.
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		t.Log("golden file updated")
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	if got != string(want) {
		t.Errorf("BuildPrompt output does not match golden file %s\n\ngot:\n%s\nwant:\n%s",
			golden, got, string(want))
	}
}
