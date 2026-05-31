package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var whichCmd = &cobra.Command{
	Use:   "which <name>",
	Short: "Print the resolved file path for an artifact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustStore()
		if err != nil {
			return err
		}
		a, err := s.Resolve(args[0])
		if err != nil {
			return err
		}
		fmt.Println(a.Path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whichCmd)
}
