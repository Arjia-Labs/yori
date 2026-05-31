package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Print an artifact's metadata",
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
		fmt.Printf("name:        %s\n", a.Name)
		if a.Description != "" {
			fmt.Printf("description: %s\n", a.Description)
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
	rootCmd.AddCommand(showCmd)
}
