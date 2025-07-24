package main

import (
	"fmt"
	"path/filepath"

	"k8s-resource-adjustment/internal/config"
	"k8s-resource-adjustment/internal/gitops"
	"k8s-resource-adjustment/internal/k8s"

	"k8s.io/apimachinery/pkg/api/resource"
)

func main() {
	var (
		configLoader config.ConfigLoader   = &config.EnvConfigLoader{}
		gitManager   gitops.GitRepoManager = &gitops.InMemoryGitRepoManager{}
		patcher      k8s.ResourcePatcher   = &k8s.DefaultResourcePatcher{}
	)

	cfg := configLoader.Load()
	for _, url := range cfg.RepoURLs {
		fmt.Println("======== Processing Repository:", url, "========")
		repoURL := fmt.Sprintf("%s/%s", cfg.BaseURL, url)
		worktree, repo, err := gitManager.CloneAndWorktree(repoURL, cfg.Branch)
		if err != nil {
			fmt.Printf("Failed to clone repo: %v\n", err)
			continue
		}

		targetPath := filepath.Join("overlays", cfg.Env, "patches", "set_resources.yaml")
		file, err := gitManager.GetFile(worktree, targetPath)
		if err != nil {
			fmt.Printf("Failed to read file: %v\n", err)
			continue
		}

		manifest, err := patcher.Patch(file, k8s.ResourceConfig{
			CPURequest: resource.MustParse(cfg.CPURequest),
			MemRequest: resource.MustParse(cfg.MemRequest),
			CPULimit:   resource.MustParse(cfg.CPULimit),
			MemLimit:   resource.MustParse(cfg.MemLimit),
		})
		if err != nil {
			fmt.Printf("Failed to update resource: %v\n", err)
			continue
		}

		f, err := worktree.Filesystem.Create(targetPath)
		if err != nil {
			fmt.Printf("Failed to open file for writing: %v\n", err)
			continue
		}
		_, err = f.Write(manifest)
		if err != nil {
			fmt.Printf("Failed to write updated YAML: %v\n", err)
			f.Close()
			continue
		}
		f.Close()

		err = gitManager.CommitAndPush(repo, worktree, targetPath)
		if err != nil {
			fmt.Printf("Failed to commit/push: %v\n", err)
			continue
		}

		fmt.Printf("Updated file content and pushed to remote!!!\n")
	}
	fmt.Println("======== Finished Processing Repository ========")
}
