// Package detect inspects a project's dependency manifests to discover its
// stack — the direct dependencies and ecosystems present — so registry items
// can be installed conditionally ("only when `next` is a dependency").
package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Stack is what a project's manifests reveal: its direct dependency identifiers
// and the ecosystems detected.
type Stack struct {
	Dir        string
	Deps       map[string]bool
	Ecosystems []string
}

// Scan reads the dependency manifests in dir.
func Scan(dir string) *Stack {
	s := &Stack{Dir: dir, Deps: map[string]bool{}}
	scanPackageJSON(s)
	scanGoMod(s)
	scanRequirements(s)
	scanComposer(s)
	scanGemfile(s)
	scanPyproject(s)
	scanCargo(s)
	sort.Strings(s.Ecosystems)
	return s
}

// FileExists reports whether any file in the project matches a glob pattern.
func (s *Stack) FileExists(pattern string) bool {
	matches, _ := filepath.Glob(filepath.Join(s.Dir, filepath.FromSlash(pattern)))
	return len(matches) > 0
}

// DepList returns the detected dependencies, sorted.
func (s *Stack) DepList() []string {
	out := make([]string, 0, len(s.Deps))
	for d := range s.Deps {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func (s *Stack) add(eco string, deps ...string) {
	found := false
	for _, d := range deps {
		if d != "" {
			s.Deps[d] = true
			found = true
		}
	}
	if found {
		for _, e := range s.Ecosystems {
			if e == eco {
				return
			}
		}
		s.Ecosystems = append(s.Ecosystems, eco)
	}
}

func (s *Stack) read(name string) ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(s.Dir, name))
	return data, err == nil
}

func scanPackageJSON(s *Stack) {
	data, ok := s.read("package.json")
	if !ok {
		return
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	for d := range pkg.Dependencies {
		s.add("node", d)
	}
	for d := range pkg.DevDependencies {
		s.add("node", d)
	}
}

var goRequireRe = regexp.MustCompile(`(?m)^\s*(?:require\s+)?([a-z0-9.\-/]+\.[a-z0-9.\-/]+)\s+v`)

func scanGoMod(s *Stack) {
	data, ok := s.read("go.mod")
	if !ok {
		return
	}
	for _, m := range goRequireRe.FindAllStringSubmatch(string(data), -1) {
		s.add("go", m[1])
	}
}

func scanRequirements(s *Stack) {
	data, ok := s.read("requirements.txt")
	if !ok {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		name := strings.FieldsFunc(line, func(r rune) bool {
			return strings.ContainsRune("=<>~!;[ \t", r)
		})
		if len(name) > 0 {
			s.add("python", name[0])
		}
	}
}

func scanComposer(s *Stack) {
	data, ok := s.read("composer.json")
	if !ok {
		return
	}
	var c struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if json.Unmarshal(data, &c) != nil {
		return
	}
	for d := range c.Require {
		s.add("php", d)
	}
	for d := range c.RequireDev {
		s.add("php", d)
	}
}

var gemRe = regexp.MustCompile(`(?m)^\s*gem\s+['"]([^'"]+)['"]`)

func scanGemfile(s *Stack) {
	data, ok := s.read("Gemfile")
	if !ok {
		return
	}
	for _, m := range gemRe.FindAllStringSubmatch(string(data), -1) {
		s.add("ruby", m[1])
	}
}

var pyNameRe = regexp.MustCompile(`['"]([A-Za-z0-9._-]+)`)

func scanPyproject(s *Stack) {
	data, ok := s.read("pyproject.toml")
	if !ok {
		return
	}
	content := string(data)
	// PEP 621: dependencies = ["fastapi>=0.1", ...]
	if i := strings.Index(content, "dependencies"); i >= 0 {
		if open := strings.Index(content[i:], "["); open >= 0 {
			if close := strings.Index(content[i+open:], "]"); close >= 0 {
				arr := content[i+open : i+open+close]
				for _, m := range pyNameRe.FindAllStringSubmatch(arr, -1) {
					s.add("python", m[1])
				}
			}
		}
	}
	// Poetry table: [tool.poetry.dependencies]
	for _, k := range tomlTableKeys(content, "tool.poetry.dependencies") {
		if k != "python" {
			s.add("python", k)
		}
	}
}

func scanCargo(s *Stack) {
	data, ok := s.read("Cargo.toml")
	if !ok {
		return
	}
	content := string(data)
	for _, sect := range []string{"dependencies", "dev-dependencies"} {
		for _, k := range tomlTableKeys(content, sect) {
			s.add("rust", k)
		}
	}
}

// tomlTableKeys returns the `key = ...` keys under a [section] table.
func tomlTableKeys(content, section string) []string {
	var out []string
	cur := ""
	for _, line := range strings.Split(content, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			cur = strings.Trim(l, "[]")
			continue
		}
		if cur != section || l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if i := strings.Index(l, "="); i > 0 {
			out = append(out, strings.TrimSpace(l[:i]))
		}
	}
	return out
}
