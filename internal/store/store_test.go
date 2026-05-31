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
	writeFile(t, fileFor(global, TypePrompt, "shared"), "global version")
	writeFile(t, fileFor(global, TypePrompt, "globalonly"), "g")
	writeFile(t, fileFor(project, TypePrompt, "shared"), "project version")

	s := &Store{projectStore: project, globalStore: global}

	a, err := s.Resolve(TypePrompt, "shared")
	if err != nil {
		t.Fatal(err)
	}
	if a.Body != "project version" || a.Layer != "project" {
		t.Errorf("shadow failed: body=%q layer=%q", a.Body, a.Layer)
	}

	g, err := s.Resolve(TypePrompt, "globalonly")
	if err != nil {
		t.Fatal(err)
	}
	if g.Layer != "global" {
		t.Errorf("layer = %q", g.Layer)
	}

	if _, err := s.Resolve(TypePrompt, "missing"); err == nil {
		t.Errorf("expected not found")
	}

	list, err := s.List(TypePrompt, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 { // shared (project) + globalonly, not the shadowed global shared
		t.Errorf("list len = %d: %+v", len(list), list)
	}
}

func TestTypedStorage(t *testing.T) {
	dir := t.TempDir()
	s := &Store{globalStore: filepath.Join(dir, "store")}

	// Save one of each type to the global store.
	for _, typ := range []Type{TypePrompt, TypeAgent, TypeCommand, TypeSkill} {
		if _, err := s.Save(typ, "thing", Scaffold("thing", typ), true); err != nil {
			t.Fatalf("save %s: %v", typ, err)
		}
	}

	// Each resolves independently and lands in the right subdir.
	for _, typ := range []Type{TypePrompt, TypeAgent, TypeCommand, TypeSkill} {
		a, err := s.Resolve(typ, "thing")
		if err != nil {
			t.Fatalf("resolve %s: %v", typ, err)
		}
		if a.Type != typ {
			t.Errorf("type = %q want %q", a.Type, typ)
		}
		wantSub := typ.subdir()
		if wantSub != "" && !contains(a.Path, wantSub) {
			t.Errorf("%s path %q missing subdir %q", typ, a.Path, wantSub)
		}
	}

	// List all types finds four; filtered finds one.
	all, err := s.List("", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Errorf("list all = %d want 4", len(all))
	}
	agents, err := s.List(TypeAgent, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].Type != TypeAgent {
		t.Errorf("list agents = %+v", agents)
	}
}

func contains(s, sub string) bool {
	return filepath.Base(filepath.Dir(s)) == sub
}

func TestRejectTraversalNames(t *testing.T) {
	dir := t.TempDir()
	s := &Store{globalStore: filepath.Join(dir, "store")}

	bad := []string{"../../outside", "nested/foo", "bad: name", "/abs", ".."}
	for _, name := range bad {
		if _, err := s.FilePath(TypePrompt, name, true); err == nil {
			t.Errorf("FilePath accepted %q", name)
		}
		if _, err := s.Save(TypePrompt, name, []byte("x"), true); err == nil {
			t.Errorf("Save accepted %q", name)
		}
		if _, err := s.Resolve(TypePrompt, name); err == nil {
			t.Errorf("Resolve accepted %q", name)
		}
		if err := s.Delete(TypePrompt, name, true); err == nil {
			t.Errorf("Delete accepted %q", name)
		}
	}

	// Nothing escaped the store directory.
	if Exists(filepath.Join(dir, "outside.md")) || Exists(filepath.Join(dir, "store", "nested")) {
		t.Errorf("a rejected name still wrote to disk")
	}
}

func TestListSkipsMalformedFile(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "store")
	writeFile(t, fileFor(global, TypePrompt, "good"), "---\nname: good\n---\nbody")
	// A hand-placed file with broken frontmatter.
	writeFile(t, filepath.Join(global, "broken.md"), "---\nname: bad: name\n---\nbody")

	s := &Store{globalStore: global}
	list, err := s.List(TypePrompt, true, "")
	if err != nil {
		t.Fatalf("List errored instead of skipping: %v", err)
	}
	if len(list) != 1 || list[0].Name != "good" {
		t.Errorf("expected only 'good', got %+v", list)
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
