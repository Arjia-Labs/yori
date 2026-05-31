package registry

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNameFromURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/acme/prompts.git": "prompts",
		"https://github.com/acme/prompts":     "prompts",
		"git@github.com:acme/prompts.git":     "prompts",
		"file:///tmp/reg-remote/":             "reg-remote",
	}
	for url, want := range cases {
		if got := NameFromURL(url); got != want {
			t.Errorf("NameFromURL(%q) = %q want %q", url, got, want)
		}
	}
}

// makeSourceRepo creates a git repo with one committed file and returns its path.
func makeSourceRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "review.md"), []byte("Review {{ input }}"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-m", "init")
	return dir
}

func TestInstallUpdateUninstall(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("YORI_HOME", t.TempDir())
	src := makeSourceRepo(t)

	idx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	p, err := idx.Install("file://"+src, "acme")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if p.Name != "acme" || p.Commit == "" {
		t.Errorf("bad pkg: %+v", p)
	}
	if !fileExists(filepath.Join(idx.Dir("acme"), "review.md")) {
		t.Errorf("cloned file missing")
	}

	// Reload from disk: the index persisted.
	reloaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Find("acme") == nil {
		t.Errorf("acme not persisted")
	}

	// Double install is rejected.
	if _, err := reloaded.Install("file://"+src, "acme"); err == nil {
		t.Errorf("expected double-install error")
	}

	// Uninstall removes the clone and the index entry.
	if err := reloaded.Uninstall("acme"); err != nil {
		t.Fatal(err)
	}
	if reloaded.Find("acme") != nil {
		t.Errorf("acme still indexed")
	}
	if fileExists(idx.Dir("acme")) {
		t.Errorf("clone dir not removed")
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
