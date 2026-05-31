package cmd

import (
	"fmt"

	"github.com/rovak/yori/internal/store"
	"github.com/spf13/cobra"
)

var (
	editGlobal bool
	editType   string
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open an existing artifact in your editor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		typ, err := store.ParseType(editType)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}

		var path string
		if editGlobal {
			path, err = s.FilePath(typ, name, true)
			if err != nil {
				return err
			}
			if !store.Exists(path) {
				return fmt.Errorf("%s %q not found in global store", typ, name)
			}
		} else {
			a, err := s.Resolve(typ, name)
			if err != nil {
				return err
			}
			if a.Package != "" {
				return fmt.Errorf("%q resolves to read-only package %q; copy it into your project or global store before editing", name, a.Package)
			}
			path = a.Path
		}
		return openEditor(path)
	},
}

func init() {
	editCmd.Flags().BoolVar(&editGlobal, "global", false, "edit the global copy")
	addTypeFlag(editCmd, &editType)
	rootCmd.AddCommand(editCmd)
}
