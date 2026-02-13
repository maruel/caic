package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maruel/caic/backend/internal/server/dto"
)

func TestCheckSafety(t *testing.T) {
	t.Run("LargeBinary", func(t *testing.T) {
		ctx := t.Context()
		clone := initTestRepo(t, "main")

		// Create a branch with a large binary file.
		runGit(t, clone, "checkout", "-b", "caic/w0")
		data := make([]byte, 600*1024) // 600 KB > 500 KB threshold
		for i := range data {
			data[i] = byte(i % 256)
		}
		if err := os.WriteFile(filepath.Join(clone, "big.bin"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, clone, "add", "big.bin")
		runGit(t, clone, "commit", "-m", "add binary")

		ds := dto.DiffStat{{Path: "big.bin", Binary: true}}
		issues, err := CheckSafety(ctx, clone, "caic/w0", "main", ds)
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 1 {
			t.Fatalf("got %d issues, want 1", len(issues))
		}
		if issues[0].Kind != "large_binary" {
			t.Errorf("kind = %q, want %q", issues[0].Kind, "large_binary")
		}
		if issues[0].File != "big.bin" {
			t.Errorf("file = %q, want %q", issues[0].File, "big.bin")
		}
	})

	t.Run("SmallBinaryOK", func(t *testing.T) {
		ctx := t.Context()
		clone := initTestRepo(t, "main")

		runGit(t, clone, "checkout", "-b", "caic/w0")
		data := make([]byte, 100) // well under threshold
		if err := os.WriteFile(filepath.Join(clone, "small.bin"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, clone, "add", "small.bin")
		runGit(t, clone, "commit", "-m", "add small binary")

		ds := dto.DiffStat{{Path: "small.bin", Binary: true}}
		issues, err := CheckSafety(ctx, clone, "caic/w0", "main", ds)
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 0 {
			t.Errorf("got %d issues, want 0", len(issues))
		}
	})

	t.Run("SecretDetection", func(t *testing.T) {
		ctx := t.Context()
		clone := initTestRepo(t, "main")

		runGit(t, clone, "checkout", "-b", "caic/w0")
		content := "package main\n" + `const awsKey = "AK` + `IAIOSFODNN7EXAMPLE"` + "\n"
		if err := os.WriteFile(filepath.Join(clone, "config.go"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, clone, "add", "config.go")
		runGit(t, clone, "commit", "-m", "add config")

		issues, err := CheckSafety(ctx, clone, "caic/w0", "main", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 1 {
			t.Fatalf("got %d issues, want 1", len(issues))
		}
		if issues[0].Kind != "secret" {
			t.Errorf("kind = %q, want %q", issues[0].Kind, "secret")
		}
		if !strings.Contains(issues[0].Detail, "AWS") {
			t.Errorf("detail = %q, want to contain AWS", issues[0].Detail)
		}
	})

	t.Run("PrivateKey", func(t *testing.T) {
		ctx := t.Context()
		clone := initTestRepo(t, "main")

		runGit(t, clone, "checkout", "-b", "caic/w0")
		content := "-----BEGIN RSA " + "PRIVATE KEY-----\nblahblah\n-----END RSA PRIVATE KEY-----\n"
		if err := os.WriteFile(filepath.Join(clone, "key.pem"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, clone, "add", "key.pem")
		runGit(t, clone, "commit", "-m", "add key")

		issues, err := CheckSafety(ctx, clone, "caic/w0", "main", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 1 {
			t.Fatalf("got %d issues, want 1", len(issues))
		}
		if !strings.Contains(issues[0].Detail, "private key") {
			t.Errorf("detail = %q, want to contain 'private key'", issues[0].Detail)
		}
	})

	t.Run("HardcodedCredential", func(t *testing.T) {
		ctx := t.Context()
		clone := initTestRepo(t, "main")

		runGit(t, clone, "checkout", "-b", "caic/w0")
		content := "pass" + `word = "supersecretpassword123"` + "\n"
		if err := os.WriteFile(filepath.Join(clone, "app.conf"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, clone, "add", "app.conf")
		runGit(t, clone, "commit", "-m", "add config")

		issues, err := CheckSafety(ctx, clone, "caic/w0", "main", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 1 {
			t.Fatalf("got %d issues, want 1", len(issues))
		}
		if !strings.Contains(issues[0].Detail, "credential") {
			t.Errorf("detail = %q, want to contain 'credential'", issues[0].Detail)
		}
	})

	t.Run("NoIssues", func(t *testing.T) {
		ctx := t.Context()
		clone := initTestRepo(t, "main")

		runGit(t, clone, "checkout", "-b", "caic/w0")
		if err := os.WriteFile(filepath.Join(clone, "clean.go"), []byte("package clean\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, clone, "add", "clean.go")
		runGit(t, clone, "commit", "-m", "add clean")

		ds := dto.DiffStat{{Path: "clean.go", Added: 1}}
		issues, err := CheckSafety(ctx, clone, "caic/w0", "main", ds)
		if err != nil {
			t.Fatal(err)
		}
		if len(issues) != 0 {
			t.Errorf("got %d issues, want 0: %+v", len(issues), issues)
		}
	})
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{500 * 1024, "500 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
	}
	for _, tt := range tests {
		got := humanSize(tt.in)
		if got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestScanDiffForSecrets_Deduplication(t *testing.T) {
	ctx := t.Context()
	clone := initTestRepo(t, "main")

	runGit(t, clone, "checkout", "-b", "caic/w0")
	// Multiple AWS keys in the same file should produce only one issue.
	content := "key1 = \"AK" + "IAIOSFODNN7EXAMPLE\"\nkey2 = \"AK" + "IAIOSFODNN7EXAMPLE\"\n"
	if err := os.WriteFile(filepath.Join(clone, "keys.go"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, clone, "add", "keys.go")
	runGit(t, clone, "commit", "-m", "add keys")

	issues, err := scanDiffForSecrets(ctx, clone, "caic/w0", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Errorf("got %d issues, want 1 (deduplication)", len(issues))
	}
}

// initTestRepo and runGit are defined in runner_test.go (same package).
