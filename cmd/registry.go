package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/arjia-labs/yori/internal/config"
	"github.com/arjia-labs/yori/internal/manifest"
	"github.com/arjia-labs/yori/internal/registry"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Build, inspect, and alias registries",
}

var registryAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a registry alias (use the short name with install/view)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		al, err := registry.LoadAliases()
		if err != nil {
			return err
		}
		if err := al.Add(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("added registry %s -> %s\n", args[0], args[1])
		return nil
	},
}

var registryLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List registry aliases",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		al, err := registry.LoadAliases()
		if err != nil {
			return err
		}
		if len(al.Registries) == 0 {
			fmt.Fprintln(os.Stderr, "no registry aliases (add one with `yori registry add`)")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tURL")
		for _, n := range al.Names() {
			fmt.Fprintf(w, "%s\t%s\n", n, al.Registries[n])
		}
		return w.Flush()
	},
}

var registryRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a registry alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		al, err := registry.LoadAliases()
		if err != nil {
			return err
		}
		if err := al.Remove(args[0]); err != nil {
			return err
		}
		fmt.Printf("removed registry %s\n", args[0])
		return nil
	},
}

// resolveRegistry maps a registry ref to a URL, expanding a known alias.
func resolveRegistry(ref string) string {
	if al, err := registry.LoadAliases(); err == nil {
		if url, ok := al.Resolve(ref); ok {
			return url
		}
	}
	return ref
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
	registryCmd.AddCommand(registryAddCmd)
	registryCmd.AddCommand(registryLsCmd)
	registryCmd.AddCommand(registryRmCmd)
	rootCmd.AddCommand(registryCmd)
}
