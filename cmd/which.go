package cmd

import (
	"fmt"

	"github.com/rovak/yori/internal/store"
	"github.com/spf13/cobra"
)

var whichType string

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
		a, err := s.Resolve(typ, args[0])
		if err != nil {
			return err
		}
		fmt.Println(a.Path)
		return nil
	},
}

func init() {
	addTypeFlag(whichCmd, &whichType)
	rootCmd.AddCommand(whichCmd)
}
