// Package graph computes the composition dependency graph of yori artifacts:
// the partials an artifact {% include %}s and the base(s) it extends:,
// transitively. It powers `yori deps` (what an artifact composes from),
// `yori affected` (blast radius before editing a shared block), and registry
// manifest generation.
package graph

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arjia-labs/yori/internal/store"
)

// includeRe matches {% include 'name' %} / {% include "name" %}.
var includeRe = regexp.MustCompile(`(?s){%\s*include\s+['"]([^'"]+)['"]\s*%}`)

// Node is a vertex in the graph: a partial, or a typed artifact (a base reached
// via extends).
type Node struct {
	Type    store.Type // empty when Partial
	Name    string
	Partial bool
}

func (n Node) key() string {
	if n.Partial {
		return "partial/" + n.Name
	}
	return string(n.Type) + "/" + n.Name
}

// Deps groups an artifact's transitive composition dependencies.
type Deps struct {
	Bases    []Node // the extends chain (artifacts), in discovery order
	Partials []Node // all transitively-included partials, in discovery order
}

// resolver is the subset of *store.Store the graph needs.
type resolver interface {
	Resolve(typ store.Type, name string) (*store.Artifact, error)
	ReadPartial(name string) ([]byte, error)
	ReadPartialIn(pkg, name string) ([]byte, error)
	List(typ store.Type, global bool, tag string) ([]*store.Artifact, error)
}

// DepsOf returns the transitive composition dependencies of an artifact,
// scoped to its package when it came from one. Missing partials/bases are
// reported as leaf nodes rather than erroring.
func DepsOf(s resolver, a *store.Artifact) Deps {
	pkg := a.Package
	seen := map[string]bool{}
	var bases, partials []Node
	var queue []Node

	enqueue := func(body, extends string, typ store.Type) {
		if extends != "" {
			queue = append(queue, Node{Type: typ, Name: extends})
		}
		for _, m := range includeRe.FindAllStringSubmatch(body, -1) {
			queue = append(queue, Node{Name: partialBase(m[1]), Partial: true})
		}
	}

	enqueue(a.Body, a.Extends, a.Type)
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if seen[n.key()] {
			continue
		}
		seen[n.key()] = true
		if n.Partial {
			partials = append(partials, n)
		} else {
			bases = append(bases, n)
		}
		body, extends, ok := load(s, pkg, n)
		if !ok {
			continue // missing dep: record it, don't expand
		}
		enqueue(body, extends, n.Type)
	}
	return Deps{Bases: bases, Partials: partials}
}

// AffectedBy returns every artifact whose transitive deps include target — the
// blast radius of editing that partial or base.
func AffectedBy(s resolver, target Node) ([]*store.Artifact, error) {
	all, err := s.List("", false, "")
	if err != nil {
		return nil, err
	}
	var out []*store.Artifact
	for _, a := range all {
		d := DepsOf(s, a)
		if dependsOn(d, target) {
			out = append(out, a)
		}
	}
	return out, nil
}

func dependsOn(d Deps, target Node) bool {
	for _, n := range d.Bases {
		if n.key() == target.key() {
			return true
		}
	}
	for _, n := range d.Partials {
		if n.key() == target.key() {
			return true
		}
	}
	return false
}

// load returns a node's body and (for artifacts) its extends, scoped to pkg.
func load(s resolver, pkg string, n Node) (body, extends string, ok bool) {
	if n.Partial {
		var (
			data []byte
			err  error
		)
		if pkg != "" {
			data, err = s.ReadPartialIn(pkg, n.Name)
		} else {
			data, err = s.ReadPartial(n.Name)
		}
		if err != nil {
			return "", "", false
		}
		return string(data), "", true
	}
	name := n.Name
	if pkg != "" {
		name = pkg + "/" + n.Name
	}
	art, err := s.Resolve(n.Type, name)
	if err != nil {
		return "", "", false
	}
	return art.Body, art.Extends, true
}

// partialBase reduces an include reference to a partial name (drop dir + .md),
// matching store.ReadPartial's resolution.
func partialBase(name string) string {
	return strings.TrimSuffix(filepath.Base(name), ".md")
}

// Direct returns an artifact's immediate dependencies: the base it extends (if
// any) and the partials it directly includes. Used for manifest generation,
// where transitivity is recovered by resolving each dependency item.
func Direct(a *store.Artifact) (bases, partials []string) {
	if a.Extends != "" {
		bases = append(bases, a.Extends)
	}
	return bases, DirectIncludes(a.Body)
}

// DirectIncludes returns the partial names a body directly includes.
func DirectIncludes(body string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range includeRe.FindAllStringSubmatch(body, -1) {
		n := partialBase(m[1])
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}
