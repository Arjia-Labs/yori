package cmd

import (
	"fmt"

	"github.com/arjia-labs/yori/internal/graph"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var affectedType string

var affectedCmd = &cobra.Command{
	Use:   "affected <name>",
	Short: "Show which artifacts include or extend a partial/base (blast radius)",
	Long: `List every artifact whose composition transitively depends on <name>.

By default <name> is treated as a partial (the common "what uses this shared
block?" case). Pass --type to target a base artifact that others extend.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustStore()
		if err != nil {
			return err
		}

		// Default target is a partial; --type targets a base artifact.
		target := graph.Node{Name: args[0], Partial: true}
		label := "partial " + args[0]
		if cmd.Flags().Changed("type") {
			typ, err := store.ParseType(affectedType)
			if err != nil {
				return err
			}
			target = graph.Node{Type: typ, Name: args[0]}
			label = fmt.Sprintf("%s %q", typ, args[0])
		}

		arts, err := graph.AffectedBy(s, target)
		if err != nil {
			return err
		}
		if len(arts) == 0 {
			fmt.Printf("nothing depends on %s\n", label)
			if target.Partial {
				fmt.Fprintln(cmd.ErrOrStderr(), "(if it's a base artifact others extend, pass --type)")
			}
			return nil
		}
		fmt.Printf("%d artifact(s) depend on %s:\n", len(arts), label)
		for _, a := range arts {
			fmt.Printf("  %s:%s\n", a.Type, a.Name)
		}
		return nil
	},
}

func init() {
	addTypeFlag(affectedCmd, &affectedType)
	rootCmd.AddCommand(affectedCmd)
}
