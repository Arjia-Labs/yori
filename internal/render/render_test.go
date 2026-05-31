package render

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rovak/yori/internal/store"
)

type fakeResolver map[string]string

// ReadPartial mirrors the real store: resolve by base name, ignoring the
// directory/extension Liquid joins onto the include path.
func (f fakeResolver) ReadPartial(name string) ([]byte, error) {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	if v, ok := f[base]; ok {
		return []byte(v), nil
	}
	return nil, fmt.Errorf("no partial %q", base)
}

// resolverWithArtifacts also resolves named artifacts (for `extends`).
type resolverWithArtifacts struct {
	partials    fakeResolver
	pkgPartials fakeResolver // returned by ReadPartialIn, to verify scoping
	artifacts   map[string]*store.Artifact
}

func (r resolverWithArtifacts) ReadPartial(name string) ([]byte, error) {
	return r.partials.ReadPartial(name)
}

func (r resolverWithArtifacts) ReadPartialIn(_ string, name string) ([]byte, error) {
	return r.pkgPartials.ReadPartial(name)
}

func (f fakeResolver) ReadPartialIn(_ string, name string) ([]byte, error) {
	return f.ReadPartial(name)
}

func (r resolverWithArtifacts) Resolve(_ store.Type, name string) (*store.Artifact, error) {
	if a, ok := r.artifacts[name]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("no artifact %q", name)
}

func (f fakeResolver) Resolve(_ store.Type, name string) (*store.Artifact, error) {
	return nil, fmt.Errorf("no artifact %q", name)
}

func render(t *testing.T, body string, opts Options) string {
	t.Helper()
	a := &store.Artifact{Name: "t", Body: body, Path: "/store/t.md"}
	out, err := Render(a, fakeResolver{}, opts)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out
}

func TestVarSubstitution(t *testing.T) {
	got := render(t, "Hello {{ name }}", Options{Vars: map[string]any{"name": "world"}})
	if got != "Hello world" {
		t.Errorf("got %q", got)
	}
}

func TestMissingVarBlank(t *testing.T) {
	got := render(t, "X{{ missing }}Y", Options{})
	if got != "XY" {
		t.Errorf("got %q", got)
	}
}

func TestDefaultFilter(t *testing.T) {
	got := render(t, `{{ tone | default: "neutral" }}`, Options{})
	if got != "neutral" {
		t.Errorf("got %q", got)
	}
}

func TestInputSubstituted(t *testing.T) {
	got := render(t, "Log:\n{{ input }}", Options{Input: "boom"})
	if got != "Log:\nboom" {
		t.Errorf("got %q", got)
	}
}

func TestInputAppendedWhenNotReferenced(t *testing.T) {
	got := render(t, "Analyze this:", Options{Input: "boom"})
	if got != "Analyze this:\n\nboom" {
		t.Errorf("got %q", got)
	}
}

func TestInputNotAppendedWhenReferenced(t *testing.T) {
	got := render(t, "A {{ input }} B", Options{Input: "x"})
	if got != "A x B" {
		t.Errorf("got %q", got)
	}
}

func TestPartialInclude(t *testing.T) {
	a := &store.Artifact{Name: "t", Body: "{% include 'house' %}\nbody", Path: "/store/t.md"}
	out, err := Render(a, fakeResolver{"house": "Be concise."}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Be concise.") || !strings.Contains(out, "body") {
		t.Errorf("got %q", out)
	}
}

func TestSlotDefaultWhenStandalone(t *testing.T) {
	// A base rendered directly emits its slot defaults.
	got := render(t, `A {% slot "g" %}default-g{% endslot %} B`, Options{})
	if got != "A default-g B" {
		t.Errorf("got %q", got)
	}
}

