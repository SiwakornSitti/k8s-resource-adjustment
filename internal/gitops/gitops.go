package gitops

import (
	"io"
	"time"

	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/storage/memory"
)

// GitRepoManager abstracts git operations
type GitRepoManager interface {
	CloneAndWorktree(url, branch string) (*git.Worktree, *git.Repository, error)
	CommitAndPush(repo *git.Repository, worktree *git.Worktree, filePath string) error
	GetFile(worktree *git.Worktree, path string) ([]byte, error)
}

type InMemoryGitRepoManager struct{}

func (g *InMemoryGitRepoManager) CloneAndWorktree(url, branch string) (*git.Worktree, *git.Repository, error) {
	fs := memfs.New()
	repo, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:           url,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName(branch),
	})
	if err != nil {
		return nil, nil, err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, nil, err
	}
	return worktree, repo, nil
}

func (g *InMemoryGitRepoManager) CommitAndPush(repo *git.Repository, worktree *git.Worktree, filePath string) error {
	_, err := worktree.Add(filePath)
	if err != nil {
		return err
	}
	_, err = worktree.Commit("Update set_resources.yaml via automation", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "AutoUpdater",
			Email: "autoupdater@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return repo.Push(&git.PushOptions{})
}

func (g *InMemoryGitRepoManager) GetFile(worktree *git.Worktree, path string) ([]byte, error) {
	file, err := worktree.Filesystem.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}
