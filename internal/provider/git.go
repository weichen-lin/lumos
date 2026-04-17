/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
	"github.com/weichen-lin/lumos/internal/transform"
)

// GitAuth holds the credentials for cloning.
type GitAuth struct {
	// Username + Password (HTTPS basic / token auth).
	// For GitHub/Forgejo tokens: Username = "token", Password = "<token>".
	Username string
	Password string

	// SSHPrivateKey is the PEM-encoded private key for SSH auth.
	SSHPrivateKey []byte
}

// GitProvider fetches config files from a Git repository.
type GitProvider struct {
	url    string
	branch string
	auth   *GitAuth
}

// NewGit creates a GitProvider. auth may be nil for public repos.
func NewGit(url, branch string, auth *GitAuth) *GitProvider {
	if branch == "" {
		branch = "main"
	}
	return &GitProvider{url: url, branch: branch, auth: auth}
}

// Fetch clones the repo into memory and reads the requested files.
func (g *GitProvider) Fetch(_ context.Context, data []syncv1alpha1.ExternalConfigData) (FetchResult, error) {
	var transportAuth transport.AuthMethod
	if g.auth != nil {
		if len(g.auth.SSHPrivateKey) > 0 {
			// SSH key auth
			keys, err := ssh.NewPublicKeys("git", g.auth.SSHPrivateKey, "")
			if err != nil {
				return FetchResult{}, fmt.Errorf("parsing SSH key: %w", err)
			}
			transportAuth = keys
		} else if g.auth.Password != "" {
			// HTTPS basic / token auth
			transportAuth = &http.BasicAuth{
				Username: g.auth.Username,
				Password: g.auth.Password,
			}
		}
	}

	// Clone into memory — no disk I/O required.
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           g.url,
		ReferenceName: plumbing.NewBranchReferenceName(g.branch),
		SingleBranch:  true,
		Depth:         1, // shallow clone; we only need HEAD
		Auth:          transportAuth,
	})
	if err != nil {
		return FetchResult{}, fmt.Errorf("cloning %s: %w", g.url, err)
	}

	// Get HEAD commit SHA.
	head, err := repo.Head()
	if err != nil {
		return FetchResult{}, fmt.Errorf("reading HEAD: %w", err)
	}
	commitSHA := head.Hash().String()

	// Get the commit object to access the file tree.
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return FetchResult{}, fmt.Errorf("reading commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return FetchResult{}, fmt.Errorf("reading tree: %w", err)
	}

	// Read each requested file.
	result := make(map[string]string)
	var mappings []EntryMapping
	for _, d := range data {
		f, err := tree.File(d.Source)
		if err != nil {
			return FetchResult{}, fmt.Errorf("file %q not found in repo: %w", d.Source, err)
		}
		reader, err := f.Reader()
		if err != nil {
			return FetchResult{}, fmt.Errorf("reading file %q: %w", d.Source, err)
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return FetchResult{}, fmt.Errorf("reading file %q: %w", d.Source, err)
		}
		entries, err := transform.Apply(d.Format, d.Key, string(content))
		if err != nil {
			return FetchResult{}, fmt.Errorf("transforming %q: %w", d.Source, err)
		}
		if err := mergeEntries(d.Source, entries, result, &mappings); err != nil {
			return FetchResult{}, err
		}
	}

	return FetchResult{
		Data:     result,
		Version:  commitSHA,
		Mappings: mappings,
	}, nil
}

// FetchRaw clones the repo and returns the raw bytes of each requested file,
// along with the HEAD commit SHA. Used by EncryptedSecretReconciler to get
// SOPS-encrypted file contents before decryption.
func (g *GitProvider) FetchRaw(_ context.Context, sources []string) (files map[string][]byte, version string, err error) {
	var transportAuth transport.AuthMethod
	if g.auth != nil {
		if len(g.auth.SSHPrivateKey) > 0 {
			keys, keyErr := ssh.NewPublicKeys("git", g.auth.SSHPrivateKey, "")
			if keyErr != nil {
				return nil, "", fmt.Errorf("parsing SSH key: %w", keyErr)
			}
			transportAuth = keys
		} else if g.auth.Password != "" {
			transportAuth = &http.BasicAuth{
				Username: g.auth.Username,
				Password: g.auth.Password,
			}
		}
	}

	repo, cloneErr := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           g.url,
		ReferenceName: plumbing.NewBranchReferenceName(g.branch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          transportAuth,
	})
	if cloneErr != nil {
		return nil, "", fmt.Errorf("cloning %s: %w", g.url, cloneErr)
	}

	head, headErr := repo.Head()
	if headErr != nil {
		return nil, "", fmt.Errorf("reading HEAD: %w", headErr)
	}
	commitSHA := head.Hash().String()

	commit, commitErr := repo.CommitObject(head.Hash())
	if commitErr != nil {
		return nil, "", fmt.Errorf("reading commit: %w", commitErr)
	}
	tree, treeErr := commit.Tree()
	if treeErr != nil {
		return nil, "", fmt.Errorf("reading tree: %w", treeErr)
	}

	files = make(map[string][]byte, len(sources))
	for _, src := range sources {
		f, fileErr := tree.File(src)
		if fileErr != nil {
			return nil, "", fmt.Errorf("file %q not found in repo: %w", src, fileErr)
		}
		reader, readerErr := f.Reader()
		if readerErr != nil {
			return nil, "", fmt.Errorf("reading file %q: %w", src, readerErr)
		}
		raw, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			return nil, "", fmt.Errorf("reading file %q: %w", src, readErr)
		}
		files[src] = raw
	}

	return files, commitSHA, nil
}
