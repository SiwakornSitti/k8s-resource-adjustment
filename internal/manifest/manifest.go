package manifest

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// K8sManifest represents a generic Kubernetes manifest
type K8sManifest map[string]interface{}

// ResourceConfig represents resource limits and requests configuration
type ResourceConfig struct {
	CPU           *string `yaml:"cpu,omitempty"`
	Memory        *string `yaml:"memory,omitempty"`
	CPURequest    *string `yaml:"cpu_request,omitempty"`
	MemoryRequest *string `yaml:"memory_request,omitempty"`
	RequestsCPU   *string `yaml:"requests_cpu,omitempty"`
	RequestsMemory *string `yaml:"requests_memory,omitempty"`
	LimitsCPU     *string `yaml:"limits_cpu,omitempty"`
	LimitsMemory  *string `yaml:"limits_memory,omitempty"`
}

// UpdateDeploymentReplicas updates the replica count in a deployment manifest
func UpdateDeploymentReplicas(manifestPath string, replicas int32) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Handle multi-document YAML files
	documents := strings.Split(string(content), "---")
	var updatedDocuments []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest K8sManifest
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			// If it's not valid YAML, keep it as is
			updatedDocuments = append(updatedDocuments, doc)
			continue
		}

		// Check if this is a Deployment
		if kind, ok := manifest["kind"].(string); ok && kind == "Deployment" {
			// Update replica count
			if spec, ok := manifest["spec"].(map[interface{}]interface{}); ok {
				spec["replicas"] = replicas
				fmt.Printf("Updated replicas to %d in deployment manifest\n", replicas)
			}
		}

		// Marshal back to YAML
		updatedDoc, err := yaml.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("failed to marshal updated manifest: %w", err)
		}
		updatedDocuments = append(updatedDocuments, string(updatedDoc))
	}

	// Write back to file
	updatedContent := strings.Join(updatedDocuments, "---\n")
	if err := os.WriteFile(manifestPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated manifest: %w", err)
	}

	return nil
}

// UpdateResourceLimits updates resource limits in a deployment manifest
func UpdateResourceLimits(manifestPath string, limits map[string]interface{}) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Handle multi-document YAML files
	documents := strings.Split(string(content), "---")
	var updatedDocuments []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest K8sManifest
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			// If it's not valid YAML, keep it as is
			updatedDocuments = append(updatedDocuments, doc)
			continue
		}

		// Check if this is a Deployment
		if kind, ok := manifest["kind"].(string); ok && kind == "Deployment" {
			if err := updateDeploymentResourceLimits(manifest, limits); err != nil {
				return fmt.Errorf("failed to update resource limits: %w", err)
			}
		}

		// Marshal back to YAML
		updatedDoc, err := yaml.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("failed to marshal updated manifest: %w", err)
		}
		updatedDocuments = append(updatedDocuments, string(updatedDoc))
	}

	// Write back to file
	updatedContent := strings.Join(updatedDocuments, "---\n")
	if err := os.WriteFile(manifestPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated manifest: %w", err)
	}

	return nil
}

func updateDeploymentResourceLimits(manifest K8sManifest, limits map[string]interface{}) error {
	// Convert map to ResourceConfig struct with nil-safe conversion
	resourceConfig := NewResourceConfig(limits)
	return updateDeploymentResourcesWithStruct(manifest, resourceConfig)
}

