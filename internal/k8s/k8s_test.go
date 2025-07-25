package k8s_test

import (
	"testing"

	"k8s-resource-adjustment/internal/k8s"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

func TestDefaultResourcePatcher_Patch(t *testing.T) {
	resCfg := k8s.ResourceConfig{
		CPURequest: resource.MustParse("100m"),
		MemRequest: resource.MustParse("128Mi"),
		CPULimit:   resource.MustParse("200m"),
		MemLimit:   resource.MustParse("256Mi"),
	}

	tests := []struct {
		name        string
		inputFile   []byte
		wantErr     bool
		errContains string
		verify      func(t *testing.T, patchedYAML []byte)
	}{
		{
			name: "valid deployment",
			inputFile: []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  template:
    spec:
      containers:
      - name: test-container
        image: nginx`),
			wantErr: false,
			verify: func(t *testing.T, patchedYAML []byte) {
				var deployment appsv1.Deployment
				err := yaml.Unmarshal(patchedYAML, &deployment)
				require.NoError(t, err)
				assert.Equal(t, resCfg.CPURequest, deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resCfg.MemRequest, deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory])
				assert.Equal(t, resCfg.CPULimit, deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resCfg.MemLimit, deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory])
			},
		},
		{
			name: "valid daemonset",
			inputFile: []byte(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: test-daemonset
spec:
  template:
    spec:
      containers:
      - name: test-container
        image: nginx`),
			wantErr: false,
			verify: func(t *testing.T, patchedYAML []byte) {
				var daemonset appsv1.DaemonSet
				err := yaml.Unmarshal(patchedYAML, &daemonset)
				require.NoError(t, err)
				assert.Equal(t, resCfg.CPURequest, daemonset.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU])
				assert.Equal(t, resCfg.MemRequest, daemonset.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory])
				assert.Equal(t, resCfg.CPULimit, daemonset.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resCfg.MemLimit, daemonset.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory])
			},
		},
		{
			name: "unsupported kind",
			inputFile: []byte(`
apiVersion: v1
kind: Service
metadata:
  name: test-service`),
			wantErr:     true,
			errContains: "unsupported kind: Service",
		},
		{
			name:        "invalid yaml",
			inputFile:   []byte(`invalid: [not yaml`),
			wantErr:     true,
			errContains: "YAML unmarshal error",
		},
		{
			name: "no containers",
			inputFile: []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  template:
    spec:
      containers: []`),
			wantErr:     true,
			errContains: "no containers found",
		},
	}

	patcher := &k8s.DefaultResourcePatcher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFile, err := patcher.Patch(tt.inputFile, resCfg)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, tt.verify, "verify function must be provided for successful tests")
				tt.verify(t, gotFile)
			}
		})
	}
}
