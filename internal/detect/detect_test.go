package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanNode(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{
	  "dependencies": {"next": "14.0.0", "@tanstack/react-query": "5"},
	  "devDependencies": {"typescript": "5"}
	}`)
	s := Scan(dir)
	for _, d := range []string{"next", "@tanstack/react-query", "typescript"} {
		if !s.Deps[d] {
			t.Errorf("missing dep %q in %v", d, s.DepList())
		}
	}
	if len(s.Ecosystems) != 1 || s.Ecosystems[0] != "node" {
		t.Errorf("ecosystems = %v", s.Ecosystems)
	}
}

func TestScanGoAndPython(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module ex\n\ngo 1.22\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.9.1\n\tgithub.com/spf13/cobra v1.8.0\n)\n")
	write(t, dir, "requirements.txt", "# comment\nfastapi>=0.100\nuvicorn\n")
	s := Scan(dir)
	for _, d := range []string{"github.com/gin-gonic/gin", "github.com/spf13/cobra", "fastapi", "uvicorn"} {
		if !s.Deps[d] {
			t.Errorf("missing dep %q in %v", d, s.DepList())
		}
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "tailwind.config.js", "module.exports = {}")
	s := Scan(dir)
	if !s.FileExists("tailwind.config.*") {
		t.Errorf("glob should match tailwind.config.js")
	}
	if s.FileExists("nope.*") {
		t.Errorf("unexpected match")
	}
}
