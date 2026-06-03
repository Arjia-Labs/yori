package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arjia-labs/yori/internal/store"
)

func writeStore(t *testing.T, files map[string]string) (*store.Store, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("YORI_HOME", home)
	dir := filepath.Join(home, "store")
	for rel, content := range files {
		p := filepath.Join(dir, rel)
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
	return s, dir
}

func TestBuildInfersFilesAndDeps(t *testing.T) {
	s, dir := writeStore(t, map[string]string{
		"partials/house.md":          "Be concise.",
		"base-reviewer.md":           "---\nname: base-reviewer\n---\n{% include 'house' %}",
		"security-review.md":         "---\nname: security-review\nextends: base-reviewer\ntags: [review]\n---\nx",
		"security-review.cases.yaml": "- vars: {}\n",
	})
	arts, err := s.List("", true, "")
	if err != nil {
		t.Fatal(err)
	}
	m, err := Build(dir, arts, Meta{Name: "acme"})
	if err != nil {
		t.Fatal(err)
	}

	sr := m.Find("security-review")
	if sr == nil {
		t.Fatal("security-review missing")
	}
	// The sibling cases file is bundled into the item's files.
	if !contains(sr.Files, "security-review.md") || !contains(sr.Files, "security-review.cases.yaml") {
		t.Errorf("files = %v", sr.Files)
	}
	// Direct dependency is the base it extends.
	if !contains(sr.Dependencies, "base-reviewer") {
		t.Errorf("deps = %v", sr.Dependencies)
	}
	// The partial is emitted as its own item.
	if p := m.Find("house"); p == nil || p.Type != "partial" {
		t.Errorf("partial item missing: %+v", p)
	}
	// base-reviewer depends on the partial it includes.
	if b := m.Find("base-reviewer"); b == nil || !contains(b.Dependencies, "house") {
		t.Errorf("base deps = %+v", b)
	}
}

func TestClosure(t *testing.T) {
	m := &Manifest{Items: []Item{
		{Name: "a", Dependencies: []string{"b"}},
		{Name: "b", Dependencies: []string{"c"}},
		{Name: "c"},
		{Name: "z"},
	}}
	got, err := m.Closure([]string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, it := range got {
		names[it.Name] = true
	}
	if !names["a"] || !names["b"] || !names["c"] {
		t.Errorf("closure = %v", names)
	}
	if names["z"] {
		t.Errorf("z should not be in closure")
	}
	if _, err := m.Closure([]string{"missing"}); err == nil {
		t.Errorf("expected error for missing item")
	}
}

func TestSafeJoinRejectsTraversal(t *testing.T) {
	base := "/store"
	for _, bad := range []string{"../escape", "../../etc/passwd", "/abs", "a/../../b"} {
		if _, err := safeJoin(base, bad); err == nil {
			t.Errorf("safeJoin accepted %q", bad)
		}
	}
	if got, err := safeJoin(base, "partials/house.md"); err != nil || got != "/store/partials/house.md" {
		t.Errorf("safeJoin(good) = %q, %v", got, err)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
