package cmd

import (
	"fmt"

	"github.com/rovak/yori/internal/store"
	"github.com/spf13/cobra"
)

var getType string

var getCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print an artifact's raw body (no rendering)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := store.ParseType(getType)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		a, err := s.Resolve(typ, args[0])
		if err != nil {
			return err
		}
		fmt.Print(a.Body)
		return nil
	},
}

func init() {
	addTypeFlag(getCmd, &getType)
	rootCmd.AddCommand(getCmd)
}
