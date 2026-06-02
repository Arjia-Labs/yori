package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest declares which artifacts a repo deploys, and to which agents, so a
// teammate can run a single `yori sync` and get the project's whole agent
// setup. It is meant to be committed (unlike the local sync-state file).
type Manifest struct {
	// Agents are the default target agents (e.g. ["claude-code"]).
	Agents []string `yaml:"agents"`
	// Artifacts are skill/command names to deploy (resolved across both types).
	Artifacts []string `yaml:"artifacts"`
}

// LoadManifest reads a manifest, returning (nil, false) when none exists.
func LoadManifest(path string) (*Manifest, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, false, fmt.Errorf("parse sync manifest %s: %w", path, err)
	}
	if len(m.Agents) == 0 {
		m.Agents = []string{"claude-code"}
	}
	return &m, true, nil
}

// Save writes the manifest to path.
func (m *Manifest) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
