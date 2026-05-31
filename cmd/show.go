package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rovak/yori/internal/store"
	"github.com/spf13/cobra"
)

var showType string

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Print an artifact's metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		typ, err := store.ParseType(showType)
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
		fmt.Printf("name:        %s\n", a.Name)
		fmt.Printf("type:        %s\n", a.Type)
		if a.Description != "" {
			fmt.Printf("description: %s\n", a.Description)
		}
		if a.Extends != "" {
			fmt.Printf("extends:     %s\n", a.Extends)
		}
		fmt.Printf("layer:       %s\n", a.Layer)
		fmt.Printf("path:        %s\n", a.Path)
		if len(a.Tags) > 0 {
			fmt.Printf("tags:        %s\n", strings.Join(a.Tags, ", "))
		}
		if a.Model != "" {
			fmt.Printf("model:       %s\n", a.Model)
		}
		if len(a.Vars) > 0 {
			fmt.Println("vars:")
			names := make([]string, 0, len(a.Vars))
			for k := range a.Vars {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, n := range names {
				v := a.Vars[n]
				line := "  " + n
				if v.Default != "" {
					line += fmt.Sprintf(" (default: %s)", v.Default)
				}
				if v.Description != "" {
					line += " — " + v.Description
				}
				fmt.Println(line)
			}
		}
		return nil
	},
}

func init() {
	addTypeFlag(showCmd, &showType)
	rootCmd.AddCommand(showCmd)
}
