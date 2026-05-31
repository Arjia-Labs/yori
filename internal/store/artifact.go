package store

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	Vars        map[string]Var `yaml:"vars,omitempty"`

	Body  string `yaml:"-"` // template body, frontmatter stripped
	Path  string `yaml:"-"` // resolved file path on disk
	Layer string `yaml:"-"` // "project" or "global"
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
		Vars        map[string]Var `yaml:"vars,omitempty"`
	}{a.Name, a.Description, a.Tags, a.Model, a.Vars}

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

// Scaffold returns starter content for `yori add`.
func Scaffold(name string) []byte {
	return []byte("---\n" +
		"name: " + name + "\n" +
		"description: \n" +
		"tags: []\n" +
		"vars:\n" +
		"  # example:\n" +
		"  #   default: neutral\n" +
		"  #   description: voice of the response\n" +
		"---\n\n" +
		"Write your prompt here. Use {{ variable }} for variables\n" +
		"and {{ input }} for piped stdin.\n")
}
