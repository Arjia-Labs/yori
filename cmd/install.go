package cmd

import (
	"fmt"
	"os"

	"github.com/arjia-labs/yori/internal/detect"
	"github.com/arjia-labs/yori/internal/manifest"
	"github.com/arjia-labs/yori/internal/registry"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	installName   string
	installGlobal bool
	installSync   bool
	installAuto   bool
	installAll    bool
	installTags   []string
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
		url, items := resolveRegistry(args[0]), args[1:]

		// Per-item install: explicit names, or --auto/--all/--tag selection.
		if len(items) > 0 || installAuto || installAll || len(installTags) > 0 {
			s, err := mustStore()
			if err != nil {
				return err
			}
			dest, err := s.StoreDir(installGlobal)
			if err != nil {
				return err
			}
			var installed []*manifest.Item
			if len(items) > 0 {
				installed, err = manifest.InstallItems(url, items, dest)
			} else {
				installed, err = manifest.InstallSelected(url, dest, selectItems)
			}
			if err != nil {
				return err
			}
			if len(installed) == 0 {
				fmt.Fprintln(os.Stderr, "no matching items to install")
				return nil
			}
			fmt.Printf("installed %d item(s) into %s:\n", len(installed), dest)
			for _, it := range installed {
				fmt.Printf("  + %s\n", it.Name)
			}
			if installSync {
				return syncInstalled(s, installed)
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

// selectItems picks item names from a manifest per --all / --auto / --tag:
// --all takes everything; --auto keeps items whose `when` matches the detected
// project stack; --tag keeps items carrying a tag. Partials are pulled via the
// dependency closure, not selected directly.
func selectItems(m *manifest.Manifest) ([]string, error) {
	var stack *detect.Stack
	if installAuto {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		stack = detect.Scan(wd)
	}
	var names []string
	for i := range m.Items {
		it := &m.Items[i]
		if it.Type == "partial" {
			continue
		}
		if !installAll {
			if installAuto && !it.When.Matches(stack.Deps, stack.FileExists) {
				continue
			}
			if len(installTags) > 0 && !hasAnyTag(it.Tags, installTags) {
				continue
			}
		}
		names = append(names, it.Name)
	}
	return names, nil
}

func hasAnyTag(tags, want []string) bool {
	for _, t := range tags {
		for _, w := range want {
			if t == w {
				return true
			}
		}
	}
	return false
}

// syncInstalled deploys the deployable items (skills/commands/agents) just
// vendored, to the default agent.
func syncInstalled(s *store.Store, installed []*manifest.Item) error {
	var names []string
	for _, it := range installed {
		switch it.Type {
		case "skill", "command", "agent", "rule":
			names = append(names, it.Name)
		}
	}
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "yori: installed items are prompts/partials; nothing to deploy")
		return nil
	}
	arts, err := gatherForSync(s, installGlobal, names)
	if err != nil {
		return err
	}
	base, statePath, err := syncScope(installGlobal)
	if err != nil {
		return err
	}
	return deployToAgents(s, arts, []string{"claude-code"}, base, statePath, installGlobal, false, false, nil)
}

func init() {
	installCmd.Flags().StringVarP(&installName, "name", "n", "", "package name for whole-repo install (default: derived from URL)")
	installCmd.Flags().BoolVar(&installGlobal, "global", false, "vendor items into the global store (per-item install)")
	installCmd.Flags().BoolVarP(&installSync, "sync", "s", false, "deploy the installed items to your agent after vendoring")
	installCmd.Flags().BoolVar(&installAuto, "auto", false, "install items whose `when` matches the detected project stack")
	installCmd.Flags().BoolVar(&installAll, "all", false, "install every item in the registry")
	installCmd.Flags().StringArrayVar(&installTags, "tag", nil, "install items carrying a tag (repeatable)")
	rootCmd.AddCommand(installCmd)
}