func updateDeploymentResourcesWithStruct(manifest K8sManifest, config ResourceConfig) error {
	// Handle Kustomize patch files which may have different structure
	specValue := manifest["spec"]
	
	// Cast to the correct type since it's also K8sManifest
	spec, ok := specValue.(K8sManifest)
	if !ok {
		// Try map[string]interface{} directly
		if specMap, ok := specValue.(map[string]interface{}); ok {
			spec = specMap
		} else {
			return fmt.Errorf("deployment spec not found")
		}
	}

	// For Kustomize patches, we might have template directly under spec
	var containers []interface{}
	var found bool

	// First try: spec.template.spec.containers (full deployment)
	if template, ok := spec["template"].(K8sManifest); ok {
		if templateSpec, ok := template["spec"].(K8sManifest); ok {
			if containersList, ok := templateSpec["containers"].([]interface{}); ok {
				containers = containersList
				found = true
			}
		}
	}

	// Second try: spec.containers (simplified patch)
	if !found {
		if containersList, ok := spec["containers"].([]interface{}); ok {
			containers = containersList
			found = true
		}
	}

	if !found {
		return fmt.Errorf("containers not found in deployment")
	}

	// Update resources for all containers
	for i, container := range containers {
		containerMap, ok := container.(K8sManifest)
		if !ok {
			// Try map[string]interface{} as fallback
			if containerMapInterface, ok := container.(map[string]interface{}); ok {
				containerMap = containerMapInterface
			} else {
				continue
			}
		}

		// Initialize resources if not exists
		if _, exists := containerMap["resources"]; !exists {
			containerMap["resources"] = make(K8sManifest)
		}

		resources := containerMap["resources"].(K8sManifest)

		// Initialize limits and requests if not exists
		if _, exists := resources["limits"]; !exists {
			resources["limits"] = make(K8sManifest)
		}
		if _, exists := resources["requests"]; !exists {
			resources["requests"] = make(K8sManifest)
		}

		resourceLimits := resources["limits"].(K8sManifest)
		resourceRequests := resources["requests"].(K8sManifest)

		// Update limits - only if not nil
		if config.CPU != nil {
			resourceLimits["cpu"] = *config.CPU
		}
		if config.Memory != nil {
			resourceLimits["memory"] = *config.Memory
		}
		if config.LimitsCPU != nil {
			resourceLimits["cpu"] = *config.LimitsCPU
		}
		if config.LimitsMemory != nil {
			resourceLimits["memory"] = *config.LimitsMemory
		}

		// Update requests - only if not nil
		if config.CPURequest != nil {
			resourceRequests["cpu"] = *config.CPURequest
		}
		if config.MemoryRequest != nil {
			resourceRequests["memory"] = *config.MemoryRequest
		}
		if config.RequestsCPU != nil {
			resourceRequests["cpu"] = *config.RequestsCPU
		}
		if config.RequestsMemory != nil {
			resourceRequests["memory"] = *config.RequestsMemory
		}

		containers[i] = containerMap
		fmt.Printf("Updated resource limits and requests for container in deployment\n")
	}

	return nil
}

// UpdateImageTag updates the image tag in a deployment manifest
func UpdateImageTag(manifestPath string, containerName string, newImage string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Handle multi-document YAML files
	documents := strings.Split(string(content), "---")
	var updatedDocuments []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest K8sManifest
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			// If it's not valid YAML, keep it as is
			updatedDocuments = append(updatedDocuments, doc)
			continue
		}

		// Check if this is a Deployment
		if kind, ok := manifest["kind"].(string); ok && kind == "Deployment" {
			if err := updateDeploymentImage(manifest, containerName, newImage); err != nil {
				return fmt.Errorf("failed to update image: %w", err)
			}
		}

		// Marshal back to YAML
		updatedDoc, err := yaml.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("failed to marshal updated manifest: %w", err)
		}
		updatedDocuments = append(updatedDocuments, string(updatedDoc))
	}

	// Write back to file
	updatedContent := strings.Join(updatedDocuments, "---\n")
	if err := os.WriteFile(manifestPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated manifest: %w", err)
	}

	return nil
}

func updateDeploymentImage(manifest K8sManifest, containerName string, newImage string) error {
	spec, ok := manifest["spec"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("deployment spec not found")
	}

	template, ok := spec["template"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("deployment template not found")
	}

	templateSpec, ok := template["spec"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("deployment template spec not found")
	}

	containers, ok := templateSpec["containers"].([]interface{})
	if !ok {
		return fmt.Errorf("containers not found in deployment")
	}

	// Update image for specified container or all containers if containerName is empty
	for i, container := range containers {
		containerMap, ok := container.(map[interface{}]interface{})
		if !ok {
			continue
		}

		// If containerName is specified, only update that container
		if containerName != "" {
			if name, exists := containerMap["name"].(string); !exists || name != containerName {
				continue
			}
		}

		containerMap["image"] = newImage
		containers[i] = containerMap
		fmt.Printf("Updated image to %s for container %s\n", newImage, containerName)
	}

	return nil
}

// ValidateManifest validates that a manifest file is valid YAML
func ValidateManifest(manifestPath string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Handle multi-document YAML files
	documents := strings.Split(string(content), "---")

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest K8sManifest
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			return fmt.Errorf("invalid YAML in manifest: %w", err)
		}

		// Basic validation - check required fields
		if kind, ok := manifest["kind"].(string); !ok || kind == "" {
			return fmt.Errorf("manifest missing 'kind' field")
		}

		if apiVersion, ok := manifest["apiVersion"].(string); !ok || apiVersion == "" {
			return fmt.Errorf("manifest missing 'apiVersion' field")
		}

		if metadata, ok := manifest["metadata"].(map[interface{}]interface{}); !ok {
			return fmt.Errorf("manifest missing 'metadata' field")
		} else {
			if name, ok := metadata["name"].(string); !ok || name == "" {
				return fmt.Errorf("manifest metadata missing 'name' field")
			}
		}
	}

	return nil
}

