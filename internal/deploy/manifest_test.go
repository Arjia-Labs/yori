package deploy

import (
	"path/filepath"
	"testing"
)

func TestManifestMissing(t *testing.T) {
	_, ok, err := LoadManifest(filepath.Join(t.TempDir(), "sync.yaml"))
	if err != nil || ok {
		t.Errorf("missing manifest: ok=%v err=%v", ok, err)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.yaml")
	m := &Manifest{Agents: []string{"claude-code"}, Artifacts: []string{"researcher", "triage"}}
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadManifest(path)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	if len(got.Artifacts) != 2 || got.Artifacts[0] != "researcher" {
		t.Errorf("artifacts = %v", got.Artifacts)
	}
}

func TestManifestDefaultsAgent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.yaml")
	// A manifest with no agents declared defaults to claude-code.
	if err := (&Manifest{Artifacts: []string{"x"}}).Save(path); err != nil {
		t.Fatal(err)
	}
	got, _, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Agents) != 1 || got.Agents[0] != "claude-code" {
		t.Errorf("default agents = %v", got.Agents)
	}
}
