package store

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Type is an artifact kind. Each non-prompt type lives in its own subfolder
// of a store; prompts live at the store root.
type Type string

const (
	TypePrompt  Type = "prompt"
	TypeAgent   Type = "agent"
	TypeCommand Type = "command"
	TypeSkill   Type = "skill"
	TypeRule    Type = "rule"
)

// AllTypes lists every artifact type in display order.
var AllTypes = []Type{TypePrompt, TypeAgent, TypeCommand, TypeSkill, TypeRule}

// subdir returns the store subdirectory for a type ("" for prompts).
func (t Type) subdir() string {
	switch t {
	case TypeAgent:
		return "agents"
	case TypeCommand:
		return "commands"
	case TypeSkill:
		return "skills"
	case TypeRule:
		return "rules"
	default:
		return ""
	}
}

// ParseType maps a user-supplied string (singular or plural) to a Type.
func ParseType(s string) (Type, error) {
	switch s {
	case "", "prompt", "prompts":
		return TypePrompt, nil
	case "agent", "agents":
		return TypeAgent, nil
	case "command", "commands", "cmd":
		return TypeCommand, nil
	case "skill", "skills":
		return TypeSkill, nil
	case "rule", "rules":
		return TypeRule, nil
	default:
		return "", fmt.Errorf("unknown type %q (want prompt|agent|command|skill|rule)", s)
	}
}

// Var is a declared template variable: an optional default and description.
type Var struct {
	Default     string `yaml:"default,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// Artifact is a single prompt: YAML frontmatter plus a Liquid body.
type Artifact struct {
	Name        string         `yaml:"name,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	Model       string         `yaml:"model,omitempty"`
	Extends     string         `yaml:"extends,omitempty"`
	Vars        map[string]Var `yaml:"vars,omitempty"`
	// Extra captures any other frontmatter keys (e.g. allowed-tools, agent,
	// argument-hint, tools, context) so they survive a round-trip and pass
	// through to the agent on deploy.
	Extra map[string]any `yaml:",inline"`

	Body      string `yaml:"-"` // template body, frontmatter stripped
	Path      string `yaml:"-"` // resolved file path on disk
	Layer     string `yaml:"-"` // "project", "global", or a package name
	Type      Type   `yaml:"-"` // derived from the file's location
	Package   string `yaml:"-"` // package name when resolved from an installed package, else ""
	BundleDir string `yaml:"-"` // for a skill bundle (skills/<name>/SKILL.md), its directory; else ""
}

var frontmatterDelim = []byte("---")

// parseArtifact splits optional YAML frontmatter from the body and decodes it.
// A file with no leading "---" block is treated as a pure body. The name
// defaults to the file's base name (without extension) when not set in
// frontmatter.
func parseArtifact(data []byte, path string) (*Artifact, error) {
	a := &Artifact{Path: path}
	fm, body, ok := splitFrontmatter(data)
	if ok {
		if err := yaml.Unmarshal(fm, a); err != nil {
			return nil, fmt.Errorf("parse frontmatter in %s: %w", path, err)
		}
	}
	a.Body = string(body)
	if a.Name == "" {
		base := filepath.Base(path)
		a.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return a, nil
}

// splitFrontmatter returns (frontmatter, body, found). Frontmatter is the
// content between a leading "---" line and the next "---" line.
func splitFrontmatter(data []byte) (fm, body []byte, found bool) {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	// Only treat as frontmatter if the very first line is exactly "---".
	if !bytes.HasPrefix(data, frontmatterDelim) && !bytes.HasPrefix(trimmed, frontmatterDelim) {
		return nil, data, false
	}
	rest := data
	// Drop the opening delimiter line.
	if i := bytes.IndexByte(rest, '\n'); i >= 0 {
		opening := bytes.TrimRight(rest[:i], " \t\r")
		if !bytes.Equal(opening, frontmatterDelim) {
			return nil, data, false
		}
		rest = rest[i+1:]
	} else {
		return nil, data, false
	}
	// Find the closing delimiter line.
	lines := bytes.SplitAfter(rest, []byte("\n"))
	var fmBuf bytes.Buffer
	for i, line := range lines {
		if bytes.Equal(bytes.TrimRight(line, " \t\r\n"), frontmatterDelim) {
			body := bytes.Join(lines[i+1:], nil)
			return fmBuf.Bytes(), body, true
		}
		fmBuf.Write(line)
	}
	// No closing delimiter: treat the whole thing as body.
	return nil, data, false
}

// Render serializes the artifact back to disk form (frontmatter + body). The
// yaml:"-" fields are skipped and Extra (inline) keys are preserved.
func (a *Artifact) Render() ([]byte, error) {
	out, err := yaml.Marshal(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(out)
	buf.WriteString("---\n\n")
	buf.WriteString(a.Body)
	return buf.Bytes(), nil
}

// AgentFrontmatter returns the YAML frontmatter to emit when deploying to an
// agent: the managed fields (description, model, optionally name) plus any
// passthrough Extra keys, but never yori-internal composition (tags, extends,
// vars). Returns nil when there's nothing to emit.
func (a *Artifact) AgentFrontmatter(includeName bool) ([]byte, error) {
	head := struct {
		Name        string `yaml:"name,omitempty"`
		Description string `yaml:"description,omitempty"`
		Model       string `yaml:"model,omitempty"`
	}{Description: a.Description, Model: a.Model}
	if includeName {
		head.Name = a.Name
	}
	out, err := yaml.Marshal(head)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(out)) == "{}" {
		out = nil // all fields empty
	}
	// Pass author frontmatter through, minus yori-internal keys (e.g. `when`,
	// used for registry install selection, not by the agent).
	extra := map[string]any{}
	for k, v := range a.Extra {
		if k == "when" {
			continue
		}
		extra[k] = v
	}
	if len(extra) > 0 {
		ex, err := yaml.Marshal(extra)
		if err != nil {
			return nil, err
		}
		out = append(out, ex...)
	}
	return out, nil
}

// Scaffold returns starter content for `yori add`, tailored to the type.
func Scaffold(name string, typ Type) []byte {
	header := "---\n" +
		"name: " + name + "\n" +
		"description: \n" +
		"tags: []\n"
	switch typ {
	case TypeAgent:
		return []byte(header +
			"model: \n" +
			"vars: {}\n" +
			"---\n\n" +
			"You are " + name + ". Describe the agent's role, tools, and constraints.\n" +
			"Use {{ variable }} for parameters and {{ input }} for the task.\n")
	case TypeCommand:
		return []byte(header +
			"vars: {}\n" +
			"---\n\n" +
			"# /" + name + " command\n\n" +
			"Describe what this slash command does. {{ input }} is the invocation argument.\n")
	case TypeSkill:
		return []byte(header +
			"---\n\n" +
			"# " + name + " skill\n\n" +
			"Describe when to use this skill and the steps it performs.\n")
	case TypeRule:
		return []byte(header +
			"# paths:               # optional: scope this rule to matching files\n" +
			"#   - \"src/**/*.ts\"\n" +
			"---\n\n" +
			"# " + name + "\n\n" +
			"State the rule as direct, checkable guidance. Compose shared blocks\n" +
			"with {% include 'partial' %}.\n")
	default:
		return []byte(header +
			"vars:\n" +
			"  # example:\n" +
			"  #   default: neutral\n" +
			"  #   description: voice of the response\n" +
			"---\n\n" +
			"Write your prompt here. Use {{ variable }} for variables\n" +
			"and {{ input }} for piped stdin.\n")
	}
}
