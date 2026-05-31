package store

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseArtifact(t *testing.T) {
	data := "---\nname: triage\ntags: [a, b]\nvars:\n  tone:\n    default: neutral\n---\nBody {{ tone }}\n"
	a, err := parseArtifact([]byte(data), "/x/triage.md")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "triage" {
		t.Errorf("name = %q", a.Name)
	}
	if a.Body != "Body {{ tone }}\n" {
		t.Errorf("body = %q", a.Body)
	}
	if a.Vars["tone"].Default != "neutral" {
		t.Errorf("default = %q", a.Vars["tone"].Default)
	}
	if len(a.Tags) != 2 {
		t.Errorf("tags = %v", a.Tags)
	}
}

func TestParseArtifactNoFrontmatter(t *testing.T) {
	a, err := parseArtifact([]byte("just a body"), "/x/plain.md")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "plain" {
		t.Errorf("name = %q", a.Name)
	}
	if a.Body != "just a body" {
		t.Errorf("body = %q", a.Body)
	}
}

func TestLayeredResolveAndShadow(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project", "store")
	global := filepath.Join(dir, "global", "store")
	writeFile(t, fileFor(global, "shared"), "global version")
	writeFile(t, fileFor(global, "globalonly"), "g")
	writeFile(t, fileFor(project, "shared"), "project version")

	s := &Store{projectStore: project, globalStore: global}

	a, err := s.Resolve("shared")
	if err != nil {
		t.Fatal(err)
	}
	if a.Body != "project version" || a.Layer != "project" {
		t.Errorf("shadow failed: body=%q layer=%q", a.Body, a.Layer)
	}

	g, err := s.Resolve("globalonly")
	if err != nil {
		t.Fatal(err)
	}
	if g.Layer != "global" {
		t.Errorf("layer = %q", g.Layer)
	}

	if _, err := s.Resolve("missing"); err == nil {
		t.Errorf("expected not found")
	}

	list, err := s.List(false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 { // shared (project) + globalonly, not the shadowed global shared
		t.Errorf("list len = %d: %+v", len(list), list)
	}
}

func TestReadPartial(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "store")
	writeFile(t, filepath.Join(global, partialsDir, "house.md"), "Be concise.")
	s := &Store{globalStore: global}

	for _, name := range []string{"house", "partials/house.md", "house.md"} {
		data, err := s.ReadPartial(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(data) != "Be concise." {
			t.Errorf("%s: got %q", name, data)
		}
	}
}
