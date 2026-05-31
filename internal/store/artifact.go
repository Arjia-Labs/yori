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
)

// AllTypes lists every artifact type in display order.
var AllTypes = []Type{TypePrompt, TypeAgent, TypeCommand, TypeSkill}

// subdir returns the store subdirectory for a type ("" for prompts).
func (t Type) subdir() string {
	switch t {
	case TypeAgent:
		return "agents"
	case TypeCommand:
		return "commands"
	case TypeSkill:
		return "skills"
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
	default:
		return "", fmt.Errorf("unknown type %q (want prompt|agent|command|skill)", s)
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

	Body  string `yaml:"-"` // template body, frontmatter stripped
	Path  string `yaml:"-"` // resolved file path on disk
	Layer string `yaml:"-"` // "project" or "global"
	Type  Type   `yaml:"-"` // derived from the file's location
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

// Render serializes the artifact back to disk form (frontmatter + body).
func (a *Artifact) Render() ([]byte, error) {
	var buf bytes.Buffer
	// Marshal a copy with only the frontmatter fields populated.
	fm := struct {
		Name        string         `yaml:"name,omitempty"`
		Description string         `yaml:"description,omitempty"`
		Tags        []string       `yaml:"tags,omitempty"`
		Model       string         `yaml:"model,omitempty"`
		Extends     string         `yaml:"extends,omitempty"`
		Vars        map[string]Var `yaml:"vars,omitempty"`
	}{a.Name, a.Description, a.Tags, a.Model, a.Extends, a.Vars}

	out, err := yaml.Marshal(fm)
	if err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.Write(out)
	buf.WriteString("---\n\n")
	buf.WriteString(a.Body)
	return buf.Bytes(), nil
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
