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
	"github.com/rovak/yori/internal/ident"
	"github.com/rovak/yori/internal/registry"
)

// ErrNotFound is returned when an artifact cannot be resolved in any layer.
var ErrNotFound = errors.New("artifact not found")

const partialsDir = "partials"

// Store resolves artifacts across the project, global, and installed-package
// layers (in that priority order).
type Store struct {
	projectStore string // "" when there is no project store
	globalStore  string
	packages     []layer // installed registry packages, read-only
}

// New constructs a Store from the current working directory and home dir,
// including any installed registry packages as read-only layers.
func New() (*Store, error) {
	global, err := config.GlobalStore()
	if err != nil {
		return nil, err
	}
	project, err := config.ProjectStore()
	if err != nil {
		return nil, err
	}
	idx, err := registry.Load()
	if err != nil {
		return nil, err
	}
	var pkgs []layer
	for _, d := range idx.Dirs() {
		pkgs = append(pkgs, layer{name: d.Name, dir: d.Dir, pkg: true})
	}
	return &Store{projectStore: project, globalStore: global, packages: pkgs}, nil
}

// layers returns the store directories in resolution order: project, then
// global, then installed packages.
func (s *Store) layers() []layer {
	var ls []layer
	if s.projectStore != "" {
		ls = append(ls, layer{name: "project", dir: s.projectStore})
	}
	ls = append(ls, layer{name: "global", dir: s.globalStore})
	ls = append(ls, s.packages...)
	return ls
}

type layer struct {
	name string
	dir  string
	pkg  bool // a read-only installed package
}

// findPackage returns the layer for an installed package by name, or nil.
func (s *Store) findPackage(name string) *layer {
	for i := range s.packages {
		if s.packages[i].name == name {
			return &s.packages[i]
		}
	}
	return nil
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

// fileFor returns the file path for a typed artifact within a store dir.
func fileFor(dir string, typ Type, name string) string {
	return filepath.Join(dir, typ.subdir(), name+".md")
}

// FilePath returns the on-disk path an artifact would have in the given scope,
// without requiring it to exist.
func (s *Store) FilePath(typ Type, name string, global bool) (string, error) {
	if err := ident.ValidatePath("artifact", name); err != nil {
		return "", err
	}
	dir, err := s.StoreDir(global)
	if err != nil {
		return "", err
	}
	return fileFor(dir, typ, name), nil
}

// Exists reports whether a file exists at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Resolve loads the highest-priority artifact of the given type and name. A
// name of the form "<pkg>/<name>" resolves only within that installed package.
func (s *Store) Resolve(typ Type, name string) (*Artifact, error) {
	searchLayers := s.layers()
	if pkgName, rest, ok := strings.Cut(name, "/"); ok {
		// The only legal slash is a package qualifier: "<pkg>/<name>".
		l := s.findPackage(pkgName)
		if l == nil {
			return nil, fmt.Errorf("invalid name %q: no installed package %q (names cannot contain '/')", name, pkgName)
		}
		searchLayers = []layer{*l}
		name = rest
	}
	if err := ident.ValidatePath("artifact", name); err != nil {
		return nil, err
	}
	for _, l := range searchLayers {
		path := fileFor(l.dir, typ, name)
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
		a.Type = typ
		if l.pkg {
			a.Package = l.name
		}
		return a, nil
	}
	return nil, fmt.Errorf("%s %q: %w", typ, name, ErrNotFound)
}

// ResolveGlobal loads an artifact only from the global store, ignoring project
// shadows and packages — used by read commands' --global flag to inspect or
// render a shadowed global artifact.
func (s *Store) ResolveGlobal(typ Type, name string) (*Artifact, error) {
	if err := ident.ValidatePath("artifact", name); err != nil {
		return nil, err
	}
	path := fileFor(s.globalStore, typ, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s %q: %w", typ, name, ErrNotFound)
		}
		return nil, err
	}
	a, err := parseArtifact(data, path)
	if err != nil {
		return nil, err
	}
	a.Layer = "global"
	a.Type = typ
	return a, nil
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

// ReadPartialIn resolves a partial only within a specific installed package,
// so an explicitly-addressed package artifact's includes stay self-contained
// and aren't shadowed by same-named project/global partials.
func (s *Store) ReadPartialIn(pkg, name string) ([]byte, error) {
	l := s.findPackage(pkg)
	if l == nil {
		return nil, fmt.Errorf("no installed package %q", pkg)
	}
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	data, err := os.ReadFile(filepath.Join(l.dir, partialsDir, base+".md"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("partial %q in package %q: %w", base, pkg, ErrNotFound)
		}
		return nil, err
	}
	return data, nil
}

// List returns artifacts across layers. An empty typ lists every type; a
// specific typ filters to it. When global is true, only the global layer is
// listed. Higher layers shadow same-(type,name) artifacts in lower ones. tag,
// if non-empty, filters to artifacts carrying that tag.
func (s *Store) List(typ Type, global bool, tag string) ([]*Artifact, error) {
	types := AllTypes
	if typ != "" {
		types = []Type{typ}
	}

	ls := s.layers()
	if global {
		ls = []layer{{name: "global", dir: s.globalStore}}
	}

	seen := map[string]bool{} // key: "<type>/<name>"
	var out []*Artifact

	for _, t := range types {
		for _, l := range ls {
			dir := filepath.Join(l.dir, t.subdir())
			entries, err := os.ReadDir(dir)
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
				key := string(t) + "/" + name
				if seen[key] {
					continue // shadowed by a higher layer
				}
				seen[key] = true

				path := filepath.Join(dir, e.Name())
				data, err := os.ReadFile(path)
				if err != nil {
					return nil, err
				}
				a, err := parseArtifact(data, path)
				if err != nil {
					// One malformed file shouldn't break the whole listing.
					fmt.Fprintf(os.Stderr, "yori: skipping %s: %v\n", path, err)
					continue
				}
				a.Layer = l.name
				a.Type = t
				if tag != "" && !hasTag(a, tag) {
					continue
				}
				out = append(out, a)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Name < out[j].Name
	})
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

// Save writes a typed artifact's content to the store for the given scope.
func (s *Store) Save(typ Type, name string, content []byte, global bool) (string, error) {
	if err := ident.Validate("artifact", name); err != nil {
		return "", err
	}
	dir, err := s.StoreDir(global)
	if err != nil {
		return "", err
	}
	path := fileFor(dir, typ, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Delete removes a typed artifact from the store for the given scope.
func (s *Store) Delete(typ Type, name string, global bool) error {
	if err := ident.ValidatePath("artifact", name); err != nil {
		return err
	}
	dir, err := s.StoreDir(global)
	if err != nil {
		return err
	}
	path := fileFor(dir, typ, name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s %q: %w", typ, name, ErrNotFound)
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
