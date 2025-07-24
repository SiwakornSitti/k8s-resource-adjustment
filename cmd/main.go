package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/joho/godotenv"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

type Config struct {
	Env        string
	BaseURL    string
	Branch     string
	RepoURLs   []string
	CPULimit   string
	MemLimit   string
	CPURequest string
	MemRequest string
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getConfigFromEnv() Config {
	_ = godotenv.Load()

	repoURLs := getEnv("REPO_URLS", "__URL_1__,__URL_2__")
	urls := []string{}
	for _, url := range strings.Split(repoURLs, ",") {
		urls = append(urls, strings.TrimSpace(url))
	}

	return Config{
		Env:        getEnv("ENV", "__ENV__"),
		BaseURL:    getEnv("BASE_URL", "__GIT_URL__"),
		Branch:     getEnv("BRANCH", "__BRANCH__"),
		RepoURLs:   urls,
		CPULimit:   getEnv("CPU_LIMIT", "20m"),
		MemLimit:   getEnv("MEM_LIMIT", "32Mi"),
		CPURequest: getEnv("CPU_REQUEST", "10m"),
		MemRequest: getEnv("MEM_REQUEST", "16Mi"),
	}
}

func unmarshalK8sResource[T any](data []byte) (*T, error) {
	var obj T
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

func getK8SKind(data []byte) (string, error) {
	type typeMeta struct {
		Kind string `yaml:"kind"`
	}
	var tm typeMeta
	if err := yaml.Unmarshal(data, &tm); err != nil {
		return "", fmt.Errorf("YAML unmarshal error: %v", err)
	}
	return tm.Kind, nil
}

func getFile(worktree *git.Worktree, path string) []byte {
	file, err := worktree.Filesystem.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	return data
}

// ResourceConfig holds parsed resource quantities for CPU and memory.
type ResourceConfig struct {
	CPURequest resource.Quantity
	MemRequest resource.Quantity
	CPULimit   resource.Quantity
	MemLimit   resource.Quantity
}

func updateResource(
	file []byte,
	resCfg ResourceConfig,
) (v []byte, err error) {
	kind, err := getK8SKind(file)
	if err != nil {
		fmt.Printf("Error getting Kubernetes kind: %v\n", err)
		return nil, err
	}

	var manifest any
	var containers []corev1.Container

	switch kind {
	case "Deployment":
		obj, err := unmarshalK8sResource[appsv1.Deployment](file)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			return nil, err
		}
		containers = obj.Spec.Template.Spec.Containers
		manifest = obj

	case "DaemonSet":
		obj, err := unmarshalK8sResource[appsv1.DaemonSet](file)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			return nil, err
		}
		containers = obj.Spec.Template.Spec.Containers
		manifest = obj

	case "StatefulSet":
		obj, err := unmarshalK8sResource[appsv1.StatefulSet](file)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			return nil, err
		}
		containers = obj.Spec.Template.Spec.Containers
		manifest = obj

	case "Pod":
		obj, err := unmarshalK8sResource[corev1.Pod](file)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			return nil, err
		}
		containers = obj.Spec.Containers
	case "Job":
		obj, err := unmarshalK8sResource[batchv1.Job](file)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			return nil, err
		}
		containers = obj.Spec.Template.Spec.Containers
		manifest = obj

	default:
		return nil, fmt.Errorf("Unsupported kind: %s", kind)
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("No containers found in %s", kind)
	}

	if len(containers) > 1 {
		fmt.Printf("Warning: Multiple containers found in %s, updating only the first one\n", kind)
	}

	containers[0].Resources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resCfg.CPURequest,
			corev1.ResourceMemory: resCfg.MemRequest,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resCfg.CPULimit,
			corev1.ResourceMemory: resCfg.MemLimit,
		},
	}

	v, err = yaml.Marshal(manifest)
	if err != nil {
		fmt.Printf("YAML marshal error: %v\n", err)
		return nil, err
	}

	return v, nil
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	cfg := getConfigFromEnv()
	for _, url := range cfg.RepoURLs {
		fmt.Println("======== Processing Repository:", url, "========")

		fs := memfs.New()
		repo, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
			URL:           fmt.Sprintf("%s/%s", cfg.BaseURL, url),
			SingleBranch:  true,
			ReferenceName: plumbing.ReferenceName(cfg.Branch),
		})
		if err != nil {
			panic(err)
		}

		worktree, err := repo.Worktree()
		if err != nil {
			panic(err)
		}

		// Read a file from the in-memory repository
		targetPath := filepath.Join("overlays", cfg.Env, "patches", "set_resources.yaml")
		file := getFile(worktree, targetPath)

		manifest, err := updateResource(
			file,
			ResourceConfig{
				CPURequest: resource.MustParse(cfg.CPURequest),
				MemRequest: resource.MustParse(cfg.MemRequest),
				CPULimit:   resource.MustParse(cfg.CPULimit),
				MemLimit:   resource.MustParse(cfg.MemLimit),
			},
		)

		if err != nil {
			fmt.Printf("Failed to update resource: %v\n", err)
			continue
		}

		// Write the updated YAML back to the in-memory filesystem
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

		// Add and commit the change
		_, err = worktree.Add(targetPath)
		if err != nil {
			fmt.Printf("Failed to add file to git: %v\n", err)
			continue
		}
		_, err = worktree.Commit("Update set_resources.yaml via automation", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "AutoUpdater",
				Email: "autoupdater@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			fmt.Printf("Failed to commit: %v\n", err)
			continue
		}

		// Push to remote
		err = repo.Push(&git.PushOptions{})
		if err != nil {
			fmt.Printf("Failed to push to remote: %v\n", err)
			continue
		}

		fmt.Printf("Updated file content and pushed to remote!!!")

	}

	fmt.Println("======== Finished Processing Repository ========")
}
