// Package registry manages installed packages: a git-cloned prompt-set under
// ~/.yori/pkg/<name>, tracked in ~/.yori/registry.yaml. Transport is git,
// shelled out via os/exec — no go-git dependency.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rovak/yori/internal/config"
	"github.com/rovak/yori/internal/ident"
	"gopkg.in/yaml.v3"
)

// Pkg is one installed package.
type Pkg struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Commit string `yaml:"commit"`
}

// Index is the on-disk registry of installed packages.
type Index struct {
	Packages []Pkg `yaml:"packages"`

	path    string // registry.yaml location
	pkgRoot string // ~/.yori/pkg
}

// Load reads the registry index, returning an empty one if none exists yet.
func Load() (*Index, error) {
	path, err := config.RegistryFile()
	if err != nil {
		return nil, err
	}
	pkgRoot, err := config.PkgRoot()
	if err != nil {
		return nil, err
	}
	idx := &Index{path: path, pkgRoot: pkgRoot}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return idx, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, idx); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return idx, nil
}

// Save writes the index back to disk.
func (i *Index) Save() error {
	if err := os.MkdirAll(filepath.Dir(i.path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(struct {
		Packages []Pkg `yaml:"packages"`
	}{i.Packages})
	if err != nil {
		return err
	}
	return os.WriteFile(i.path, out, 0o644)
}

// Dir returns the clone directory for a package name.
func (i *Index) Dir(name string) string {
	return filepath.Join(i.pkgRoot, name)
}

// Find returns the package with the given name, or nil.
func (i *Index) Find(name string) *Pkg {
	for idx := range i.Packages {
		if i.Packages[idx].Name == name {
			return &i.Packages[idx]
		}
	}
	return nil
}

// Dirs returns each installed package's name and clone dir, in order.
func (i *Index) Dirs() []struct{ Name, Dir string } {
	out := make([]struct{ Name, Dir string }, 0, len(i.Packages))
	for _, p := range i.Packages {
		out = append(out, struct{ Name, Dir string }{p.Name, i.Dir(p.Name)})
	}
	return out
}

// Install clones url, records it, and returns the new package. If name is
// empty it is derived from the URL's last path segment.
func (i *Index) Install(url, name string) (*Pkg, error) {
	if name == "" {
		name = NameFromURL(url)
	}
	if name == "" {
		return nil, fmt.Errorf("could not derive a package name from %q; pass --name", url)
	}
	if err := ident.Validate("package", name); err != nil {
		return nil, err
	}
	if i.Find(name) != nil {
		return nil, fmt.Errorf("package %q already installed; use `yori update %s`", name, name)
	}
	dir := i.Dir(name)
	if err := os.MkdirAll(i.pkgRoot, 0o755); err != nil {
		return nil, err
	}
	if err := Clone(url, dir); err != nil {
		return nil, err
	}
	commit, err := HeadCommit(dir)
	if err != nil {
		return nil, err
	}
	p := Pkg{Name: name, URL: url, Commit: commit}
	i.Packages = append(i.Packages, p)
	if err := i.Save(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Uninstall removes a package's clone and index entry.
func (i *Index) Uninstall(name string) error {
	if err := ident.Validate("package", name); err != nil {
		return err
	}
	if i.Find(name) == nil {
		return fmt.Errorf("package %q is not installed", name)
	}
	if err := os.RemoveAll(i.Dir(name)); err != nil {
		return err
	}
	out := i.Packages[:0]
	for _, p := range i.Packages {
		if p.Name != name {
			out = append(out, p)
		}
	}
	i.Packages = out
	return i.Save()
}

// Update pulls a package (or all packages when name is "") and re-pins commits.
func (i *Index) Update(name string) error {
	if name != "" {
		if err := ident.Validate("package", name); err != nil {
			return err
		}
	}
	for idx := range i.Packages {
		p := &i.Packages[idx]
		if name != "" && p.Name != name {
			continue
		}
		commit, err := Pull(i.Dir(p.Name))
		if err != nil {
			return fmt.Errorf("update %s: %w", p.Name, err)
		}
		p.Commit = commit
	}
	if name != "" && i.Find(name) == nil {
		return fmt.Errorf("package %q is not installed", name)
	}
	return i.Save()
}

// NameFromURL derives a package name from a git URL's last path segment.
func NameFromURL(url string) string {
	u := strings.TrimSuffix(strings.TrimRight(url, "/"), ".git")
	if idx := strings.LastIndexAny(u, "/:"); idx >= 0 {
		u = u[idx+1:]
	}
	return u
}
