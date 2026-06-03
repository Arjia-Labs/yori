package cmd

import (
	"fmt"

	"github.com/arjia-labs/yori/internal/graph"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	depsType   string
	depsGlobal bool
)

var depsCmd = &cobra.Command{
	Use:   "deps <name>",
	Short: "Show what an artifact composes from (includes + extends)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := store.ParseType(depsType)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		a, err := resolveArtifact(s, typ, args[0], depsGlobal)
		if err != nil {
			return err
		}
		d := graph.DepsOf(s, a)
		if len(d.Bases) == 0 && len(d.Partials) == 0 {
			fmt.Printf("%s %q has no composition dependencies\n", a.Type, a.Name)
			return nil
		}
		fmt.Printf("%s:%s composes from:\n", a.Type, a.Name)
		for _, b := range d.Bases {
			fmt.Printf("  extends  %s:%s\n", b.Type, b.Name)
		}
		for _, p := range d.Partials {
			fmt.Printf("  include  %s\n", p.Name)
		}
		return nil
	},
}

func init() {
	addTypeFlag(depsCmd, &depsType)
	depsCmd.Flags().BoolVar(&depsGlobal, "global", false, "read from the global store only")
	rootCmd.AddCommand(depsCmd)
}