func TestExtendsFillOverrides(t *testing.T) {
	base := &store.Artifact{Name: "base", Path: "/s/base.md",
		Body: `Intro.
{% slot "guidelines" %}Be concise.{% endslot %}
{% slot "extra" %}none{% endslot %}`}
	child := &store.Artifact{Name: "child", Path: "/s/child.md", Extends: "base",
		Body: `{% fill "guidelines" %}Be verbose.{% endfill %}`}
	r := resolverWithArtifacts{
		partials:  fakeResolver{},
		artifacts: map[string]*store.Artifact{"base": base},
	}
	out, err := Render(child, r, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := "Intro.\nBe verbose.\nnone"
	if out != want {
		t.Errorf("got %q want %q", out, want)
	}
}

func TestExtendsChainAndVars(t *testing.T) {
	grand := &store.Artifact{Name: "grand", Path: "/s/grand.md",
		Body: `Top {% slot "mid" %}mid-default{% endslot %} {% slot "leaf" %}leaf-default{% endslot %}`}
	mid := &store.Artifact{Name: "mid", Path: "/s/mid.md", Extends: "grand",
		Body: `{% fill "mid" %}MID for {{ who }}{% endfill %}`}
	leaf := &store.Artifact{Name: "leaf", Path: "/s/leaf.md", Extends: "mid",
		Body: `{% fill "leaf" %}LEAF{% endfill %}`}
	r := resolverWithArtifacts{
		partials:  fakeResolver{},
		artifacts: map[string]*store.Artifact{"grand": grand, "mid": mid},
	}
	out, err := Render(leaf, r, Options{Vars: map[string]any{"who": "team"}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Top MID for team LEAF" {
		t.Errorf("got %q", out)
	}
}

func TestExtendsCycle(t *testing.T) {
	a := &store.Artifact{Name: "a", Path: "/s/a.md", Extends: "b", Body: ""}
	b := &store.Artifact{Name: "b", Path: "/s/b.md", Extends: "a", Body: ""}
	r := resolverWithArtifacts{
		partials:  fakeResolver{},
		artifacts: map[string]*store.Artifact{"a": a, "b": b},
	}
	if _, err := Render(a, r, Options{}); err == nil {
		t.Errorf("expected cycle error")
	}
}

func TestPackageScopedExtends(t *testing.T) {
	// A package child's `extends: base` must bind to the package's own base,
	// not a same-named project base.
	child := &store.Artifact{Name: "child", Path: "/pkg/child.md", Package: "pkg", Extends: "base",
		Body: `{% fill "g" %}C{% endfill %}`}
	pkgBase := &store.Artifact{Name: "base", Path: "/pkg/base.md", Package: "pkg",
		Body: `PKG {% slot "g" %}d{% endslot %}`}
	projBase := &store.Artifact{Name: "base", Path: "/proj/base.md",
		Body: `PROJ {% slot "g" %}d{% endslot %}`}
	r := resolverWithArtifacts{
		partials:  fakeResolver{},
		artifacts: map[string]*store.Artifact{"pkg/base": pkgBase, "base": projBase},
	}
	out, err := Render(child, r, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "PKG C" {
		t.Errorf("got %q, want package base (\"PKG C\")", out)
	}
}

func TestPackageScopedInclude(t *testing.T) {
	// A package artifact's {% include %} must read the package's partial.
	art := &store.Artifact{Name: "p", Path: "/pkg/p.md", Package: "pkg",
		Body: `{% include 'style' %}`}
	r := resolverWithArtifacts{
		partials:    fakeResolver{"style": "GLOBAL"},
		pkgPartials: fakeResolver{"style": "PKG"},
		artifacts:   map[string]*store.Artifact{},
	}
	out, err := Render(art, r, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "PKG" {
		t.Errorf("got %q, want package partial (\"PKG\")", out)
	}
}

func TestConditional(t *testing.T) {
	body := "{% if tone %}tone={{ tone }}{% else %}none{% endif %}"
	if got := render(t, body, Options{Vars: map[string]any{"tone": "blunt"}}); got != "tone=blunt" {
		t.Errorf("with tone: %q", got)
	}
	if got := render(t, body, Options{}); got != "none" {
		t.Errorf("without tone: %q", got)
	}
}
