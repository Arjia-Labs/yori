package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arjia-labs/yori/internal/detect"
	"github.com/spf13/cobra"
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Show the project stack yori detects (for context-aware install)",
	Long: `Read the project's dependency manifests (package.json, go.mod,
pyproject.toml, requirements.txt, Cargo.toml, Gemfile, composer.json) and print
the detected ecosystems and direct dependencies. This is what 'install --auto'
matches an item's 'when' condition against.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		st := detect.Scan(wd)
		if len(st.Deps) == 0 {
			fmt.Fprintf(os.Stderr, "no dependency manifests found in %s\n", wd)
			return nil
		}
		fmt.Printf("ecosystems: %s\n", strings.Join(st.Ecosystems, ", "))
		fmt.Printf("%d direct dependencies:\n", len(st.Deps))
		for _, d := range st.DepList() {
			fmt.Printf("  %s\n", d)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
}
