package cmd

import (
	"fmt"
	"os"

	"github.com/rovak/yori/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a project store (./.yori/store)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir, err := store.InitProject(wd)
		if err != nil {
			return err
		}
		fmt.Printf("Initialized Yori store at %s\n", dir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
