package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arjia-labs/yori/internal/render"
	"github.com/arjia-labs/yori/internal/store"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <name> [--var=value ...] [--set key=value] [--file path]",
	Short: "Render an artifact: fill variables and inject stdin",
	Long: `Render an artifact to stdout.

Variables come from (highest priority first):
  --key=value         set variable "key" (any name)
  --set key=value     same, for names that clash with reserved flags
  a @path value       reads file contents (e.g. --notes=@notes.md)
  piped stdin         bound to {{ input }} (or appended if not referenced)
  frontmatter default
`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, a := range args {
			if a == "-h" || a == "--help" {
				return cmd.Help()
			}
		}
		p, err := parseRunArgs(args)
		if err != nil {
			return err
		}
		if p.name == "" {
			return fmt.Errorf("missing artifact name")
		}
		typ, err := store.ParseType(p.typ)
		if err != nil {
			return err
		}

		s, err := mustStore()
		if err != nil {
			return err
		}
		art, err := resolveArtifact(s, typ, p.name, p.global)
		if err != nil {
			return err
		}

		// Assemble variables: frontmatter defaults, then user overrides.
		vars := map[string]any{}
		for name, v := range art.Vars {
			if v.Default != "" {
				vars[name] = v.Default
			}
		}
		for k, raw := range p.vars {
			val, err := resolveValue(raw)
			if err != nil {
				return err
			}
			vars[k] = val
		}

		input, err := readInput(p.file)
		if err != nil {
			return err
		}

		out, err := render.Render(art, s, render.Options{Vars: vars, Input: input})
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	},
}

type runParams struct {
	name   string
	typ    string
	vars   map[string]string
	file   string
	global bool
}

// parseRunArgs hand-parses run's args because variable flags are dynamic.
func parseRunArgs(args []string) (runParams, error) {
	p := runParams{vars: map[string]string{}}
	for i := 0; i < len(args); i++ {
		tok := args[i]
		// Short -t for the artifact type: -t agent, -t=agent, or -tagent.
		if tok == "-t" {
			val, ok := takeNext(args, &i)
			if !ok {
				return p, fmt.Errorf("-t requires a type")
			}
			p.typ = val
			continue
		}
		if strings.HasPrefix(tok, "-t") && !strings.HasPrefix(tok, "--") {
			val := strings.TrimPrefix(tok[2:], "=")
			if val == "" {
				return p, fmt.Errorf("-t requires a type")
			}
			p.typ = val
			continue
		}
		if !strings.HasPrefix(tok, "--") {
			if p.name == "" {
				p.name = tok
				continue
			}
			return p, fmt.Errorf("unexpected argument %q", tok)
		}
		key := strings.TrimPrefix(tok, "--")
		var val string
		hasVal := false
		if eq := strings.IndexByte(key, '='); eq >= 0 {
			val, hasVal = key[eq+1:], true
			key = key[:eq]
		}
		// Reserved flags.
		switch key {
		case "type":
			if !hasVal {
				val, hasVal = takeNext(args, &i)
			}
			if !hasVal {
				return p, fmt.Errorf("--type requires a value")
			}
			p.typ = val
			continue
		case "global":
			p.global = true
			continue
		case "file":
			if !hasVal {
				val, hasVal = takeNext(args, &i)
			}
			if !hasVal {
				return p, fmt.Errorf("--file requires a path")
			}
			p.file = val
			continue
		case "set":
			if !hasVal {
				val, hasVal = takeNext(args, &i)
			}
			k, v, ok := strings.Cut(val, "=")
			if !ok {
				return p, fmt.Errorf("--set expects key=value, got %q", val)
			}
			p.vars[k] = v
			continue
		}
		// Dynamic variable flag.
		if !hasVal {
			val, _ = takeNext(args, &i)
		}
		p.vars[key] = val
	}
	return p, nil
}

// takeNext consumes the next arg as a value if it isn't itself a flag.
func takeNext(args []string, i *int) (string, bool) {
	if *i+1 < len(args) && !strings.HasPrefix(args[*i+1], "--") {
		*i++
		return args[*i], true
	}
	return "", false
}

// resolveValue expands an @path value into the file's contents.
func resolveValue(raw string) (string, error) {
	if strings.HasPrefix(raw, "@") {
		data, err := os.ReadFile(raw[1:])
		if err != nil {
			return "", fmt.Errorf("read %s: %w", raw[1:], err)
		}
		return string(data), nil
	}
	return raw, nil
}

// readInput returns --file contents if set, else piped stdin, else "".
func readInput(file string) (string, error) {
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", nil
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return "", nil // interactive terminal, no piped input
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func init() {
	// run hand-parses its args (DisableFlagParsing) to allow dynamic --<var>
	// flags, but we still register the reserved flags so they show up in
	// `yori run --help` and shell completion. Their values are read by
	// parseRunArgs, not by cobra.
	runCmd.Flags().StringP("type", "t", "prompt", "artifact type: prompt|agent|command|skill|rule")
	runCmd.Flags().Bool("global", false, "render the global artifact only")
	runCmd.Flags().String("file", "", "read {{ input }} from a file")
	runCmd.Flags().StringArray("set", nil, "set a variable (key=value), repeatable")
	rootCmd.AddCommand(runCmd)
}
