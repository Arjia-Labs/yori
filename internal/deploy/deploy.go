// Package deploy materializes yori artifacts into the on-disk locations that
// coding agents (Claude Code, …) discover them from. Skills and commands are
// rendered (variables/includes/slots resolved) and written, tracked in a state
// file so re-sync can prune and unsync can clean up.
package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arjia-labs/yori/internal/render"
	"github.com/arjia-labs/yori/internal/store"
)

// Agent maps artifact types to an agent's discovery directories, relative to a
// base (the project root, or the home dir for global installs).
type Agent struct {
	Name        string
	SkillsDir   string // e.g. ".claude/skills"
	CommandsDir string // e.g. ".claude/commands"
	// ArgPlaceholder is what {{ input }} in a command renders to, so the agent
	// fills the argument at invocation rather than yori at sync time.
	ArgPlaceholder string
}

// Agents is the registry of supported targets.
var Agents = map[string]Agent{
	"claude-code": {
		Name: "claude-code", SkillsDir: ".claude/skills", CommandsDir: ".claude/commands",
		ArgPlaceholder: "$ARGUMENTS",
	},
}

// AgentNames returns the supported agent identifiers, sorted.
func AgentNames() []string {
	names := make([]string, 0, len(Agents))
	for n := range Agents {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Options configures a sync.
type Options struct {
	Agent   string            // target agent identifier
	BaseDir string            // where agent dirs live (project root or home)
	State   string            // path to the sync-state file
	Link    bool              // symlink instead of render+copy (static artifacts only)
	Force   bool              // overwrite files yori didn't create
	Set     map[string]string // variable overrides applied at render time
}

// Result summarizes a sync.
type Result struct {
	Written []string // artifact labels written this run
	Pruned  []string // target paths removed because their source is gone
}

// Sync renders and places the given artifacts, pruning previously-synced
// targets that are no longer present. arts should contain skills and commands.
func Sync(rs render.Resolver, arts []*store.Artifact, opts Options) (*Result, error) {
	agent, ok := Agents[opts.Agent]
	if !ok {
		return nil, fmt.Errorf("unknown agent %q (supported: %s)", opts.Agent, strings.Join(AgentNames(), ", "))
	}

	st, err := loadState(opts.State)
	if err != nil {
		return nil, err
	}
	prev := st.set(opts.Agent)

	res := &Result{}
	current := map[string]bool{}

	for _, a := range arts {
		target, err := targetPath(agent, opts.BaseDir, a)
		if err != nil {
			return nil, err
		}
		if err := guardClobber(target, prev, opts.Force); err != nil {
			return nil, err
		}
		if err := place(rs, a, target, agent, opts); err != nil {
			return nil, fmt.Errorf("sync %s %q: %w", a.Type, a.Name, err)
		}
		current[target] = true
		res.Written = append(res.Written, string(a.Type)+":"+a.Name)
	}

	// Prune targets from a previous sync that we didn't write this time.
	for old := range prev {
		if !current[old] {
			if err := os.RemoveAll(old); err != nil {
				return nil, err
			}
			res.Pruned = append(res.Pruned, old)
		}
	}

	st.Deployed[opts.Agent] = sortedKeys(current)
	if err := st.save(opts.State); err != nil {
		return nil, err
	}
	sort.Strings(res.Written)
	return res, nil
}

// Unsync removes every target previously synced for the agent and clears state.
func Unsync(opts Options) ([]string, error) {
	if _, ok := Agents[opts.Agent]; !ok {
		return nil, fmt.Errorf("unknown agent %q (supported: %s)", opts.Agent, strings.Join(AgentNames(), ", "))
	}
	st, err := loadState(opts.State)
	if err != nil {
		return nil, err
	}
	var removed []string
	for target := range st.set(opts.Agent) {
		if err := os.RemoveAll(target); err != nil {
			return nil, err
		}
		removed = append(removed, target)
	}
	delete(st.Deployed, opts.Agent)
	if err := st.save(opts.State); err != nil {
		return nil, err
	}
	sort.Strings(removed)
	return removed, nil
}

// targetPath is the path yori manages for an artifact: a skill's bundle dir or
// a command's file.
func targetPath(agent Agent, base string, a *store.Artifact) (string, error) {
	switch a.Type {
	case store.TypeSkill:
		return filepath.Join(base, agent.SkillsDir, a.Name), nil
	case store.TypeCommand:
		return filepath.Join(base, agent.CommandsDir, a.Name+".md"), nil
	default:
		return "", fmt.Errorf("agent %q does not support type %q", agent.Name, a.Type)
	}
}

// place writes one artifact to target (render+copy, or symlink with --link).
func place(rs render.Resolver, a *store.Artifact, target string, agent Agent, opts Options) error {
	if opts.Link {
		return link(a, target)
	}
	// A command's {{ input }} is the argument the agent fills at invocation, so
	// render it to the agent's placeholder rather than to empty.
	var inputArg string
	if a.Type == store.TypeCommand {
		inputArg = agent.ArgPlaceholder
	}
	body, err := renderArtifact(rs, a, opts.Set, inputArg)
	if err != nil {
		return err
	}
	switch a.Type {
	case store.TypeSkill:
		if err := os.RemoveAll(target); err != nil {
			return err
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, "SKILL.md"), []byte(body), 0o644); err != nil {
			return err
		}
		return copySupportFiles(a.BundleDir, target)
	case store.TypeCommand:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(body), 0o644)
	}
	return fmt.Errorf("unsupported type %q", a.Type)
}

