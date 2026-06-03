package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arjia-labs/yori/internal/manifest"
	"github.com/spf13/cobra"
)

var (
	publishRemote   string
	publishMsg      string
	publishName     string
	publishDesc     string
	publishHomepage string
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Build the registry manifest and push the global store, in one step",
	Long: `Regenerate .yori.json from the global store, then commit and push it.

One command instead of registry build + git add/commit/push, so the published
manifest can never drift from your prompts. First publish takes --remote <url>.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustStore()
		if err != nil {
			return err
		}
		storeDir, arts, err := buildScope(s, true) // global store is the registry
		if err != nil {
			return err
		}
		meta := manifest.Meta{Name: publishName, Description: publishDesc, Homepage: publishHomepage}
		if meta.Name == "" {
			meta.Name = defaultRegistryName(storeDir, true)
		}
		m, err := manifest.Build(storeDir, arts, meta)
		if err != nil {
			return err
		}
		out, err := m.Bytes()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(storeDir, manifest.FileName), out, 0o644); err != nil {
			return err
		}
		fmt.Printf("built %s (%d items)\n", manifest.FileName, len(m.Items))
		return pushStoreDir(storeDir, publishRemote, publishMsg)
	},
}

func init() {
	publishCmd.Flags().StringVar(&publishRemote, "remote", "", "git remote URL (sets origin; required on first publish)")
	publishCmd.Flags().StringVarP(&publishMsg, "message", "m", "", "commit message")
	publishCmd.Flags().StringVar(&publishName, "name", "", "registry name")
	publishCmd.Flags().StringVar(&publishDesc, "description", "", "registry description")
	publishCmd.Flags().StringVar(&publishHomepage, "homepage", "", "registry homepage URL")
	rootCmd.AddCommand(publishCmd)
}
