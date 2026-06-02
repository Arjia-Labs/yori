package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjia-labs/yori/internal/store"
)

type fakeResolver map[string]string

func (f fakeResolver) ReadPartial(name string) ([]byte, error) {
	// Mirror the real store: resolve by base name (strip dir + .md).
	base := strings.TrimSuffix(filepath.Base(name), ".md")
	if v, ok := f[base]; ok {
		return []byte(v), nil
	}
	return nil, os.ErrNotExist
}
func (f fakeResolver) ReadPartialIn(_ string, name string) ([]byte, error) {
	return f.ReadPartial(name)
}
func (f fakeResolver) Resolve(store.Type, string) (*store.Artifact, error) {
	return nil, os.ErrNotExist
}

func TestPromptfooExport(t *testing.T) {
	art := &store.Artifact{
		Name: "review", Type: store.TypePrompt, Path: "/s/review.md",
		Description: "code review",
		Body:        "{% include 'house' %}\nReview this {{ lang }} code:\n{{ input }}",
		Vars:        map[string]store.Var{"lang": {Default: "go"}},
	}
	cases := []Case{
		{"vars": map[string]any{"lang": "python", "input": "x"}, "assert": []any{map[string]any{"type": "contains", "value": "return"}}},
	}
	out, err := Promptfoo(fakeResolver{"house": "Be concise."}, art, cases, []string{"anthropic:claude-opus-4-8"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	checks := []string{
		"Be concise.",               // composition resolved (include inlined)
		"{{ lang }}",                // variable preserved for promptfoo
		"{{ input }}",               // input preserved
		"description: code review",  // metadata carried over
		"anthropic:claude-opus-4-8", // provider
		"type: contains",            // case assertion passed through
	}
	for _, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("export missing %q:\n%s", want, s)
		}
	}
	// The include directive itself must be gone (resolved, not literal).
	if strings.Contains(s, "{% include") {
		t.Errorf("include not resolved:\n%s", s)
	}
}

func TestGuessProvider(t *testing.T) {
	cases := map[string]string{
		"claude-opus-4-8": "anthropic:claude-opus-4-8",
		"gpt-4o":          "openai:gpt-4o",
		"o3-mini":         "openai:o3-mini",
		"llama-3":         "llama-3",
	}
	for model, want := range cases {
		if got := GuessProvider(model); got != want {
			t.Errorf("GuessProvider(%q) = %q want %q", model, got, want)
		}
	}
}
