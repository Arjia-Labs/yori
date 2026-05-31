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

func TestConditional(t *testing.T) {
	body := "{% if tone %}tone={{ tone }}{% else %}none{% endif %}"
	if got := render(t, body, Options{Vars: map[string]any{"tone": "blunt"}}); got != "tone=blunt" {
		t.Errorf("with tone: %q", got)
	}
	if got := render(t, body, Options{}); got != "none" {
		t.Errorf("without tone: %q", got)
	}
}
