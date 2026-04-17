package provider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gossh "golang.org/x/crypto/ssh"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
)

var (
	testRepoURL    string
	testRepoBranch string
)

// TestMain runs once for the whole provider package.
// It creates a temporary git repo with a few test files,
// then runs all tests, and cleans up afterwards.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "lumos-git-test-*")
	if err != nil {
		panic(err)
	}

	branch, err := setupTestRepo(dir)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}

	testRepoURL = "file://" + dir
	testRepoBranch = branch

	code := m.Run()

	os.RemoveAll(dir)
	os.Exit(code)
}

// setupTestRepo initialises a git repo in dir, writes a few files, commits, and
// returns the branch name that was created (usually "master" in go-git).
func setupTestRepo(dir string) (string, error) {
	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		return "", err
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	files := map[string]string{
		"plain.txt": "hello world",
		"config/app.yaml": `
host: localhost
port: 5432
debug: false
`,
		"config/.env": `
DB_HOST=localhost
DB_PORT=5432
`,
		"config/env.json": `{"API_KEY": "secret", "TIMEOUT": "30s"}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return "", err
		}
		if _, err := w.Add(path); err != nil {
			return "", err
		}
	}

	if _, err := w.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	}); err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	return head.Name().Short(), nil
}

// ── Raw ──────────────────────────────────────────────────────────────────────

func TestGitProvider_Fetch_Raw(t *testing.T) {
	p := NewGit(testRepoURL, testRepoBranch, nil)

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertData(t, result.Data, map[string]string{
		"plain": "hello world",
	})
	if result.Version == "" {
		t.Error("expected non-empty Version (commit SHA)")
	}
}

// ── Env ──────────────────────────────────────────────────────────────────────

func TestGitProvider_Fetch_Env_DotEnv(t *testing.T) {
	p := NewGit(testRepoURL, testRepoBranch, nil)

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "config/.env", Format: syncv1alpha1.FormatEnv},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertData(t, result.Data, map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": "5432",
	})
}

func TestGitProvider_Fetch_Env_JSON(t *testing.T) {
	p := NewGit(testRepoURL, testRepoBranch, nil)

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "config/env.json", Format: syncv1alpha1.FormatEnv},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertData(t, result.Data, map[string]string{
		"API_KEY": "secret",
		"TIMEOUT": "30s",
	})
}

// ── Multiple files ───────────────────────────────────────────────────────────

func TestGitProvider_Fetch_MultipleFiles(t *testing.T) {
	p := NewGit(testRepoURL, testRepoBranch, nil)

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
		{Source: "config/.env", Format: syncv1alpha1.FormatEnv},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertData(t, result.Data, map[string]string{
		"plain":   "hello world",
		"DB_HOST": "localhost",
		"DB_PORT": "5432",
	})
}

// ── Error cases ──────────────────────────────────────────────────────────────

func TestGitProvider_Fetch_MissingFile(t *testing.T) {
	p := NewGit(testRepoURL, testRepoBranch, nil)

	_, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "does-not-exist.txt", Key: "nope", Format: syncv1alpha1.FormatRaw},
	})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestGitProvider_Fetch_WrongBranch(t *testing.T) {
	p := NewGit(testRepoURL, "no-such-branch", nil)

	_, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
	})
	if err == nil {
		t.Fatal("expected error for non-existent branch, got nil")
	}
}

// ── Auth ─────────────────────────────────────────────────────────────────────

func TestGitProvider_NewGit_DefaultBranch(t *testing.T) {
	p := NewGit(testRepoURL, "", nil)
	if p.branch != "main" {
		t.Errorf("expected default branch %q, got %q", "main", p.branch)
	}
}

func TestGitProvider_Fetch_WithHTTPSAuth(t *testing.T) {
	// file:// ignores auth; this exercises the HTTPS auth code path.
	p := NewGit(testRepoURL, testRepoBranch, &GitAuth{
		Username: "token",
		Password: "fake-token",
	})

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	assertData(t, result.Data, map[string]string{"plain": "hello world"})
}

func TestGitProvider_Fetch_WithValidSSHKey(t *testing.T) {
	// file:// ignores auth; this exercises the SSH key auth code path.
	p := NewGit(testRepoURL, testRepoBranch, &GitAuth{
		SSHPrivateKey: generateSSHKey(t),
	})

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	assertData(t, result.Data, map[string]string{"plain": "hello world"})
}

func TestGitProvider_Fetch_WithInvalidSSHKey(t *testing.T) {
	p := NewGit(testRepoURL, testRepoBranch, &GitAuth{
		SSHPrivateKey: []byte("not a valid pem key"),
	})

	_, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
	})
	if err == nil {
		t.Fatal("expected error for invalid SSH key, got nil")
	}
}

// generateSSHKey returns a PEM-encoded ed25519 private key for use in tests.
func generateSSHKey(t *testing.T) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pemBlock, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(pemBlock)
}
