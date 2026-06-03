// Package cmd implements the yori CLI commands.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "yori",
	Short:         "Yori — the home for everything you tell your AI",
	Long:          "Yori is a local library of reusable AI building blocks — prompts, agents,\nslash-commands, skills, and rules. Store, compose, and render them into ready-to-pipe text.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "yori:", err)
		os.Exit(1)
	}
}

// mustStore builds a Store or returns the error to the command.
func mustStore() (*store.Store, error) {
	return store.New()
}

// addTypeFlag registers the shared --type/-t flag on a command.
func addTypeFlag(cmd *cobra.Command, v *string) {
	cmd.Flags().StringVarP(v, "type", "t", "prompt", "artifact type: prompt|agent|command|skill|rule")
}

// resolveArtifact loads an artifact, honoring --global by restricting the
// lookup to the global store (bypassing project shadows and packages).
func resolveArtifact(s *store.Store, typ store.Type, name string, global bool) (*store.Artifact, error) {
	if global {
		return s.ResolveGlobal(typ, name)
	}
	return s.Resolve(typ, name)
}

// openEditor opens path in the user's editor ($VISUAL, then $EDITOR, then vi),
// wired to the controlling terminal.
func openEditor(path string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command("sh", "-c", editor+" "+shellQuote(path))
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
