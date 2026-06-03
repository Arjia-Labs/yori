package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arjia-labs/yori/internal/store"
)

// writeStore lays out a global store under YORI_HOME and returns a *store.Store.
func writeStore(t *testing.T, files map[string]string) *store.Store {
	t.Helper()
	home := t.TempDir()
	t.Setenv("YORI_HOME", home)
	for rel, content := range files {
		p := filepath.Join(home, "store", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := store.New()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func names(ns []Node) map[string]bool {
	m := map[string]bool{}
	for _, n := range ns {
		m[n.Name] = true
	}
	return m
}

func TestDepsTransitive(t *testing.T) {
	s := writeStore(t, map[string]string{
		"partials/house.md":  "Be concise.",
		"partials/output.md": "{% include 'house' %}\nFormat rules.",
		"base-reviewer.md":   "---\nname: base-reviewer\n---\n{% include 'house' %}\nReview {{ input }}",
		"security-review.md": "---\nname: security-review\nextends: base-reviewer\n---\n{% include 'output' %}\nfill",
	})

	a, err := s.Resolve(store.TypePrompt, "security-review")
	if err != nil {
		t.Fatal(err)
	}
	d := DepsOf(s, a)

	// extends chain: base-reviewer
	if bn := names(d.Bases); !bn["base-reviewer"] || len(d.Bases) != 1 {
		t.Errorf("bases = %v", d.Bases)
	}
	// transitive partials: output (direct), house (via output AND via base)
	pn := names(d.Partials)
	if !pn["output"] || !pn["house"] {
		t.Errorf("partials = %v", d.Partials)
	}
}

func TestAffected(t *testing.T) {
	s := writeStore(t, map[string]string{
		"partials/house.md":  "Be concise.",
		"base-reviewer.md":   "---\nname: base-reviewer\n---\n{% include 'house' %}",
		"security-review.md": "---\nname: security-review\nextends: base-reviewer\n---\nx",
		"unrelated.md":       "---\nname: unrelated\n---\nno deps",
	})

	// Editing partial 'house' affects base-reviewer (includes it) and
	// security-review (extends base-reviewer).
	got := names(toArts(AffectedBy(s, Node{Name: "house", Partial: true})))
	if !got["base-reviewer"] || !got["security-review"] {
		t.Errorf("affected by house = %v", got)
	}
	if got["unrelated"] {
		t.Errorf("unrelated should not be affected")
	}

	// Editing base 'base-reviewer' affects security-review (extends it).
	got = names(toArts(AffectedBy(s, Node{Type: store.TypePrompt, Name: "base-reviewer"})))
	if !got["security-review"] {
		t.Errorf("affected by base-reviewer = %v", got)
	}
}

func TestDepsCycleTerminates(t *testing.T) {
	s := writeStore(t, map[string]string{
		"a.md": "---\nname: a\nextends: b\n---\nx",
		"b.md": "---\nname: b\nextends: a\n---\ny",
	})
	a, err := s.Resolve(store.TypePrompt, "a")
	if err != nil {
		t.Fatal(err)
	}
	d := DepsOf(s, a) // must not loop forever
	if !names(d.Bases)["b"] {
		t.Errorf("expected b in deps: %v", d.Bases)
	}
}

// toArts adapts AffectedBy's (slice, error) to a slice for names() in tests.
func toArts(arts []*store.Artifact, err error) []Node {
	if err != nil {
		panic(err)
	}
	ns := make([]Node, len(arts))
	for i, a := range arts {
		ns[i] = Node{Type: a.Type, Name: a.Name}
	}
	return ns
}
