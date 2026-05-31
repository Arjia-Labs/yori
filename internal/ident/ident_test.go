package ident

import "testing"

func TestValid(t *testing.T) {
	good := []string{"triage", "pr-bot", "house_style", "v1.2", "a", "Z9"}
	for _, n := range good {
		if !Valid(n) {
			t.Errorf("expected %q to be valid", n)
		}
		if err := Validate("artifact", n); err != nil {
			t.Errorf("Validate(%q) = %v", n, err)
		}
	}

	bad := []string{
		"", ".", "..", "../escape", "../../outside-store", "nested/foo",
		"/abs", "a/b", `a\b`, "bad: name", "has space", ".hidden", "-flag",
	}
	for _, n := range bad {
		if Valid(n) {
			t.Errorf("expected %q to be invalid", n)
		}
		if err := Validate("artifact", n); err == nil {
			t.Errorf("expected error for %q", n)
		}
	}
}
