package cmd

import (
	"fmt"

	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	addGlobal bool
	addType   string
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new artifact and open it in your editor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		typ, err := store.ParseType(addType)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		path, err := s.FilePath(typ, name, addGlobal)
		if err != nil {
			return err
		}
		if store.Exists(path) {
			return fmt.Errorf("%q already exists at %s; use `yori edit`", name, path)
		}
		if _, err := s.Save(typ, name, store.Scaffold(name, typ), addGlobal); err != nil {
			return err
		}
		if err := openEditor(path); err != nil {
			return fmt.Errorf("editor: %w", err)
		}
		fmt.Printf("saved %s\n", path)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVar(&addGlobal, "global", false, "save to the global store")
	addTypeFlag(addCmd, &addType)
	rootCmd.AddCommand(addCmd)
}
