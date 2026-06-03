package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/arjia-labs/yori/internal/config"
	"github.com/arjia-labs/yori/internal/ident"
	"gopkg.in/yaml.v3"
)

// Aliases maps short registry names to URLs (~/.yori/registries.yaml), so
// `yori install acme <item>` works instead of a full git URL.
type Aliases struct {
	Registries map[string]string `yaml:"registries"`

	path string
}

// LoadAliases reads the alias map, returning an empty one if none exists.
func LoadAliases() (*Aliases, error) {
	path, err := config.RegistriesFile()
	if err != nil {
		return nil, err
	}
	a := &Aliases{Registries: map[string]string{}, path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, a); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if a.Registries == nil {
		a.Registries = map[string]string{}
	}
	return a, nil
}

// Save writes the alias map.
func (a *Aliases) Save() error {
	if err := os.MkdirAll(filepath.Dir(a.path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(struct {
		Registries map[string]string `yaml:"registries"`
	}{a.Registries})
	if err != nil {
		return err
	}
	return os.WriteFile(a.path, out, 0o644)
}

// Add records a name->url alias (name must be a safe identifier).
func (a *Aliases) Add(name, url string) error {
	if err := ident.Validate("registry", name); err != nil {
		return err
	}
	a.Registries[name] = url
	return a.Save()
}

// Remove deletes an alias.
func (a *Aliases) Remove(name string) error {
	if _, ok := a.Registries[name]; !ok {
		return fmt.Errorf("no registry alias %q", name)
	}
	delete(a.Registries, name)
	return a.Save()
}

// Resolve returns the URL for an alias, or (ref, false) if it isn't one.
func (a *Aliases) Resolve(ref string) (string, bool) {
	url, ok := a.Registries[ref]
	return url, ok
}

// Names returns the alias names, sorted.
func (a *Aliases) Names() []string {
	out := make([]string, 0, len(a.Registries))
	for n := range a.Registries {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
