package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjia-labs/yori/internal/store"
)

// noResolver satisfies render.Resolver; the test artifacts use no partials or
// extends, so its methods are never exercised.
type noResolver struct{}

func (noResolver) ReadPartial(string) ([]byte, error)           { return nil, os.ErrNotExist }
func (noResolver) ReadPartialIn(string, string) ([]byte, error) { return nil, os.ErrNotExist }
func (noResolver) Resolve(store.Type, string) (*store.Artifact, error) {
	return nil, os.ErrNotExist
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func claudeOpts(base, state string) Options {
	return Options{Agent: "claude-code", BaseDir: base, State: state}
}

func TestSyncRenderCopyPrune(t *testing.T) {
	base := t.TempDir()
	state := filepath.Join(t.TempDir(), "synced.json")

	bundle := t.TempDir()
	if err := os.WriteFile(filepath.Join(bundle, "helper.py"), []byte("h"), 0o644); err != nil {
		t.Fatal(err)
	}
	skill := &store.Artifact{
		Name: "researcher", Type: store.TypeSkill,
		Path: filepath.Join(bundle, "SKILL.md"), BundleDir: bundle,
		Body: "Do {{ depth }} research.",
		Vars: map[string]store.Var{"depth": {Default: "shallow"}},
	}
	cmd := &store.Artifact{Name: "triage", Type: store.TypeCommand, Path: "/x/triage.md", Body: "Triage: {{ input }}"}

	opts := claudeOpts(base, state)
	opts.Set = map[string]string{"depth": "deep"}
	res, err := Sync(noResolver{}, []*store.Artifact{skill, cmd}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 2 {
		t.Errorf("written = %v", res.Written)
	}

	// Skill rendered with the override, supporting file copied.
	skillMd := read(t, filepath.Join(base, ".claude/skills/researcher/SKILL.md"))
	if strings.TrimSpace(skillMd) != "Do deep research." {
		t.Errorf("skill body = %q", skillMd)
	}
	if !exists(filepath.Join(base, ".claude/skills/researcher/helper.py")) {
		t.Errorf("supporting file not copied")
	}
	// Command's {{ input }} becomes the agent placeholder, not empty.
	cmdMd := read(t, filepath.Join(base, ".claude/commands/triage.md"))
	if strings.TrimSpace(cmdMd) != "Triage: $ARGUMENTS" {
		t.Errorf("command body = %q", cmdMd)
	}

	// Re-sync with only the skill prunes the command.
	res2, err := Sync(noResolver{}, []*store.Artifact{skill}, claudeOpts(base, state))
	if err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(base, ".claude/commands/triage.md")) {
		t.Errorf("command not pruned")
	}
	if len(res2.Pruned) != 1 {
		t.Errorf("pruned = %v", res2.Pruned)
	}
}

func TestSyncClobberGuard(t *testing.T) {
	base := t.TempDir()
	state := filepath.Join(t.TempDir(), "s.json")
	// A foreign file yori didn't create.
	if err := os.MkdirAll(filepath.Join(base, ".claude/commands"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".claude/commands/triage.md"), []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := &store.Artifact{Name: "triage", Type: store.TypeCommand, Body: "mine"}

	if _, err := Sync(noResolver{}, []*store.Artifact{cmd}, claudeOpts(base, state)); err == nil {
		t.Errorf("expected clobber refusal")
	}
	opts := claudeOpts(base, state)
	opts.Force = true
	if _, err := Sync(noResolver{}, []*store.Artifact{cmd}, opts); err != nil {
		t.Errorf("--force should overwrite: %v", err)
	}
	if got := strings.TrimSpace(read(t, filepath.Join(base, ".claude/commands/triage.md"))); got != "mine" {
		t.Errorf("not overwritten: %q", got)
	}
}

func TestSyncLink(t *testing.T) {
	base := t.TempDir()
	dir := t.TempDir()
	src := filepath.Join(dir, "static.md")
	if err := os.WriteFile(src, []byte("static body"), 0o644); err != nil {
		t.Fatal(err)
	}
	static := &store.Artifact{Name: "static", Type: store.TypeCommand, Path: src, Body: "static body"}

	opts := claudeOpts(base, filepath.Join(t.TempDir(), "s.json"))
	opts.Link = true
	if _, err := Sync(noResolver{}, []*store.Artifact{static}, opts); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, ".claude/commands/static.md")
	fi, err := os.Lstat(target)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected a symlink at %s (err=%v)", target, err)
	}

	// A templated artifact can't be linked.
	tmpl := &store.Artifact{Name: "t", Type: store.TypeCommand, Path: "/x", Body: "{{ x }}"}
	opts2 := claudeOpts(base, filepath.Join(t.TempDir(), "s2.json"))
	opts2.Link = true
	if _, err := Sync(noResolver{}, []*store.Artifact{tmpl}, opts2); err == nil {
		t.Errorf("expected --link to refuse a template")
	}
}

