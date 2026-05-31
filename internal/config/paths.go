// Package config locates Yori's store directories: a global store under
// ~/.yori and an optional per-project store under ./.yori (discovered by
// walking up from the working directory, git-style).
package config

import (
	"os"
	"path/filepath"
)

// DirName is the directory Yori looks for in a project and the home dir.
const DirName = ".yori"

// GlobalRoot returns ~/.yori. It honors $YORI_HOME as an override.
func GlobalRoot() (string, error) {
	if h := os.Getenv("YORI_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName), nil
}

// GlobalStore returns ~/.yori/store.
func GlobalStore() (string, error) {
	root, err := GlobalRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "store"), nil
}

// PkgRoot returns ~/.yori/pkg, where installed registry packages are cloned.
func PkgRoot() (string, error) {
	root, err := GlobalRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "pkg"), nil
}

// RegistryFile returns ~/.yori/registry.yaml, the installed-package index.
func RegistryFile() (string, error) {
	root, err := GlobalRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "registry.yaml"), nil
}

// FindProjectRoot walks up from start looking for a ".yori" directory and
// returns the path to that directory (e.g. /repo/.yori). It returns "" when
// none is found.
func FindProjectRoot(start string) string {
	dir := start
	for {
		candidate := filepath.Join(dir, DirName)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// ProjectStore returns the project's <root>/.yori/store, or "" if there is no
// project store reachable from the working directory.
func ProjectStore() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := FindProjectRoot(wd)
	if root == "" {
		return "", nil
	}
	return filepath.Join(root, "store"), nil
}
