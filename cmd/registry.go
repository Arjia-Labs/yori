package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arjia-labs/yori/internal/config"
	"github.com/arjia-labs/yori/internal/manifest"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Build and inspect registry manifests (.yori.json)",
}

var (
	buildGlobal   bool
	buildOut      string
	buildName     string
	buildDesc     string
	buildHomepage string
)

var registryBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate a .yori.json manifest from the store",
	Long: `Scan the store and write a .yori.json registry manifest. Each artifact
becomes an installable item; its files and dependencies are inferred from the
composition graph (no hand-authoring). Partials are emitted as items too, so
dependency resolution at install time is a simple recursive lookup.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustStore()
		if err != nil {
			return err
		}
		storeDir, arts, err := buildScope(s, buildGlobal)
		if err != nil {
			return err
		}
		meta := manifest.Meta{Name: buildName, Description: buildDesc, Homepage: buildHomepage}
		if meta.Name == "" {
			meta.Name = defaultRegistryName(storeDir, buildGlobal)
		}
		m, err := manifest.Build(storeDir, arts, meta)
		if err != nil {
			return err
		}
		out, err := m.Bytes()
		if err != nil {
			return err
		}

		target := buildOut
		if target == "" {
			target = filepath.Join(storeDir, manifest.FileName)
		}
		if target == "-" {
			_, err = os.Stdout.Write(out)
			return err
		}
		if err := os.WriteFile(target, out, 0o644); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d items)\n", target, len(m.Items))
		return nil
	},
}

// buildScope returns the store directory to publish and the artifacts in that
// scope's own layer (not the merged view).
func buildScope(s *store.Store, global bool) (string, []*store.Artifact, error) {
	if global {
		dir, err := config.GlobalStore()
		if err != nil {
			return "", nil, err
		}
		arts, err := s.List("", true, "")
		return dir, arts, err
	}
	dir, err := config.ProjectStore()
	if err != nil {
		return "", nil, err
	}
	if dir == "" {
		return "", nil, fmt.Errorf("no project store found; run `yori init` or use --global")
	}
	all, err := s.List("", false, "")
	if err != nil {
		return "", nil, err
	}
	var arts []*store.Artifact
	for _, a := range all {
		if a.Layer == "project" {
			arts = append(arts, a)
		}
	}
	return dir, arts, nil
}

func defaultRegistryName(storeDir string, global bool) string {
	if global {
		return "yori"
	}
	// <root>/.yori/store -> <root> name
	return filepath.Base(filepath.Dir(filepath.Dir(storeDir)))
}

func init() {
	registryBuildCmd.Flags().BoolVar(&buildGlobal, "global", false, "build from the global store")
	registryBuildCmd.Flags().StringVarP(&buildOut, "out", "o", "", "output path (default <store>/.yori.json; '-' for stdout)")
	registryBuildCmd.Flags().StringVar(&buildName, "name", "", "registry name (default: project dir)")
	registryBuildCmd.Flags().StringVar(&buildDesc, "description", "", "registry description")
	registryBuildCmd.Flags().StringVar(&buildHomepage, "homepage", "", "registry homepage URL")
	registryCmd.AddCommand(registryBuildCmd)
	rootCmd.AddCommand(registryCmd)
}
