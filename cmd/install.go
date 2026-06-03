package cmd

import (
	"fmt"

	"github.com/arjia-labs/yori/internal/manifest"
	"github.com/arjia-labs/yori/internal/registry"
	"github.com/spf13/cobra"
)

var (
	installName   string
	installGlobal bool
)

var installCmd = &cobra.Command{
	Use:   "install <git-url> [item...]",
	Short: "Install a prompt-set, or individual items, from a git repository",
	Long: `With no item names, install the whole repo as a read-only package:
it's shallow-cloned into ~/.yori/pkg/<name>, pinned in ~/.yori/registry.yaml,
and becomes a resolution layer addressable bare or as <pkg>/<name>.

With item names, vendor just those items (and their dependency closure) from
the registry's .yori.json into your store as editable source:

  yori install github.com/acme/prompts security-review commit-message`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url, items := args[0], args[1:]

		// Per-item install: vendor items as editable source into the store.
		if len(items) > 0 {
			s, err := mustStore()
			if err != nil {
				return err
			}
			dest, err := s.StoreDir(installGlobal)
			if err != nil {
				return err
			}
			installed, err := manifest.InstallItems(url, items, dest)
			if err != nil {
				return err
			}
			fmt.Printf("installed %d item(s) into %s:\n", len(installed), dest)
			for _, n := range installed {
				fmt.Printf("  + %s\n", n)
			}
			return nil
		}

		// Whole-repo package install (read-only layer).
		idx, err := registry.Load()
		if err != nil {
			return err
		}
		p, err := idx.Install(url, installName)
		if err != nil {
			return err
		}
		fmt.Printf("installed %s @ %s\n  from %s\n", p.Name, p.Commit, p.URL)
		return nil
	},
}

func init() {
	installCmd.Flags().StringVarP(&installName, "name", "n", "", "package name for whole-repo install (default: derived from URL)")
	installCmd.Flags().BoolVar(&installGlobal, "global", false, "vendor items into the global store (per-item install)")
	rootCmd.AddCommand(installCmd)
}
