package cmd

import (
	"fmt"

	"github.com/rovak/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	rmGlobal bool
	rmType   string
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Delete an artifact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := store.ParseType(rmType)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		if err := s.Delete(typ, args[0], rmGlobal); err != nil {
			return err
		}
		scope := "project"
		if rmGlobal {
			scope = "global"
		}
		fmt.Printf("removed %s %s (%s)\n", typ, args[0], scope)
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVar(&rmGlobal, "global", false, "target the global store")
	addTypeFlag(rmCmd, &rmType)
	rootCmd.AddCommand(rmCmd)
}
