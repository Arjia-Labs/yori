package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arjia-labs/yori/internal/config"
	"github.com/arjia-labs/yori/internal/deploy"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	syncAgents []string
	syncGlobal bool
	syncLink   bool
	syncForce  bool
	syncSave   bool
	syncSet    []string
)

var syncCmd = &cobra.Command{
	Use:   "sync [names...]",
	Short: "Materialize skills and commands into an agent's discovery directories",
	Long: `Render skills and commands and place them where a coding agent finds them.

By default this targets Claude Code: skills become .claude/skills/<name>/SKILL.md
and commands become .claude/commands/<name>.md. Templates are rendered (variables,
includes, slots) before writing — pass --set key=value to override defaults.

Project scope (default) syncs the project store into ./.claude; --global syncs the
global store into ~/.claude. yori tracks what it wrote, so a later sync prunes
removed artifacts and 'yori unsync' cleans everything up.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := parseSet(syncSet)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		base, statePath, err := syncScope(syncGlobal)
		if err != nil {
			return err
		}
		manifestPath := filepath.Join(filepath.Dir(statePath), "sync.yaml")

		// Decide the target agents and artifact names from flags / manifest.
		agents := syncAgents
		names := args

		switch {
		case syncSave:
			if len(names) == 0 {
				if names, err = allArtifactNames(s, syncGlobal); err != nil {
					return err
				}
			}
			m := &deploy.Manifest{Agents: syncAgents, Artifacts: names}
			if err := m.Save(manifestPath); err != nil {
				return err
			}
			fmt.Printf("saved sync manifest %s\n", manifestPath)
		case len(args) == 0:
			if m, ok, err := deploy.LoadManifest(manifestPath); err != nil {
				return err
			} else if ok {
				names = m.Artifacts
				if !cmd.Flags().Changed("agent") {
					agents = m.Agents // manifest agents unless overridden on the CLI
				}
			}
		}

		arts, err := gatherForSync(s, syncGlobal, names)
		if err != nil {
			return err
		}
		scope := "project"
		if syncGlobal {
			scope = "global"
		}
		for _, agent := range deploy.ExpandAgents(agents) {
			res, err := deploy.Sync(s, arts, deploy.Options{
				Agent:   agent,
				BaseDir: base,
				Global:  syncGlobal,
				State:   statePath,
				Link:    syncLink,
				Force:   syncForce,
				Set:     set,
			})
			if err != nil {
				return err
			}
			fmt.Printf("synced %d artifact(s) to %s (%s)\n", len(res.Written), agent, scope)
			for _, w := range res.Written {
				fmt.Printf("  + %s\n", w)
			}
			for _, p := range res.Pruned {
				fmt.Printf("  - pruned %s\n", p)
			}
			for _, sk := range res.Skipped {
				fmt.Printf("  · skipped %s (unsupported by %s at %s scope)\n", sk, agent, scope)
			}
		}
		return nil
	},
}

// allArtifactNames returns the unique skill/command names in scope (for
// `--save` with no explicit names).
func allArtifactNames(s *store.Store, global bool) ([]string, error) {
	arts, err := gatherForSync(s, global, nil)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for _, a := range arts {
		if !seen[a.Name] {
			seen[a.Name] = true
			names = append(names, a.Name)
		}
	}
	return names, nil
}

var unsyncCmd = &cobra.Command{
	Use:   "unsync",
	Short: "Remove artifacts previously placed by `yori sync`",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, statePath, err := syncScope(syncGlobal)
		if err != nil {
			return err
		}
		for _, agent := range deploy.ExpandAgents(syncAgents) {
			removed, err := deploy.Unsync(deploy.Options{Agent: agent, State: statePath})
			if err != nil {
				return err
			}
			fmt.Printf("removed %d synced artifact(s) for %s\n", len(removed), agent)
			for _, r := range removed {
				fmt.Printf("  - %s\n", r)
			}
		}
		return nil
	},
}

// gatherForSync collects the skills and commands to deploy. Project scope uses
// only the project layer; global scope uses the global store. names, if given,
// filters by artifact name.
func gatherForSync(s *store.Store, global bool, names []string) ([]*store.Artifact, error) {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	matched := map[string]bool{}

	var arts []*store.Artifact
	for _, typ := range []store.Type{store.TypeSkill, store.TypeCommand, store.TypeAgent} {
		list, err := s.List(typ, global, "")
		if err != nil {
			return nil, err
		}
		for _, a := range list {
			if !global && a.Layer != "project" {
				continue // project scope: only the project's own artifacts
			}
			if len(want) > 0 && !want[a.Name] {
				continue
			}
			matched[a.Name] = true
			arts = append(arts, a)
		}
	}
	for n := range want {
		if !matched[n] {
			return nil, fmt.Errorf("no skill, command, or agent named %q to sync", n)
		}
	}
	return arts, nil
}

// syncScope returns the base directory agent dirs live under, plus the path to
// the sync-state file, for the chosen scope.
func syncScope(global bool) (base, statePath string, err error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		groot, err := config.GlobalRoot()
		if err != nil {
			return "", "", err
		}
		return home, filepath.Join(groot, "synced.json"), nil
	}
	root, err := config.ProjectRoot()
	if err != nil {
		return "", "", err
	}
	if root == "" {
		return "", "", fmt.Errorf("no project found; run `yori init` or use --global")
	}
	return root, filepath.Join(root, config.DirName, "synced.json"), nil
}

func parseSet(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, fmt.Errorf("--set expects key=value, got %q", p)
		}
		out[k] = v
	}
	return out, nil
}

func init() {
	for _, c := range []*cobra.Command{syncCmd, unsyncCmd} {
		c.Flags().StringArrayVarP(&syncAgents, "agent", "a", []string{"claude-code"}, "target agent (repeatable, '*' = all): "+strings.Join(deploy.AgentNames(), ", "))
		c.Flags().BoolVar(&syncGlobal, "global", false, "sync the global store into the agent's global dir (~)")
	}
	syncCmd.Flags().BoolVar(&syncLink, "link", false, "symlink static artifacts instead of rendering them")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "overwrite existing files yori didn't create")
	syncCmd.Flags().BoolVar(&syncSave, "save", false, "record the synced artifacts to .yori/sync.yaml")
	syncCmd.Flags().StringArrayVar(&syncSet, "set", nil, "set a template variable (key=value), repeatable")
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(unsyncCmd)
}
