package manifest

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arjia-labs/yori/internal/registry"
)

var ghRe = regexp.MustCompile(`github\.com[:/]+([^/]+)/([^/]+?)(?:\.git)?/?$`)

// FetchRemote returns a registry's manifest without cloning when possible
// (GitHub raw), falling back to a shallow clone for other hosts.
func FetchRemote(url string) ([]byte, error) {
	if m := ghRe.FindStringSubmatch(url); m != nil {
		for _, branch := range []string{"main", "master"} {
			raw := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", m[1], m[2], branch, FileName)
			if data, err := httpGet(raw); err == nil {
				return data, nil
			}
		}
	}
	dir, cleanup, err := CloneTemp(url)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		return nil, fmt.Errorf("no %s found in %s", FileName, url)
	}
	return data, nil
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// CloneTemp shallow-clones url into a temp directory, returning the repo path
// and a cleanup func.
func CloneTemp(url string) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "yori-reg-*")
	if err != nil {
		return "", nil, err
	}
	repo := filepath.Join(tmp, "repo")
	if err := registry.Clone(url, repo); err != nil {
		os.RemoveAll(tmp)
		return "", nil, err
	}
	return repo, func() { os.RemoveAll(tmp) }, nil
}

// safeJoin joins base and a manifest-supplied relative path, refusing any path
// that escapes base. Manifest file paths are remote-controlled, so this guards
// against path-traversal on install.
func safeJoin(base, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	if rel == "" || filepath.IsAbs(rel) {
		return "", fmt.Errorf("unsafe path in manifest: %q", rel)
	}
	base = filepath.Clean(base)
	full := filepath.Join(base, rel) // Join cleans, collapsing any ..
	if full != base && !strings.HasPrefix(full, base+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe path in manifest: %q", rel)
	}
	return full, nil
}
