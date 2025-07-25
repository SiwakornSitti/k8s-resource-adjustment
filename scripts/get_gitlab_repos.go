package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jdxcode/netrc"
	"github.com/joho/godotenv"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func getGitLabToken(gitlabBaseURL string) (string, error) {
	// First, try to get the token from the environment variable
	if token := os.Getenv("GITLAB_TOKEN"); token != "" {
		return token, nil
	}

	// If not found, try to get it from the .netrc file
	netrcPath := filepath.Join(os.Getenv("HOME"), ".netrc")
	n, err := netrc.Parse(netrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("GITLAB_TOKEN not set and .netrc file not found at %s", netrcPath)
		}
		return "", fmt.Errorf("error parsing .netrc file: %w", err)
	}

	parsedURL, err := url.Parse(gitlabBaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse GITLAB_BASE_URL: %w", err)
	}
	hostname := parsedURL.Hostname()

	machine := n.Machine(hostname)
	if machine == nil {
		return "", fmt.Errorf("no entry for %s found in .netrc file", hostname)
	}

	return machine.Get("password"), nil
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Get GitLab base URL, default to gitlab.com
	gitlabBaseURL := os.Getenv("GITLAB_BASE_URL")
	if gitlabBaseURL == "" {
		gitlabBaseURL = "https://gitlab.com"
	}

	// Get GitLab token
	gitlabToken, err := getGitLabToken(gitlabBaseURL)
	if err != nil {
		log.Fatal(err)
	}

	groupID := os.Getenv("GITLAB_GROUP_ID")
	if groupID == "" {
		log.Fatal("GITLAB_GROUP_ID environment variable not set")
	}

	// Create a new GitLab client
	git, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabBaseURL))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// List all projects in the group, handling pagination
	var allProjects []*gitlab.Project
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	for {
		projects, resp, err := git.Groups.ListGroupProjects(groupID, opt)
		if err != nil {
			log.Fatalf("Failed to list projects: %v", err)
		}
		allProjects = append(allProjects, projects...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Extract repository URLs
	var repoURLs []string
	for _, project := range allProjects {
		repoURLs = append(repoURLs, project.PathWithNamespace)
	}

	// Write the REPO_URLS back to the .env file
	urlsStr := strings.Join(repoURLs, ",")
	updateEnvFile("REPO_URLS", urlsStr)

	fmt.Printf("Successfully updated REPO_URLS in .env file with %d repositories.\n", len(repoURLs))
}

func updateEnvFile(key, value string) {
	envPath := ".env"
	input, err := os.ReadFile(envPath)
	// If the file doesn't exist, we'll create it. Otherwise, fail on other errors.
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to read .env file: %v", err)
	}

	lines := strings.Split(string(input), "\n")
	found := false
	newLine := fmt.Sprintf("%s=%s", key, value)

	// Using a temporary slice to avoid issues with empty lines at the end of the file.
	var newLines []string
	for _, line := range lines {
		if line != "" { // Skip empty lines from the old file
			newLines = append(newLines, line)
		}
	}
	lines = newLines

	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = newLine
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, newLine)
	}

	output := strings.Join(lines, "\n") + "\n" // Add a trailing newline for POSIX compliance
	err = os.WriteFile(envPath, []byte(output), 0644)
	if err != nil {
		log.Fatalf("Failed to write to .env file: %v", err)
	}
}
