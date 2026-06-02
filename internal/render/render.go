// Package render fills an artifact's Liquid template with variables and stdin,
// resolving {% include %} partials through the layered store.
package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/arjia-labs/yori/internal/store"
	"github.com/osteele/liquid"
)

// Resolver loads partials and base artifacts during a render. *store.Store
// satisfies it.
type Resolver interface {
	// ReadPartial loads a partial's bytes (by base name, no extension).
	ReadPartial(name string) ([]byte, error)
	// ReadPartialIn loads a partial only from the named installed package.
	ReadPartialIn(pkg, name string) ([]byte, error)
	// Resolve loads a typed artifact by name (used to follow `extends`).
	Resolve(typ store.Type, name string) (*store.Artifact, error)
}

// inputRef matches a reference to the `input` variable inside a Liquid
// object ({{ ... }}) or tag ({% ... %}). Used to decide whether piped stdin
// should be substituted in-place or appended to the output.
var inputRef = regexp.MustCompile(`(\{\{|\{%)[^}%]*\binput\b`)

// slotRe matches {% slot "name" %}default{% endslot %}.
var slotRe = regexp.MustCompile(`(?s){%\s*slot\s+"?([\w-]+)"?\s*%}(.*?){%\s*endslot\s*%}`)

// fillRe matches {% fill "name" %}body{% endfill %}.
var fillRe = regexp.MustCompile(`(?s){%\s*fill\s+"?([\w-]+)"?\s*%}(.*?){%\s*endfill\s*%}`)

// Options controls a render.
type Options struct {
	// Vars is the assembled variable binding (defaults already overlaid).
	Vars map[string]any
	// Input is stdin / --file content bound to {{ input }}.
	Input string
}

// templateStore adapts a Resolver to liquid's TemplateStore, which is called
// for {% include %} with a path; we resolve by base name. When pkg is set (the
// artifact was addressed within an installed package), includes resolve only
// within that package so the render stays self-contained.
type templateStore struct {
	r   Resolver
	pkg string
}

func (t templateStore) ReadTemplate(name string) ([]byte, error) {
	if t.pkg != "" {
		return t.r.ReadPartialIn(t.pkg, name)
	}
	return t.r.ReadPartial(name)
}

// Render fills the artifact body and returns the rendered text. Frontmatter
// var defaults are seeded under opts.Vars; undefined variables render blank
// (Liquid default). If the template references {{ input }} it is substituted
// there; otherwise non-empty input is appended to the output.
func Render(a *store.Artifact, resolver Resolver, opts Options) (string, error) {
	body, err := resolveInheritance(a, resolver)
	if err != nil {
		return "", err
	}

	engine := liquid.NewEngine()
	engine.RegisterTemplateStore(templateStore{r: resolver, pkg: a.Package})

	bindings := liquid.Bindings{}
	for k, v := range opts.Vars {
		bindings[k] = v
	}
	bindings["input"] = opts.Input

	tmpl, err := engine.ParseTemplateLocation([]byte(body), a.Path, 1)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", a.Name, err)
	}
	out, err := tmpl.Render(bindings)
	if err != nil {
		return "", fmt.Errorf("render %s: %w", a.Name, err)
	}

	rendered := string(out)
	if opts.Input != "" && !inputRef.MatchString(body) {
		rendered = appendInput(rendered, opts.Input)
	}
	return rendered, nil
}

// resolveInheritance applies template inheritance: it pours a child's
// {% fill %} blocks into its base's {% slot %} regions (following a chain of
// `extends`), then closes any remaining slots with their defaults. The result
// contains no slot/fill tags and is ready for Liquid.
func resolveInheritance(a *store.Artifact, resolver Resolver) (string, error) {
	body, err := layout(a, resolver, map[string]bool{})
	if err != nil {
		return "", err
	}
	return closeSlots(body), nil
}

// layout returns an artifact's body with its own fills applied to its base,
// leaving its own (unfilled) slots open for a further child to fill.
func layout(a *store.Artifact, resolver Resolver, visited map[string]bool) (string, error) {
	if a.Extends == "" {
		return a.Body, nil
	}
	if visited[a.Name] {
		return "", fmt.Errorf("extends cycle detected at %q", a.Name)
	}
	visited[a.Name] = true

	// Scope the base lookup to the same package when the child came from one,
	// so package inheritance is self-contained and not shadowed by a
	// same-named project/global base.
	extendsName := a.Extends
	if a.Package != "" {
		extendsName = a.Package + "/" + a.Extends
	}
	base, err := resolver.Resolve(a.Type, extendsName)
	if err != nil {
		return "", fmt.Errorf("%s extends %s: %w", a.Name, a.Extends, err)
	}
	baseBody, err := layout(base, resolver, visited)
	if err != nil {
		return "", err
	}
	return fillOpenSlots(baseBody, extractFills(a.Body)), nil
}

// extractFills collects {% fill name %}body{% endfill %} blocks.
func extractFills(s string) map[string]string {
	fills := map[string]string{}
	for _, m := range fillRe.FindAllStringSubmatch(s, -1) {
		fills[m[1]] = m[2]
	}
	return fills
}

// fillOpenSlots substitutes slots named in fills; unmatched slots stay open.
func fillOpenSlots(body string, fills map[string]string) string {
	return slotRe.ReplaceAllStringFunc(body, func(match string) string {
		sub := slotRe.FindStringSubmatch(match)
		if v, ok := fills[sub[1]]; ok {
			return v
		}
		return match
	})
}

// closeSlots replaces any remaining open slots with their default content.
func closeSlots(body string) string {
	return slotRe.ReplaceAllStringFunc(body, func(match string) string {
		return slotRe.FindStringSubmatch(match)[2]
	})
}

// appendInput joins rendered output and input with a blank line separator.
func appendInput(rendered, input string) string {
	r := strings.TrimRight(rendered, "\n")
	if r == "" {
		return input
	}
	return r + "\n\n" + input
}
