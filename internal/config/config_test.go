package config_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"k8s-resource-adjustment/internal/config"
)

func TestEnvConfigLoader_Load(t *testing.T) {
	// Save and restore environment variables to avoid side effects
	origEnv := os.Environ()
	restoreEnv := func() {
		os.Clearenv()
		for _, kv := range origEnv {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}
	defer restoreEnv()

	tests := []struct {
		name     string
		env      map[string]string
		expected config.Config
	}{
		{
			name: "all env vars set",
			env: map[string]string{
				"ENV":         "prod",
				"BASE_URL":    "https://github.com/example/repo.git",
				"BRANCH":      "main",
				"REPO_URLS":   "https://repo1.git, https://repo2.git",
				"CPU_LIMIT":   "100m",
				"MEM_LIMIT":   "256Mi",
				"CPU_REQUEST": "50m",
				"MEM_REQUEST": "128Mi",
			},
			expected: config.Config{
				Env:        "prod",
				BaseURL:    "https://github.com/example/repo.git",
				Branch:     "main",
				RepoURLs:   []string{"https://repo1.git", "https://repo2.git"},
				CPULimit:   "100m",
				MemLimit:   "256Mi",
				CPURequest: "50m",
				MemRequest: "128Mi",
			},
		},
		{
			name: "missing env vars uses defaults",
			env:  map[string]string{},
			expected: config.Config{
				Env:        "__ENV__",
				BaseURL:    "__GIT_URL__",
				Branch:     "__BRANCH__",
				RepoURLs:   []string{"__URL_1__", "__URL_2__"},
				CPULimit:   "20m",
				MemLimit:   "32Mi",
				CPURequest: "10m",
				MemRequest: "16Mi",
			},
		},
		{
			name: "REPO_URLS with spaces and empty entries",
			env: map[string]string{
				"REPO_URLS": " url1.git , , url2.git ",
			},
			expected: config.Config{
				Env:        "__ENV__",
				BaseURL:    "__GIT_URL__",
				Branch:     "__BRANCH__",
				RepoURLs:   []string{"url1.git", "", "url2.git"},
				CPULimit:   "20m",
				MemLimit:   "32Mi",
				CPURequest: "10m",
				MemRequest: "16Mi",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreEnv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			loader := &config.EnvConfigLoader{}
			got := loader.Load()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Load() = %+v; want %+v", got, tt.expected)
			}
		})
	}
}
