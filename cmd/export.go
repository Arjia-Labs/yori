package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arjia-labs/yori/internal/export"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	exportType      string
	exportGlobal    bool
	exportProviders []string
)

var exportCmd = &cobra.Command{
	Use:   "export <format> <name>",
	Short: "Export an artifact to another tool's format",
	Long: `Export a yori artifact to another tool. Currently supports promptfoo:

  yori export promptfoo review > promptfooconfig.yaml
  promptfoo eval

The artifact's composition (includes, slots) is resolved, but variables are
left as {{ name }} placeholders for promptfoo to fill. Test cases come from a
sibling file (<name>.cases.yaml, or cases.yaml in a skill bundle) — a YAML list
of promptfoo test objects.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, name := args[0], args[1]
		if format != "promptfoo" {
			return fmt.Errorf("unknown export format %q (supported: promptfoo)", format)
		}
		typ, err := store.ParseType(exportType)
		if err != nil {
			return err
		}
		s, err := mustStore()
		if err != nil {
			return err
		}
		art, err := resolveArtifact(s, typ, name, exportGlobal)
		if err != nil {
			return err
		}

		cases, err := loadCases(art)
		if err != nil {
			return err
		}
		providers := resolveProviders(art)

		out, err := export.Promptfoo(s, art, cases, providers)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(out)
		return err
	},
}

// casesPath returns the sibling cases file for an artifact.
func casesPath(art *store.Artifact) string {
	if art.BundleDir != "" {
		return filepath.Join(art.BundleDir, "cases.yaml")
	}
	return strings.TrimSuffix(art.Path, ".md") + ".cases.yaml"
}

// loadCases reads the artifact's cases file, returning nil (with a note) when
// there isn't one.
func loadCases(art *store.Artifact) ([]export.Case, error) {
	path := casesPath(art)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "yori: no cases file at %s; exporting the prompt only\n", path)
			return nil, nil
		}
		return nil, err
	}
	var cases []export.Case
	if err := yaml.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return cases, nil
}

// resolveProviders picks promptfoo providers from --provider, else the model:
// frontmatter hint, else a placeholder.
func resolveProviders(art *store.Artifact) []string {
	if len(exportProviders) > 0 {
		return exportProviders
	}
	if art.Model != "" {
		return []string{export.GuessProvider(art.Model)}
	}
	fmt.Fprintln(os.Stderr, "yori: no providers; set --provider (e.g. anthropic:claude-sonnet-4-5) or a model: in frontmatter")
	return []string{"REPLACE_ME (set --provider, e.g. anthropic:claude-sonnet-4-5)"}
}

func init() {
	addTypeFlag(exportCmd, &exportType)
	exportCmd.Flags().BoolVar(&exportGlobal, "global", false, "read from the global store only")
	exportCmd.Flags().StringArrayVarP(&exportProviders, "provider", "p", nil, "promptfoo provider (repeatable), e.g. anthropic:claude-sonnet-4-5")
	rootCmd.AddCommand(exportCmd)
}
