// Package render fills an artifact's Liquid template with variables and stdin,
// resolving {% include %} partials through the layered store.
package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/osteele/liquid"
	"github.com/rovak/yori/internal/store"
)

// PartialResolver loads a partial's bytes by name (base name, no extension).
type PartialResolver interface {
	ReadPartial(name string) ([]byte, error)
}

// inputRef matches a reference to the `input` variable inside a Liquid
// object ({{ ... }}) or tag ({% ... %}). Used to decide whether piped stdin
// should be substituted in-place or appended to the output.
var inputRef = regexp.MustCompile(`(\{\{|\{%)[^}%]*\binput\b`)

// Options controls a render.
type Options struct {
	// Vars is the assembled variable binding (defaults already overlaid).
	Vars map[string]any
	// Input is stdin / --file content bound to {{ input }}.
	Input string
}

// templateStore adapts a PartialResolver to liquid's TemplateStore, which is
// called for {% include %} with a path; we resolve by base name.
type templateStore struct{ r PartialResolver }

func (t templateStore) ReadTemplate(name string) ([]byte, error) {
	return t.r.ReadPartial(name)
}

// Render fills the artifact body and returns the rendered text. Frontmatter
// var defaults are seeded under opts.Vars; undefined variables render blank
// (Liquid default). If the template references {{ input }} it is substituted
// there; otherwise non-empty input is appended to the output.
func Render(a *store.Artifact, resolver PartialResolver, opts Options) (string, error) {
	engine := liquid.NewEngine()
	engine.RegisterTemplateStore(templateStore{resolver})

	bindings := liquid.Bindings{}
	for k, v := range opts.Vars {
		bindings[k] = v
	}
	bindings["input"] = opts.Input

	tmpl, err := engine.ParseTemplateLocation([]byte(a.Body), a.Path, 1)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", a.Name, err)
	}
	out, err := tmpl.Render(bindings)
	if err != nil {
		return "", fmt.Errorf("render %s: %w", a.Name, err)
	}

	rendered := string(out)
	if opts.Input != "" && !inputRef.MatchString(a.Body) {
		rendered = appendInput(rendered, opts.Input)
	}
	return rendered, nil
}

// appendInput joins rendered output and input with a blank line separator.
func appendInput(rendered, input string) string {
	r := strings.TrimRight(rendered, "\n")
	if r == "" {
		return input
	}
	return r + "\n\n" + input
}
