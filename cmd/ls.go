package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	lsTag    string
	lsGlobal bool
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List artifacts (project shadows global)",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustStore()
		if err != nil {
			return err
		}
		arts, err := s.List(lsGlobal, lsTag)
		if err != nil {
			return err
		}
		if len(arts) == 0 {
			fmt.Fprintln(os.Stderr, "no artifacts found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tLAYER\tDESCRIPTION")
		for _, a := range arts {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.Layer, a.Description)
		}
		return w.Flush()
	},
}

func init() {
	lsCmd.Flags().StringVar(&lsTag, "tag", "", "filter by tag")
	lsCmd.Flags().BoolVar(&lsGlobal, "global", false, "list only the global store")
	rootCmd.AddCommand(lsCmd)
}
