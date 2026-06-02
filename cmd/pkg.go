package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/arjia-labs/yori/internal/ident"
	"github.com/arjia-labs/yori/internal/registry"
	"github.com/spf13/cobra"
)

var pkgCmd = &cobra.Command{
	Use:   "pkg",
	Short: "Manage installed registry packages",
}

var pkgLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List installed packages",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := registry.Load()
		if err != nil {
			return err
		}
		if len(idx.Packages) == 0 {
			fmt.Fprintln(os.Stderr, "no packages installed")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCOMMIT\tURL")
		for _, p := range idx.Packages {
			note := ""
			if !ident.Valid(p.Name) {
				note = "\t⚠ invalid name (inactive; run `yori uninstall` to remove)"
			}
			fmt.Fprintf(w, "%s\t%s\t%s%s\n", p.Name, p.Commit, p.URL, note)
		}
		return w.Flush()
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <name>",
	Short: "Remove an installed package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := registry.Load()
		if err != nil {
			return err
		}
		if err := idx.Uninstall(args[0]); err != nil {
			return err
		}
		fmt.Printf("uninstalled %s\n", args[0])
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Pull installed packages and re-pin commits",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := registry.Load()
		if err != nil {
			return err
		}
		var name string
		if len(args) == 1 {
			name = args[0]
		}
		if err := idx.Update(name); err != nil {
			return err
		}
		if name != "" {
			fmt.Printf("updated %s @ %s\n", name, idx.Find(name).Commit)
		} else {
			fmt.Printf("updated %d package(s)\n", len(idx.Packages))
		}
		return nil
	},
}

func init() {
	pkgCmd.AddCommand(pkgLsCmd)
	rootCmd.AddCommand(pkgCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(updateCmd)
}
