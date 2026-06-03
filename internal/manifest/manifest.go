// Package manifest builds and consumes a registry manifest (.yori.json): a
// repo-root file that declares the installable items a registry offers — their
// files and dependencies — inferred from the composition graph. It turns
// "clone the whole store as a layer" into "discover and install individual
// items".
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arjia-labs/yori/internal/graph"
	"github.com/arjia-labs/yori/internal/store"
)

// FileName is the manifest's well-known location at a store/repo root.
const FileName = ".yori.json"

// Manifest is the registry manifest.
type Manifest struct {
	Schema      string `json:"$schema,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	Items       []Item `json:"items"`
}

// Item is one installable unit.
type Item struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"` // prompt|agent|command|skill|partial
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Files        []string `json:"files"`                  // paths relative to the store root
	Dependencies []string `json:"dependencies,omitempty"` // in-registry item names
	// Reserved for later phases (declared now so the schema is stable):
	RegistryDependencies []string `json:"registryDependencies,omitempty"` // cross-registry (Phase 2)
	When                 any      `json:"when,omitempty"`                 // context-aware install (clu-8ed346)
}

// Meta is the registry-level metadata for Build.
type Meta struct {
	Name        string
	Description string
	Homepage    string
}

// Build infers a manifest from a store directory and its artifacts. Partials
// are included as items (type "partial") so dependency resolution at install
// time is a simple recursive lookup.
func Build(storeDir string, arts []*store.Artifact, meta Meta) (*Manifest, error) {
	m := &Manifest{
		Schema:      "https://yori.dev/schema/registry-v1.json",
		Name:        meta.Name,
		Description: meta.Description,
		Homepage:    meta.Homepage,
	}

	for _, a := range arts {
		files, err := itemFiles(storeDir, a)
		if err != nil {
			return nil, err
		}
		bases, partials := graph.Direct(a)
		it := Item{
			Name:         a.Name,
			Type:         string(a.Type),
			Description:  a.Description,
			Tags:         a.Tags,
			Files:        files,
			Dependencies: append(append([]string{}, bases...), partials...),
		}
		if w, ok := a.Extra["when"]; ok {
			it.When = w // pass through an author-declared `when:` (eval is a later phase)
		}
		m.Items = append(m.Items, it)
	}

	partials, err := partialItems(storeDir)
	if err != nil {
		return nil, err
	}
	m.Items = append(m.Items, partials...)

	sort.Slice(m.Items, func(i, j int) bool {
		if m.Items[i].Type != m.Items[j].Type {
			return m.Items[i].Type < m.Items[j].Type
		}
		return m.Items[i].Name < m.Items[j].Name
	})
	return m, nil
}

// itemFiles returns an artifact's files relative to storeDir: its own file(s)
// plus a sibling cases file or, for a skill, the whole bundle directory.
func itemFiles(storeDir string, a *store.Artifact) ([]string, error) {
	var files []string
	add := func(abs string) error {
		rel, err := filepath.Rel(storeDir, abs)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}

	if a.BundleDir != "" {
		// A skill bundle: every file under the bundle directory.
		return files, filepath.WalkDir(a.BundleDir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			return add(p)
		})
	}

	if err := add(a.Path); err != nil {
		return nil, err
	}
	// A sibling cases file (<name>.cases.yaml) ships with the prompt.
	cases := strings.TrimSuffix(a.Path, ".md") + ".cases.yaml"
	if store.Exists(cases) {
		if err := add(cases); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// partialItems scans <storeDir>/partials for partial items.
func partialItems(storeDir string) ([]Item, error) {
	dir := filepath.Join(storeDir, "partials")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var items []Item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		items = append(items, Item{
			Name:         name,
			Type:         "partial",
			Files:        []string{"partials/" + e.Name()},
			Dependencies: graph.DirectIncludes(string(body)),
		})
	}
	return items, nil
}

// Bytes serializes the manifest as pretty JSON.
func (m *Manifest) Bytes() ([]byte, error) {
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Parse decodes a manifest from JSON.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// Find returns the item with the given name, or nil.
func (m *Manifest) Find(name string) *Item {
	for i := range m.Items {
		if m.Items[i].Name == name {
			return &m.Items[i]
		}
	}
	return nil
}

// Closure returns the requested items plus their transitive dependency items,
// deduped, suitable for installing as a self-contained set.
func (m *Manifest) Closure(names []string) ([]*Item, error) {
	seen := map[string]bool{}
	var out []*Item
	var visit func(name string) error
	visit = func(name string) error {
		if seen[name] {
			return nil
		}
		it := m.Find(name)
		if it == nil {
			return fmt.Errorf("item %q not found in registry", name)
		}
		seen[name] = true
		out = append(out, it)
		for _, dep := range it.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range names {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return out, nil
}
