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
		return pushStoreDir(storeDir, pushRemote, pushMsg)
	},
}

// pushStoreDir commits and pushes a store directory as a git repo, initializing
// it (and origin) on first push. Shared by `yori push` and `yori publish`.
func pushStoreDir(storeDir, remote, msg string) error {
	if !registry.IsRepo(storeDir) {
		if remote == "" {
			return fmt.Errorf("store is not a git repo; pass --remote <url> to initialize")
		}
		if err := registry.InitRepo(storeDir); err != nil {
			return err
		}
		if err := registry.SetRemote(storeDir, remote); err != nil {
			return err
		}
	} else if remote != "" {
		if err := registry.SetRemote(storeDir, remote); err != nil {
			return err
		}
	}

	if msg == "" {
		msg = "Update prompts"
	}
	committed, err := registry.CommitAll(storeDir, msg)
	if err != nil {
		return err
	}
	if !registry.HasCommits(storeDir) {
		return fmt.Errorf("nothing to publish (store is empty)")
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
}

func init() {
	pushCmd.Flags().StringVar(&pushRemote, "remote", "", "git remote URL (sets origin; required on first push)")
	pushCmd.Flags().StringVarP(&pushMsg, "message", "m", "", "commit message (default: \"Update prompts\")")
	rootCmd.AddCommand(pushCmd)
}
