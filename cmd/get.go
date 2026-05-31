package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print an artifact's raw body (no rendering)",
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
		fmt.Print(a.Body)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
