// Package ident validates user-supplied identifiers (artifact and package
// names) so they are safe to use as single path segments and as unquoted YAML
// scalars. This blocks path traversal (../, absolute paths, nested dirs) and
// characters that would corrupt generated frontmatter.
package ident

import (
	"fmt"
	"regexp"
)

// valid matches a logical name: starts alphanumeric, then alphanumerics plus
// '.', '_', '-'. No slashes, spaces, colons, or leading dots — so it can never
// be ".", "..", an absolute path, or a nested path.
var valid = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Validate reports whether name is a safe identifier of the given kind (used
// only in the error message, e.g. "artifact" or "package").
func Validate(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is empty", kind)
	}
	if !valid.MatchString(name) {
		return fmt.Errorf("invalid %s name %q: use letters, digits, '.', '_' or '-' "+
			"(no slashes, spaces, or leading dots)", kind, name)
	}
	return nil
}

// Valid reports whether name is acceptable, without an error message.
func Valid(name string) bool {
	return valid.MatchString(name)
}
