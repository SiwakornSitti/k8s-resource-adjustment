package k8s

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

// K8sResourcePatcher defines the interface for patching resource requirements in K8s manifests
type K8sResourcePatcher interface {
	Patch([]byte, ResourceConfig) ([]byte, error)
}

// DefaultK8sResourcePatcher implements K8sResourcePatcher for common K8s kinds
type DefaultK8sResourcePatcher struct{}

// ResourceConfig holds parsed resource quantities for CPU and memory.
type ResourceConfig struct {
	CPURequest resource.Quantity
	MemRequest resource.Quantity
	CPULimit   resource.Quantity
	MemLimit   resource.Quantity
}

func unmarshalK8sResource[T any](data []byte) (*T, error) {
	var obj T
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

type typeMeta struct {
	Kind string `yaml:"kind"`
}

func getKind(data []byte) (string, error) {
	var tm typeMeta
	if err := yaml.Unmarshal(data, &tm); err != nil {
		return "", fmt.Errorf("YAML unmarshal error: %v", err)
	}
	return tm.Kind, nil
}

// containerExtractor is a function that extracts the manifest object and containers from a resource file.
type containerExtractor func(file []byte) (any, []corev1.Container, error)

// newExtractor creates a containerExtractor for a specific Kubernetes resource type using generics.
func newExtractor[T any](getContainers func(obj *T) []corev1.Container) containerExtractor {
	return func(file []byte) (any, []corev1.Container, error) {
		obj, err := unmarshalK8sResource[T](file)
		if err != nil {
			return nil, nil, err
		}
		return obj, getContainers(obj), nil
	}
}

var extractorMap = map[string]containerExtractor{
	"Deployment": newExtractor(func(o *appsv1.Deployment) []corev1.Container {
		return o.Spec.Template.Spec.Containers
	}),
	"DaemonSet": newExtractor(func(o *appsv1.DaemonSet) []corev1.Container {
		return o.Spec.Template.Spec.Containers
	}),
	"StatefulSet": newExtractor(func(o *appsv1.StatefulSet) []corev1.Container {
		return o.Spec.Template.Spec.Containers
	}),
	"Pod": newExtractor(func(o *corev1.Pod) []corev1.Container {
		return o.Spec.Containers
	}),
	"Job": newExtractor(func(o *batchv1.Job) []corev1.Container {
		return o.Spec.Template.Spec.Containers
	}),
}

func (p *DefaultK8sResourcePatcher) Patch(file []byte, resCfg ResourceConfig) ([]byte, error) {
	kind, err := getKind(file)
	if err != nil {
		return nil, err
	}

	extractor, ok := extractorMap[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}

	manifest, containers, err := extractor(file)
	if err != nil {
		return nil, fmt.Errorf("failed to extract containers for kind %s: %w", kind, err)
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("no containers found in %s", kind)
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

	return yaml.Marshal(manifest)
}
