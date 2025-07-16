package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Git struct {
		BaseURL string `yaml:"base_url"`
		Branch  string `yaml:"branch"`
	} `yaml:"git"`
	Repositories []string        `yaml:"repositories"`
	Environment  string          `yaml:"env"`
	Resources    ResourcesConfig `yaml:"resources"`
}

type ResourcesConfig struct {
	Limits   ResourceLimits `yaml:"limits"`
	Requests ResourceLimits `yaml:"requests"`
}

type ResourceLimits struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type DeploymentConfig struct {
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace"`
	Replicas   int32  `yaml:"replicas"`
	Repository string `yaml:"repository"`
}

type ResourceConfig struct {
	CPU        *string `yaml:"cpu,omitempty"`
	Memory     *string `yaml:"memory,omitempty"`
	CPURequest *string `yaml:"cpu_request,omitempty"`
	MemoryRequest *string `yaml:"memory_request,omitempty"`
	RequestsCPU   *string `yaml:"requests_cpu,omitempty"`
	RequestsMemory *string `yaml:"requests_memory,omitempty"`
	LimitsCPU     *string `yaml:"limits_cpu,omitempty"`
	LimitsMemory  *string `yaml:"limits_memory,omitempty"`
}

type ImageConfig struct {
	Name          string `yaml:"name"`
	Namespace     string `yaml:"namespace"`
	ContainerName string `yaml:"container_name"`
	Image         string `yaml:"image"`
	Repository    string `yaml:"repository"`
}

func Load() (*Config, error) {
	configPath := getConfigPath()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "config.yaml"
}

func (c *Config) Validate() error {
	// Validate git configuration
	if c.Git.BaseURL == "" {
		return fmt.Errorf("git base_url cannot be empty")
	}
	if c.Git.Branch == "" {
		return fmt.Errorf("git branch cannot be empty")
	}
	
	// Validate repositories
	if len(c.Repositories) == 0 {
		return fmt.Errorf("at least one repository must be configured")
	}
	
	for i, repo := range c.Repositories {
		if repo == "" {
			return fmt.Errorf("repository[%d] path cannot be empty", i)
		}
	}
	
	// Validate resources
	if c.Resources.Limits.CPU == "" && c.Resources.Limits.Memory == "" &&
		c.Resources.Requests.CPU == "" && c.Resources.Requests.Memory == "" {
		return fmt.Errorf("at least one resource configuration must be set")
	}
	
	return nil
}
