package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"k8s-resource-adjustment/internal/config"
	"k8s-resource-adjustment/internal/git"
	"k8s-resource-adjustment/internal/manifest"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	
	// Check if we have repositories to process
	if len(cfg.Repositories) == 0 {
		log.Fatalf("No repositories configured. Please add repositories to the configuration.")
	}
	
	fmt.Printf("Found %d repositories to process\n", len(cfg.Repositories))

	// Initialize Git client
	fmt.Printf("Initializing Git client with base URL: %s\n", cfg.Git.BaseURL)
	gitClient, err := git.NewClient(cfg.Git.BaseURL)
	if err != nil {
		log.Fatalf("Failed to create Git client: %v", err)
	}

	ctx := context.Background()

	// Process resource adjustments by updating Git manifests for all repositories
	fmt.Println("Processing Kubernetes resource adjustments via GitOps...")
	
	// Iterate through all repositories
	for i, repoPath := range cfg.Repositories {
		fmt.Printf("\n--- Processing repository %d/%d: %s ---\n", i+1, len(cfg.Repositories), repoPath)
		
		// Set repository path for current repository
		gitClient.SetRepository(repoPath)
		
		// Pull latest changes from Git repository
		fmt.Printf("Pulling latest changes from repository: %s\n", repoPath)
		if err := gitClient.Pull(ctx); err != nil {
			log.Printf("Warning: Failed to pull from Git repository: %v", err)
			log.Println("Continuing with local files...")
		}
		
		// Update only resource limits in deployment manifests
		if err := updateResourceManifests(gitClient, cfg); err != nil {
			log.Printf("Failed to update resource manifests for repository %s: %v", repoPath, err)
			continue // Continue with next repository
		}

		// Commit and push changes
		fmt.Printf("Committing changes to repository: %s\n", repoPath)
		if err := gitClient.CommitAndPush(ctx, fmt.Sprintf("GitOps: Auto-adjust Kubernetes resources for %s", repoPath)); err != nil {
			if strings.Contains(err.Error(), "no remote configured") {
				fmt.Printf("Local changes committed successfully. No remote configured for pushing.\n")
			} else if strings.Contains(err.Error(), "invalid remote configuration") {
				fmt.Printf("Local changes committed successfully. Remote configuration needs to be fixed.\n")
			} else {
				log.Printf("Warning: Failed to commit and push changes: %v", err)
			}
		} else {
			fmt.Printf("Changes pushed to %s. ArgoCD will handle the deployment.\n", repoPath)
		}
		
		fmt.Printf("Repository %s processed successfully.\n", repoPath)
	}

	fmt.Println("\nGitOps resource adjustment completed for all repositories!")
}

func switchToRepository(gitClient *git.Client, repoName, repoPath, baseURL string) error {
	remoteURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), repoName)
	return gitClient.ChangeRepoPath(repoPath, remoteURL)
}

func updateResourceManifests(gitClient *git.Client, cfg *config.Config) error {
	resources := cfg.Resources
	fmt.Printf("Updating resource limits in repository\n")
	
	repoPath := gitClient.GetRepoPath()

	manifestPath := filepath.Join(repoPath, "overlays", cfg.Environment, "patches", "set_resource.yaml")
	
	fmt.Printf("Using manifest file: %s\n", manifestPath)
	
	// Convert new ResourcesConfig to manifest.ResourceConfig
	manifestResourceConfig := manifest.ResourceConfig{
		CPU:           &resources.Limits.CPU,
		Memory:        &resources.Limits.Memory,
		CPURequest:    &resources.Requests.CPU,
		MemoryRequest: &resources.Requests.Memory,
		RequestsCPU:   &resources.Requests.CPU,
		RequestsMemory: &resources.Requests.Memory,
		LimitsCPU:     &resources.Limits.CPU,
		LimitsMemory:  &resources.Limits.Memory,
	}
	
	if err := manifest.UpdateResourceLimitsWithStruct(manifestPath, manifestResourceConfig); err != nil {
		return fmt.Errorf("failed to update resource limits in manifest %s: %w", manifestPath, err)
	}
	
	return nil
}