// link symlinks a static (non-templated) artifact to target.
func link(a *store.Artifact, target string) error {
	if hasTemplate(a.Body) {
		return fmt.Errorf("%q uses template syntax and can't be linked; sync without --link to render it", a.Name)
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	src := a.Path
	if a.Type == store.TypeSkill && a.BundleDir != "" {
		src = a.BundleDir // link the whole bundle dir as .../skills/<name>
	}
	return os.Symlink(src, target)
}

// renderArtifact resolves an artifact's template with its var defaults plus
// any --set overrides. inputArg binds {{ input }} (e.g. an agent placeholder)
// without being appended when unused.
func renderArtifact(rs render.Resolver, a *store.Artifact, set map[string]string, inputArg string) (string, error) {
	vars := map[string]any{}
	for name, v := range a.Vars {
		if v.Default != "" {
			vars[name] = v.Default
		}
	}
	for k, v := range set {
		vars[k] = v
	}
	return render.Render(a, rs, render.Options{Vars: vars, Input: inputArg, NoAppend: true})
}

// copySupportFiles copies everything in a skill bundle except SKILL.md into dst.
func copySupportFiles(bundleDir, dst string) error {
	if bundleDir == "" {
		return nil
	}
	entries, err := os.ReadDir(bundleDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == "SKILL.md" {
			continue
		}
		if err := copyPath(filepath.Join(bundleDir, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// copyPath copies a file or directory tree.
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

// guardClobber refuses to overwrite a target yori didn't create (unless forced).
func guardClobber(target string, prev map[string]bool, force bool) error {
	if force || prev[target] {
		return nil
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("refusing to overwrite %s (not created by yori); pass --force to replace it", target)
	}
	return nil
}

func hasTemplate(body string) bool {
	return strings.Contains(body, "{{") || strings.Contains(body, "{%")
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// --- sync state ---

type state struct {
	// Deployed maps an agent identifier to the target paths yori manages.
	Deployed map[string][]string `json:"deployed"`
}

func loadState(path string) (*state, error) {
	st := &state{Deployed: map[string][]string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("parse sync state %s: %w", path, err)
	}
	if st.Deployed == nil {
		st.Deployed = map[string][]string{}
	}
	return st, nil
}

func (s *state) set(agent string) map[string]bool {
	m := map[string]bool{}
	for _, p := range s.Deployed[agent] {
		m[p] = true
	}
	return m
}

func (s *state) save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