// UpdateResourceRequests updates only resource requests in a deployment manifest
func UpdateResourceRequests(manifestPath string, requests map[string]interface{}) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Handle multi-document YAML files
	documents := strings.Split(string(content), "---")
	var updatedDocuments []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest K8sManifest
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			// If it's not valid YAML, keep it as is
			updatedDocuments = append(updatedDocuments, doc)
			continue
		}

		// Check if this is a Deployment
		if kind, ok := manifest["kind"].(string); ok && kind == "Deployment" {
			if err := updateDeploymentResourceRequests(manifest, requests); err != nil {
				return fmt.Errorf("failed to update resource requests: %w", err)
			}
		}

		// Marshal back to YAML
		updatedDoc, err := yaml.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("failed to marshal updated manifest: %w", err)
		}
		updatedDocuments = append(updatedDocuments, string(updatedDoc))
	}

	// Write back to file
	updatedContent := strings.Join(updatedDocuments, "---\n")
	if err := os.WriteFile(manifestPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated manifest: %w", err)
	}

	return nil
}

func updateDeploymentResourceRequests(manifest K8sManifest, requests map[string]interface{}) error {
	spec, ok := manifest["spec"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("deployment spec not found")
	}

	template, ok := spec["template"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("deployment template not found")
	}

	templateSpec, ok := template["spec"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("deployment template spec not found")
	}

	containers, ok := templateSpec["containers"].([]interface{})
	if !ok {
		return fmt.Errorf("containers not found in deployment")
	}

	// Update resource requests for all containers
	for i, container := range containers {
		containerMap, ok := container.(map[interface{}]interface{})
		if !ok {
			continue
		}

		// Initialize resources if not exists
		if _, exists := containerMap["resources"]; !exists {
			containerMap["resources"] = make(map[interface{}]interface{})
		}

		resources := containerMap["resources"].(map[interface{}]interface{})

		// Initialize requests if not exists
		if _, exists := resources["requests"]; !exists {
			resources["requests"] = make(map[interface{}]interface{})
		}

		resourceRequests := resources["requests"].(map[interface{}]interface{})

		// Update requests
		if cpu, exists := requests["cpu"]; exists {
			resourceRequests["cpu"] = cpu
		}
		if memory, exists := requests["memory"]; exists {
			resourceRequests["memory"] = memory
		}

		containers[i] = containerMap
		fmt.Printf("Updated resource requests for container in deployment\n")
	}

	return nil
}

// NewResourceConfig creates a ResourceConfig from a map with nil-safe conversion
func NewResourceConfig(limits map[string]interface{}) ResourceConfig {
	// Helper function to get string pointer from interface
	getStringPtr := func(value interface{}) *string {
		if value == nil {
			return nil
		}
		if str, ok := value.(string); ok {
			return &str
		}
		return nil
	}
	
	return ResourceConfig{
		CPU:           getStringPtr(limits["cpu"]),
		Memory:        getStringPtr(limits["memory"]),
		CPURequest:    getStringPtr(limits["cpu_request"]),
		MemoryRequest: getStringPtr(limits["memory_request"]),
		RequestsCPU:   getStringPtr(limits["requests_cpu"]),
		RequestsMemory: getStringPtr(limits["requests_memory"]),
		LimitsCPU:     getStringPtr(limits["limits_cpu"]),
		LimitsMemory:  getStringPtr(limits["limits_memory"]),
	}
}

// UpdateResourceLimitsWithStruct updates resource limits using ResourceConfig struct
func UpdateResourceLimitsWithStruct(manifestPath string, config ResourceConfig) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Handle multi-document YAML files
	documents := strings.Split(string(content), "---")
	var updatedDocuments []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest K8sManifest
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			// If it's not valid YAML, keep it as is
			updatedDocuments = append(updatedDocuments, doc)
			continue
		}

		// Check if this is a Deployment
		if kind, ok := manifest["kind"].(string); ok && kind == "Deployment" {
			if err := updateDeploymentResourcesWithStruct(manifest, config); err != nil {
				return fmt.Errorf("failed to update resource limits: %w", err)
			}
		}

		// Marshal back to YAML
		updatedDoc, err := yaml.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("failed to marshal updated manifest: %w", err)
		}
		updatedDocuments = append(updatedDocuments, string(updatedDoc))
	}

	// Write back to file
	updatedContent := strings.Join(updatedDocuments, "---\n")
	if err := os.WriteFile(manifestPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated manifest: %w", err)
	}

	return nil
}

// Helper function to get keys from a map for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
