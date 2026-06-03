package registry

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arjia-labs/yori/internal/config"
)

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"github.com/acme/prompts":         "https://github.com/acme/prompts",
		"gitlab.com/org/repo":             "https://gitlab.com/org/repo",
		"https://github.com/acme/prompts": "https://github.com/acme/prompts", // unchanged
		"git@github.com:acme/prompts.git": "git@github.com:acme/prompts.git", // unchanged
		"file:///tmp/reg":                 "file:///tmp/reg",                 // unchanged
		"/tmp/local/path":                 "/tmp/local/path",                 // local path, unchanged
	}
	for in, want := range cases {
		if got := NormalizeURL(in); got != want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", in, got, want)
		}
	}
}

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

func TestInvalidPersistedNameQuarantined(t *testing.T) {
	t.Setenv("YORI_HOME", t.TempDir())
	regFile, err := config.RegistryFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(regFile), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "packages:\n  - name: ../evil\n    url: file:///x\n    commit: dead\n"
	if err := os.WriteFile(regFile, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// The invalid entry is never exposed as a resolution layer.
	if dirs := idx.Dirs(); len(dirs) != 0 {
		t.Errorf("Dirs() = %+v, want none (invalid entry excluded)", dirs)
	}
	// update-all skips it without trying to git-pull an untrusted path, and
	// reports zero updated (not one).
	if n, err := idx.Update(""); err != nil || n != 0 {
		t.Errorf("Update(all): n=%d err=%v, want 0 updated and no error", n, err)
	}
	// But it can still be cleaned up by name, with no filesystem operation.
	if idx.Find("../evil") == nil {
		t.Fatal("invalid entry should still be listed for cleanup")
	}
	if err := idx.Uninstall("../evil"); err != nil {
		t.Errorf("Uninstall of invalid persisted name: %v", err)
	}
	if idx.Find("../evil") != nil {
		t.Errorf("invalid entry not removed")
	}
}

func TestRejectBadPackageName(t *testing.T) {
	t.Setenv("YORI_HOME", t.TempDir())
	idx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../pkg-escape", "a/b", "..", "/abs"} {
		if _, err := idx.Install("file:///irrelevant", name); err == nil {
			t.Errorf("Install accepted bad name %q", name)
		}
		if err := idx.Uninstall(name); err == nil {
			t.Errorf("Uninstall accepted bad name %q", name)
		}
	}
	// No escaped clone directory was created.
	if fileExists(filepath.Join(idx.pkgRoot, "..", "pkg-escape")) {
		t.Errorf("escaped clone dir was created")
	}
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

func TestPushRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	for _, kv := range [][2]string{
		{"GIT_AUTHOR_NAME", "t"}, {"GIT_AUTHOR_EMAIL", "t@t"},
		{"GIT_COMMITTER_NAME", "t"}, {"GIT_COMMITTER_EMAIL", "t@t"},
	} {
		t.Setenv(kv[0], kv[1])
	}

	// A bare remote registry.
	bare := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v\n%s", err, out)
	}

	// A local store with one artifact, published to the bare remote.
	storeDir := filepath.Join(t.TempDir(), "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "review.md"), []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InitRepo(storeDir); err != nil {
		t.Fatal(err)
	}
	if err := SetRemote(storeDir, bare); err != nil {
		t.Fatal(err)
	}
	if committed, err := CommitAll(storeDir, "publish"); err != nil || !committed {
		t.Fatalf("commit: committed=%v err=%v", committed, err)
	}
	if err := Push(storeDir); err != nil {
		t.Fatal(err)
	}

	// Install from the bare remote and confirm the artifact arrived.
	t.Setenv("YORI_HOME", t.TempDir())
	idx, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Install(bare, "shared"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if !fileExists(filepath.Join(idx.Dir("shared"), "review.md")) {
		t.Errorf("pushed artifact missing after install")
	}
}
