package manifest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// InstallItems clones a registry, resolves the dependency closure of the named
// items, and copies their files into destStore as editable source. Returns the
// installed items (closure), so callers can tell which are deployable.
func InstallItems(url string, names []string, destStore string) ([]*Item, error) {
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
