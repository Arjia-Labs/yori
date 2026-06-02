package store

import (
	"os"
	"path/filepath"
	"strings"
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
		if wantSub != "" && !strings.Contains(a.Path, "/"+wantSub+"/") {
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

func TestSkillBundles(t *testing.T) {
	dir := t.TempDir()
	s := &Store{globalStore: filepath.Join(dir, "store")}

	// add --type skill creates a bundle: skills/<name>/SKILL.md.
	path, err := s.Save(TypeSkill, "researcher", Scaffold("researcher", TypeSkill), true)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "SKILL.md" || filepath.Base(filepath.Dir(path)) != "researcher" {
		t.Errorf("bundle path = %q", path)
	}
	// A supporting file alongside SKILL.md.
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "helper.py"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := s.Resolve(TypeSkill, "researcher")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "researcher" {
		t.Errorf("name = %q (should come from the bundle dir, not 'SKILL')", a.Name)
	}
	if a.BundleDir != filepath.Dir(path) {
		t.Errorf("BundleDir = %q", a.BundleDir)
	}

	// A single-file skill still resolves (and the bundle wins when both exist).
	single := filepath.Join(dir, "store", "skills", "legacy.md")
	writeFile(t, single, "---\nname: legacy\n---\nbody")
	if a, err := s.Resolve(TypeSkill, "legacy"); err != nil || a.BundleDir != "" {
		t.Errorf("single-file skill: err=%v bundleDir=%q", err, a.BundleDir)
	}

	// List shows both, once each.
	skills, err := s.List(TypeSkill, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 {
		t.Errorf("list skills = %d (%+v)", len(skills), skills)
	}

	// Delete removes the whole bundle directory.
	if err := s.Delete(TypeSkill, "researcher", true); err != nil {
		t.Fatal(err)
	}
	if Exists(filepath.Dir(path)) {
		t.Errorf("bundle dir not removed")
	}
}

func TestRejectTraversalNames(t *testing.T) {
	dir := t.TempDir()
	s := &Store{globalStore: filepath.Join(dir, "store")}

	// Traversal / non-contained names are rejected by every path operation.
	traversal := []string{"../../outside", "nested/foo", "/abs", "..", "."}
	for _, name := range traversal {
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

// A contained-but-ugly name (no path separators) is rejected on *creation*
// but remains manageable for path ops, so legacy files stay reachable.
func TestContainedUglyNameSplit(t *testing.T) {
	dir := t.TempDir()
	s := &Store{globalStore: filepath.Join(dir, "store")}

	if _, err := s.Save(TypePrompt, "bad: name", []byte("x"), true); err == nil {
		t.Errorf("Save should reject strict-invalid name")
	}
	if _, err := s.FilePath(TypePrompt, "bad: name", true); err != nil {
		t.Errorf("FilePath should allow a contained name: %v", err)
	}
	// Hand-place a legacy file, then confirm it's resolvable and deletable.
	writeFile(t, fileFor(filepath.Join(dir, "store"), TypePrompt, "bad: name"), "---\nname: \"bad: name\"\n---\nbody")
	if _, err := s.Resolve(TypePrompt, "bad: name"); err != nil {
		t.Errorf("Resolve should reach legacy file: %v", err)
	}
	if err := s.Delete(TypePrompt, "bad: name", true); err != nil {
		t.Errorf("Delete should remove legacy file: %v", err)
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

	// A dotted name must not be truncated at the dot: style.v1 resolves
	// style.v1.md, not style.md.
	writeFile(t, filepath.Join(global, partialsDir, "style.md"), "base")
	writeFile(t, filepath.Join(global, partialsDir, "style.v1.md"), "v1")
	for name, want := range map[string]string{"style": "base", "style.v1": "v1", "style.v1.md": "v1"} {
		data, err := s.ReadPartial(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(data) != want {
			t.Errorf("partial %q = %q, want %q", name, data, want)
		}
	}
}
