package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// ConfigLoader is responsible for loading configuration from environment variables
type ConfigLoader interface {
	Load() Config
}

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

type EnvConfigLoader struct{}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func (e *EnvConfigLoader) Load() Config {
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
