package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileFromEnv(t *testing.T) {
	const envVar = "TEST_READ_FILE_OR_ENV"

	t.Run("empty", func(t *testing.T) {
		t.Setenv(envVar, "")
		if got := readFileFromEnv(envVar); got != "" {
			t.Fatalf("want empty, got %q", got)
		}
	})

	t.Run("absolute path", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "key.pem")
		if err := os.WriteFile(f, []byte("ABS-PEM"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv(envVar, f)
		if got := readFileFromEnv(envVar); got != "ABS-PEM" {
			t.Fatalf("want ABS-PEM, got %q", got)
		}
	})

	t.Run("relative path resolves to config dir", func(t *testing.T) {
		cfgDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", cfgDir)
		caicDir := filepath.Join(cfgDir, "caic")
		if err := os.MkdirAll(caicDir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(caicDir, "key.pem"), []byte("REL-PEM"), 0o600); err != nil {
			t.Fatal(err)
		}

		t.Setenv(envVar, "key.pem")
		if got := readFileFromEnv(envVar); got != "REL-PEM" {
			t.Fatalf("want REL-PEM, got %q", got)
		}
	})

	t.Run("missing file returns empty", func(t *testing.T) {
		t.Setenv(envVar, "/nonexistent/path/key.pem")
		if got := readFileFromEnv(envVar); got != "" {
			t.Fatalf("want empty, got %q", got)
		}
	})
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("bare tilde", func(t *testing.T) {
		got := expandTilde("~")
		if got != home {
			t.Errorf("expandTilde(~) = %q, want %q", got, home)
		}
	})

	t.Run("tilde with path", func(t *testing.T) {
		got := expandTilde("~/repos")
		want := filepath.Join(home, "repos")
		if got != want {
			t.Errorf("expandTilde(~/repos) = %q, want %q", got, want)
		}
	})

	t.Run("absolute path unchanged", func(t *testing.T) {
		got := expandTilde("/opt/repos")
		if got != "/opt/repos" {
			t.Errorf("expandTilde(/opt/repos) = %q, want /opt/repos", got)
		}
	})

	t.Run("empty string unchanged", func(t *testing.T) {
		got := expandTilde("")
		if got != "" {
			t.Errorf("expandTilde(\"\") = %q, want \"\"", got)
		}
	})

	t.Run("tilde with backslash", func(t *testing.T) {
		got := expandTilde(`~\repos`)
		want := filepath.Join(home, "repos")
		if got != want {
			t.Errorf(`expandTilde(~\repos) = %q, want %q`, got, want)
		}
	})

	t.Run("relative path unchanged", func(t *testing.T) {
		got := expandTilde("repos/foo")
		if got != "repos/foo" {
			t.Errorf("expandTilde(repos/foo) = %q, want repos/foo", got)
		}
	})
}
