package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/arjia-labs/yori/internal/manifest"
	"github.com/spf13/cobra"
)

var viewAll bool

var viewCmd = &cobra.Command{
	Use:   "view <registry-url> [item]",
	Short: "Browse a registry's items from its manifest (no clone)",
	Long: `Fetch a registry's .yori.json (raw from GitHub when possible — no clone)
and list its items, or show one item's detail.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := manifest.FetchRemote(resolveRegistry(args[0]))
		if err != nil {
			return err
		}
		m, err := manifest.Parse(data)
		if err != nil {
			return err
		}

		// Detail view for a single item.
		if len(args) == 2 {
			it := m.Find(args[1])
			if it == nil {
				return fmt.Errorf("no item %q in registry %q", args[1], m.Name)
			}
			fmt.Printf("name:         %s\n", it.Name)
			fmt.Printf("type:         %s\n", it.Type)
			if it.Description != "" {
				fmt.Printf("description:  %s\n", it.Description)
			}
			if len(it.Tags) > 0 {
				fmt.Printf("tags:         %s\n", strings.Join(it.Tags, ", "))
			}
			if len(it.Dependencies) > 0 {
				fmt.Printf("dependencies: %s\n", strings.Join(it.Dependencies, ", "))
			}
			fmt.Printf("files:        %s\n", strings.Join(it.Files, ", "))
			fmt.Printf("\ninstall:      yori install %s %s\n", args[0], it.Name)
			return nil
		}

		// List view.
		header := m.Name
		if m.Description != "" {
			header += " — " + m.Description
		}
		fmt.Println(header)
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "TYPE\tNAME\tDESCRIPTION")
		for _, it := range m.Items {
			if it.Type == "partial" && !viewAll {
				continue // partials are dependencies, not headline items
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", it.Type, it.Name, it.Description)
		}
		return w.Flush()
	},
}

func init() {
	viewCmd.Flags().BoolVar(&viewAll, "all", false, "include partials (dependency items)")
	rootCmd.AddCommand(viewCmd)
}
