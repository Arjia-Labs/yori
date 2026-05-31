package cmd

import (
	"fmt"

	"github.com/rovak/yori/internal/registry"
	"github.com/spf13/cobra"
)

var installName string

var installCmd = &cobra.Command{
	Use:   "install <git-url>",
	Short: "Install a prompt-set from a git repository",
	Long: `Install a published prompt-set (a git repo whose root is a Yori store).

The repo is shallow-cloned into ~/.yori/pkg/<name> and pinned in
~/.yori/registry.yaml. Its artifacts become read-only resolution layers
(after project and global), addressable bare or as <pkg>/<name>.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := registry.Load()
		if err != nil {
			return err
		}
		p, err := idx.Install(args[0], installName)
		if err != nil {
			return err
		}
		fmt.Printf("installed %s @ %s\n  from %s\n", p.Name, p.Commit, p.URL)
		return nil
	},
}

func init() {
	installCmd.Flags().StringVarP(&installName, "name", "n", "", "package name (default: derived from URL)")
	rootCmd.AddCommand(installCmd)
}