func TestExpandAgents(t *testing.T) {
	if got := ExpandAgents([]string{"*"}); len(got) != len(Agents) {
		t.Errorf("'*' expanded to %v, want %d agents", got, len(Agents))
	}
	if got := ExpandAgents([]string{"claude-code"}); len(got) != 1 || got[0] != "claude-code" {
		t.Errorf("explicit agents changed: %v", got)
	}
}

func TestSubagentFrontmatter(t *testing.T) {
	base := t.TempDir()
	agent := &store.Artifact{
		Name: "pr-bot", Type: store.TypeAgent, Path: "/x/pr-bot.md",
		Description: "reviews PRs", Model: "claude-opus-4-8",
		Body: "You are a PR reviewer.",
	}
	if _, err := Sync(noResolver{}, []*store.Artifact{agent}, claudeOpts(base, filepath.Join(t.TempDir(), "s.json"))); err != nil {
		t.Fatal(err)
	}
	out := read(t, filepath.Join(base, ".claude/agents/pr-bot.md"))
	for _, want := range []string{"---", "name: pr-bot", "description: reviews PRs", "model: claude-opus-4-8", "You are a PR reviewer."} {
		if !strings.Contains(out, want) {
			t.Errorf("subagent file missing %q:\n%s", want, out)
		}
	}
}

func TestScopeAwareSkipAndPlacement(t *testing.T) {
	cmd := &store.Artifact{Name: "deploy", Type: store.TypeCommand, Path: "/x/deploy.md", Body: "Deploy: {{ input }}"}

	// Codex has no project-scope prompt dir → skipped, nothing written.
	base := t.TempDir()
	res, err := Sync(noResolver{}, []*store.Artifact{cmd}, Options{Agent: "codex", BaseDir: base, State: filepath.Join(t.TempDir(), "s.json")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 0 || len(res.Skipped) != 1 {
		t.Errorf("codex project: written=%v skipped=%v", res.Written, res.Skipped)
	}

	// At global scope it lands in .codex/prompts.
	gbase := t.TempDir()
	if _, err := Sync(noResolver{}, []*store.Artifact{cmd}, Options{Agent: "codex", BaseDir: gbase, Global: true, State: filepath.Join(t.TempDir(), "s.json")}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(read(t, filepath.Join(gbase, ".codex/prompts/deploy.md"))); got != "Deploy: $ARGUMENTS" {
		t.Errorf("codex global command = %q", got)
	}
}

func TestCodexSkillPlacement(t *testing.T) {
	base := t.TempDir()
	bundle := t.TempDir()
	skill := &store.Artifact{Name: "researcher", Type: store.TypeSkill, Path: filepath.Join(bundle, "SKILL.md"), BundleDir: bundle, Body: "body"}
	if _, err := Sync(noResolver{}, []*store.Artifact{skill}, Options{Agent: "codex", BaseDir: base, State: filepath.Join(t.TempDir(), "s.json")}); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(base, ".agents/skills/researcher/SKILL.md")) {
		t.Errorf("codex skill not at .agents/skills")
	}
}

func TestUnsync(t *testing.T) {
	base := t.TempDir()
	state := filepath.Join(t.TempDir(), "s.json")
	cmd := &store.Artifact{Name: "triage", Type: store.TypeCommand, Body: "x"}
	if _, err := Sync(noResolver{}, []*store.Artifact{cmd}, claudeOpts(base, state)); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, ".claude/commands/triage.md")
	if !exists(target) {
		t.Fatal("not written")
	}
	removed, err := Unsync(Options{Agent: "claude-code", State: state})
	if err != nil || len(removed) != 1 {
		t.Fatalf("unsync removed=%v err=%v", removed, err)
	}
	if exists(target) {
		t.Errorf("target not removed")
	}
}
