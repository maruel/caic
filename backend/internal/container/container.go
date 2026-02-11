// Package container wraps md CLI operations for container lifecycle management.
package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Ops abstracts md container lifecycle operations.
type Ops interface {
	Start(ctx context.Context, dir string, labels []string) (name string, err error)
	Diff(ctx context.Context, dir string, args ...string) (string, error)
	Pull(ctx context.Context, dir string) error
	Push(ctx context.Context, dir string) error
	Kill(ctx context.Context, dir string) error
}

// MD implements Ops using the real md CLI.
type MD struct{}

// Start creates and starts an md container for the current branch.
//
// It does not SSH into it (--no-ssh). Labels are passed as --label flags.
func (MD) Start(ctx context.Context, dir string, labels []string) (string, error) {
	args := make([]string, 0, 2+2*len(labels))
	args = append(args, "start", "--no-ssh")
	for _, l := range labels {
		args = append(args, "--label", l)
	}
	cmd := exec.CommandContext(ctx, "md", args...) //nolint:gosec // args are constructed from trusted labels, not user input.
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("md start: %w: %s", err, stderr.String())
	}
	name, err := containerName(ctx, dir)
	if err != nil {
		return "", err
	}
	return name, nil
}

// Diff runs `md diff` and returns the diff output.
func (MD) Diff(ctx context.Context, dir string, args ...string) (string, error) {
	cmdArgs := append([]string{"diff"}, args...)
	cmd := exec.CommandContext(ctx, "md", cmdArgs...) //nolint:gosec // args are not user-controlled.
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("md diff: %w", err)
	}
	return string(out), nil
}

// Pull pulls changes from the container to the local branch.
func (MD) Pull(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "md", "pull")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("md pull: %w: %s", err, stderr.String())
	}
	return nil
}

// Push pushes local changes into the container.
func (MD) Push(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "md", "push")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("md push: %w: %s", err, stderr.String())
	}
	return nil
}

// Kill stops and removes the container.
func (MD) Kill(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "md", "kill")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("md kill: %w: %s", err, stderr.String())
	}
	return nil
}

// LabelValue returns the value of a Docker label on a running container.
//
// Returns empty string if the label is not set.
func LabelValue(ctx context.Context, containerName, label string) (string, error) {
	format := fmt.Sprintf("{{index .Config.Labels %q}}", label)
	cmd := exec.CommandContext(ctx, "docker", "inspect", containerName, "--format", format) //nolint:gosec // containerName and format are not user-controlled.
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect label %q on %s: %w", label, containerName, err)
	}
	v := strings.TrimSpace(string(out))
	if v == "<no value>" {
		return "", nil
	}
	return v, nil
}

// Entry represents a container returned by md list.
type Entry struct {
	Name   string
	Status string
}

// List returns all md containers.
func List(ctx context.Context) ([]Entry, error) {
	cmd := exec.CommandContext(ctx, "md", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("md list: %w", err)
	}
	return parseList(string(out)), nil
}

// parseList parses md list output into entries.
func parseList(raw string) []Entry {
	var entries []Entry
	for line := range strings.SplitSeq(strings.TrimSpace(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasPrefix(fields[0], "md-") {
			entries = append(entries, Entry{Name: fields[0], Status: fields[1]})
		}
	}
	return entries
}

// BranchFromContainer derives the git branch name from a container name by
// stripping the "md-<repo>-" prefix and restoring the "wmao/" prefix that was
// flattened to "wmao-" by md.
func BranchFromContainer(containerName, repoName string) (string, bool) {
	prefix := "md-" + repoName + "-"
	if !strings.HasPrefix(containerName, prefix) {
		return "", false
	}
	slug := containerName[len(prefix):]
	// md replaces "/" with "-", so "wmao/foo" becomes "wmao-foo".
	if strings.HasPrefix(slug, "wmao-") {
		return "wmao/" + slug[len("wmao-"):], true
	}
	return slug, true
}

// containerName returns the md container name for the current repo+branch by
// filtering the global container list to entries matching the repo derived
// from dir.
func containerName(ctx context.Context, dir string) (string, error) {
	entries, err := List(ctx)
	if err != nil {
		return "", err
	}
	repo := filepath.Base(dir)
	prefix := "md-" + repo + "-"
	var match string
	for _, e := range entries {
		if strings.HasPrefix(e.Name, prefix) {
			match = e.Name
		}
	}
	if match == "" {
		return "", errors.New("no md container found for repo " + repo)
	}
	return match, nil
}
