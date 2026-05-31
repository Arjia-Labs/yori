// Package store loads and persists Yori artifacts from a layered file store:
// a project store (./.yori/store) shadows a global store (~/.yori/store).
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rovak/yori/internal/config"
)

// ErrNotFound is returned when an artifact cannot be resolved in any layer.
var ErrNotFound = errors.New("artifact not found")

const partialsDir = "partials"

// Store resolves artifacts across the project and global layers.
type Store struct {
	projectStore string // "" when there is no project store
	globalStore  string
}

// New constructs a Store from the current working directory and home dir.
func New() (*Store, error) {
	global, err := config.GlobalStore()
	if err != nil {
		return nil, err
	}
	project, err := config.ProjectStore()
	if err != nil {
		return nil, err
	}
	return &Store{projectStore: project, globalStore: global}, nil
}

// layers returns the store directories in resolution order (project first).
func (s *Store) layers() []layer {
	var ls []layer
	if s.projectStore != "" {
		ls = append(ls, layer{name: "project", dir: s.projectStore})
	}
	ls = append(ls, layer{name: "global", dir: s.globalStore})
	return ls
}

type layer struct {
	name string
	dir  string
}

// StoreDir returns the directory to write to for the given scope.
func (s *Store) StoreDir(global bool) (string, error) {
	if global {
		return s.globalStore, nil
	}
	if s.projectStore == "" {
		return "", fmt.Errorf("no project store found; run `yori init` or use --global")
	}
	return s.projectStore, nil
}

// fileFor returns the file path for an artifact name within a store dir.
func fileFor(dir, name string) string {
	return filepath.Join(dir, name+".md")
}

// FilePath returns the on-disk path an artifact would have in the given scope,
// without requiring it to exist.
func (s *Store) FilePath(name string, global bool) (string, error) {
	dir, err := s.StoreDir(global)
	if err != nil {
		return "", err
	}
	return fileFor(dir, name), nil
}

// Exists reports whether a file exists at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Resolve loads the highest-priority artifact with the given name.
func (s *Store) Resolve(name string) (*Artifact, error) {
	for _, l := range s.layers() {
		path := fileFor(l.dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		a, err := parseArtifact(data, path)
		if err != nil {
			return nil, err
		}
		a.Layer = l.name
		return a, nil
	}
	return nil, fmt.Errorf("%q: %w", name, ErrNotFound)
}

// ReadPartial resolves a partial by base name through the layered stores.
// The lookup ignores any directory/extension on name so Liquid's
// `{% include 'house' %}` and `{% include 'partials/house.md' %}` both work.
func (s *Store) ReadPartial(name string) ([]byte, error) {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	for _, l := range s.layers() {
		path := filepath.Join(l.dir, partialsDir, base+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		return data, nil
	}
	return nil, fmt.Errorf("partial %q: %w", base, ErrNotFound)
}

// List returns artifacts across layers. When global is true, only the global
// layer is listed. Project artifacts shadow same-named global ones. tag, if
// non-empty, filters to artifacts carrying that tag.
func (s *Store) List(global bool, tag string) ([]*Artifact, error) {
	seen := map[string]bool{}
	var out []*Artifact

	ls := s.layers()
	if global {
		ls = []layer{{name: "global", dir: s.globalStore}}
	}

	for _, l := range ls {
		entries, err := os.ReadDir(l.dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			if seen[name] {
				continue // shadowed by a higher layer
			}
			seen[name] = true

			data, err := os.ReadFile(filepath.Join(l.dir, e.Name()))
			if err != nil {
				return nil, err
			}
			a, err := parseArtifact(data, filepath.Join(l.dir, e.Name()))
			if err != nil {
				return nil, err
			}
			a.Layer = l.name
			if tag != "" && !hasTag(a, tag) {
				continue
			}
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func hasTag(a *Artifact, tag string) bool {
	for _, t := range a.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Save writes an artifact's content to the store for the given scope.
func (s *Store) Save(name string, content []byte, global bool) (string, error) {
	dir, err := s.StoreDir(global)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := fileFor(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Delete removes an artifact from the store for the given scope.
func (s *Store) Delete(name string, global bool) error {
	dir, err := s.StoreDir(global)
	if err != nil {
		return err
	}
	path := fileFor(dir, name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%q: %w", name, ErrNotFound)
		}
		return err
	}
	return nil
}

// Init creates the store and partials directories for the given scope.
func (s *Store) Init(global bool) (string, error) {
	dir, err := s.StoreDir(global)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, partialsDir), 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// InitProject creates ./.yori/store (+partials) relative to the working dir,
// used by `yori init` when no project store exists yet.
func InitProject(wd string) (string, error) {
	dir := filepath.Join(wd, config.DirName, "store")
	if err := os.MkdirAll(filepath.Join(dir, partialsDir), 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
