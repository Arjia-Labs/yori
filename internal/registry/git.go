package registry

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// git runs a git command in dir and returns trimmed stdout. On failure it
// returns an error carrying git's stderr.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out, errBuf strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(out.String()), nil
}

// NormalizeURL turns a bare host/owner/repo reference (e.g.
// github.com/acme/prompts) into a clonable https URL, leaving full URLs
// (https://, git@, ssh://, file://) and local paths untouched.
func NormalizeURL(url string) string {
	if strings.Contains(url, "://") || strings.HasPrefix(url, "git@") {
		return url
	}
	// host/owner/repo — a host has a dot before the first slash.
	if i := strings.Index(url, "/"); i > 0 && strings.Contains(url[:i], ".") {
		return "https://" + strings.TrimSuffix(url, "/")
	}
	return url
}

// Clone shallow-clones url into dir.
func Clone(url, dir string) error {
	_, err := git("", "clone", "--depth", "1", NormalizeURL(url), dir)
	return err
}

// HeadCommit returns the short HEAD commit of the repo at dir.
func HeadCommit(dir string) (string, error) {
	return git(dir, "rev-parse", "--short", "HEAD")
}

// Pull fast-forwards the repo at dir and returns the new HEAD commit.
func Pull(dir string) (string, error) {
	if _, err := git(dir, "pull", "--ff-only"); err != nil {
		return "", err
	}
	return HeadCommit(dir)
}

// IsRepo reports whether dir is inside a git working tree.
func IsRepo(dir string) bool {
	out, err := git(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// InitRepo initializes a git repo at dir (no-op if already one).
func InitRepo(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if IsRepo(dir) {
		return nil
	}
	_, err := git(dir, "init")
	return err
}

// SetRemote sets origin to url, replacing any existing origin.
func SetRemote(dir, url string) error {
	if cur, _ := git(dir, "remote", "get-url", "origin"); cur != "" {
		_, err := git(dir, "remote", "set-url", "origin", url)
		return err
	}
	_, err := git(dir, "remote", "add", "origin", url)
	return err
}

// HasCommits reports whether the repo at dir has at least one commit.
func HasCommits(dir string) bool {
	_, err := git(dir, "rev-parse", "HEAD")
	return err == nil
}

// HasUpstream reports whether the current branch has an upstream set.
func HasUpstream(dir string) bool {
	_, err := git(dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	return err == nil
}

// CommitAll stages everything and commits with msg. It reports whether a
// commit was actually created (false when there was nothing to commit).
func CommitAll(dir, msg string) (bool, error) {
	if _, err := git(dir, "add", "-A"); err != nil {
		return false, err
	}
	if status, err := git(dir, "status", "--porcelain"); err == nil && status == "" {
		return false, nil // nothing staged
	}
	if _, err := git(dir, "commit", "-m", msg); err != nil {
		return false, err
	}
	return true, nil
}

// Push pushes the current branch to origin, setting upstream on first push.
func Push(dir string) error {
	if HasUpstream(dir) {
		_, err := git(dir, "push")
		return err
	}
	branch, err := git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	_, err = git(dir, "push", "-u", "origin", branch)
	return err
}
