package manifest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// InstallItems clones a registry and installs the named items (and their
// dependency closure) into destStore as editable source.
func InstallItems(url string, names []string, destStore string) ([]*Item, error) {
	return InstallSelected(url, destStore, func(*Manifest) ([]string, error) { return names, nil })
}

// InstallSelected clones a registry, lets pick choose item names from the
// parsed manifest, and installs their dependency closure into destStore.
// Returns the installed items, so callers can tell which are deployable.
func InstallSelected(url, destStore string, pick func(*Manifest) ([]string, error)) ([]*Item, error) {
	repo, cleanup, err := CloneTemp(url)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	data, err := os.ReadFile(filepath.Join(repo, FileName))
	if err != nil {
		return nil, fmt.Errorf("no %s found in %s", FileName, url)
	}
	m, err := Parse(data)
	if err != nil {
		return nil, err
	}
	names, err := pick(m)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}
	items, err := m.Closure(names)
	if err != nil {
		return nil, err
	}

	for _, it := range items {
		for _, f := range it.Files {
			src, err := safeJoin(repo, f)
			if err != nil {
				return nil, err
			}
			dst, err := safeJoin(destStore, f)
			if err != nil {
				return nil, err
			}
			if err := copyFile(src, dst); err != nil {
				return nil, fmt.Errorf("install %s: %w", it.Name, err)
			}
		}
	}
	return items, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
