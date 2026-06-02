package cmd

import (
	"fmt"

	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	whichType   string
	whichGlobal bool
)

var whichCmd = &cobra.Command{
	Use:   "which <name>",
	Short: "Print the resolved file path for an artifact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := store.ParseType(whichType)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		a, err := resolveArtifact(s, typ, args[0], whichGlobal)
		if err != nil {
			return err
		}
		fmt.Println(a.Path)
		return nil
	},
}

func init() {
	addTypeFlag(whichCmd, &whichType)
	whichCmd.Flags().BoolVar(&whichGlobal, "global", false, "read from the global store only")
	rootCmd.AddCommand(whichCmd)
}
