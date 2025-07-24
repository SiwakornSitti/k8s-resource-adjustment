package gitops_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s-resource-adjustment/internal/gitops"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// setupTestRepo initializes a new git repository in a temporary directory for testing.
func setupTestRepo(t *testing.T) string {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Create and commit a test file.
	filePath := filepath.Join(dir, "testfile.txt")
	err = os.WriteFile(filePath, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	_, err = w.Add("testfile.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Tester",
			Email: "tester@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	return dir
}

func TestCloneAndWorktree(t *testing.T) {
	dir := setupTestRepo(t)
	repoURL := "file://" + dir
	manager := &gitops.InMemoryGitRepoManager{}

	t.Run("success", func(t *testing.T) {
		_, _, err := manager.CloneAndWorktree(repoURL, "refs/heads/master")
		if err != nil {
			t.Errorf("CloneAndWorktree() unexpected error = %v", err)
		}
	})

	t.Run("non-existent branch", func(t *testing.T) {
		_, _, err := manager.CloneAndWorktree(repoURL, "refs/heads/doesnotexist")
		if err == nil {
			t.Error("CloneAndWorktree() expected an error for non-existent branch, but got nil")
		}
	})
}

func TestGetFile(t *testing.T) {
	dir := setupTestRepo(t)
	repoURL := "file://" + dir
	manager := &gitops.InMemoryGitRepoManager{}

	worktree, _, err := manager.CloneAndWorktree(repoURL, "refs/heads/master")
	if err != nil {
		t.Fatalf("CloneAndWorktree() failed: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		data, err := manager.GetFile(worktree, "testfile.txt")
		if err != nil {
			t.Errorf("GetFile() unexpected error = %v", err)
		}
		expected := "hello world"
		if string(data) != expected {
			t.Errorf("GetFile() got %q, want %q", string(data), expected)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := manager.GetFile(worktree, "nonexistent.txt")
		if err == nil {
			t.Error("GetFile() expected an error for non-existent file, but got nil")
		}
	})
}

func TestCommitAndPush(t *testing.T) {
	dir := setupTestRepo(t)
	repoURL := "file://" + dir
	manager := &gitops.InMemoryGitRepoManager{}

	t.Run("success", func(t *testing.T) {
		worktree, repo, err := manager.CloneAndWorktree(repoURL, "refs/heads/master")
		if err != nil {
			t.Fatalf("CloneAndWorktree() failed: %v", err)
		}

		newFilePath := "newfile.txt"
		f, err := worktree.Filesystem.Create(newFilePath)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		newContent := []byte("new content")
		_, err = f.Write(newContent)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		f.Close()

		err = manager.CommitAndPush(repo, worktree, newFilePath)
		if err != nil {
			t.Errorf("CommitAndPush() unexpected error = %v", err)
		}

		// Verify by cloning again and checking the file
		worktree2, _, err := manager.CloneAndWorktree(repoURL, "refs/heads/master")
		if err != nil {
			t.Fatalf("CloneAndWorktree() for verification failed: %v", err)
		}
		data, err := manager.GetFile(worktree2, newFilePath)
		if err != nil {
			t.Fatalf("GetFile() for verification failed: %v", err)
		}
		if string(data) != string(newContent) {
			t.Errorf("GetFile() after push got %q, want %q", string(data), string(newContent))
		}
	})

	t.Run("error on adding non-existent file", func(t *testing.T) {
		worktree, repo, err := manager.CloneAndWorktree(repoURL, "refs/heads/master")
		if err != nil {
			t.Fatal(err)
		}
		err = manager.CommitAndPush(repo, worktree, "doesnotexist.txt")
		if err == nil {
			t.Errorf("Expected error when adding non-existent file, but got nil")
		}
	})
}
