package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	lsTag    string
	lsGlobal bool
	lsType   string
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List artifacts (all types by default; project shadows global)",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var typ store.Type // empty = all types
		if lsType != "" {
			t, err := store.ParseType(lsType)
			if err != nil {
				return err
			}
			typ = t
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		arts, err := s.List(typ, lsGlobal, lsTag)
		if err != nil {
			return err
		}
		if len(arts) == 0 {
			fmt.Fprintln(os.Stderr, "no artifacts found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "TYPE\tNAME\tLAYER\tDESCRIPTION")
		for _, a := range arts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Type, a.Name, a.Layer, a.Description)
		}
		return w.Flush()
	},
}

func init() {
	lsCmd.Flags().StringVar(&lsTag, "tag", "", "filter by tag")
	lsCmd.Flags().BoolVar(&lsGlobal, "global", false, "list only the global store")
	lsCmd.Flags().StringVarP(&lsType, "type", "t", "", "filter by type: prompt|agent|command|skill|rule")
	rootCmd.AddCommand(lsCmd)
}
