package cmd

import (
	"fmt"

	"github.com/arjia-labs/yori/internal/config"
	"github.com/arjia-labs/yori/internal/registry"
	"github.com/spf13/cobra"
)

var (
	pushRemote string
	pushMsg    string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Publish your global store to a git remote",
	Long: `Publish the global store (~/.yori/store) to a git registry.

The first time, pass --remote <url> to initialize the store as a git repo
and set its origin. Subsequent pushes need no flags. Others install your
published set with: yori install <url>.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		storeDir, err := config.GlobalStore()
		if err != nil {
			return err
		}

		if !registry.IsRepo(storeDir) {
			if pushRemote == "" {
				return fmt.Errorf("global store is not a git repo; pass --remote <url> to initialize")
			}
			if err := registry.InitRepo(storeDir); err != nil {
				return err
			}
			if err := registry.SetRemote(storeDir, pushRemote); err != nil {
				return err
			}
		} else if pushRemote != "" {
			if err := registry.SetRemote(storeDir, pushRemote); err != nil {
				return err
			}
		}

		msg := pushMsg
		if msg == "" {
			msg = "Update prompts"
		}
		committed, err := registry.CommitAll(storeDir, msg)
		if err != nil {
			return err
		}
		if !registry.HasCommits(storeDir) {
			return fmt.Errorf("nothing to publish (global store is empty)")
		}
		if err := registry.Push(storeDir); err != nil {
			return err
		}
		if committed {
			fmt.Printf("published %s\n", storeDir)
		} else {
			fmt.Println("already up to date; pushed")
		}
		return nil
	},
}

func init() {
	pushCmd.Flags().StringVar(&pushRemote, "remote", "", "git remote URL (sets origin; required on first push)")
	pushCmd.Flags().StringVarP(&pushMsg, "message", "m", "", "commit message (default: \"Update prompts\")")
	rootCmd.AddCommand(pushCmd)
}
