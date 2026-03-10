package resources

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

func TestParseQuantity(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		expected     resource.Quantity
	}{
		{
			name:         "Valid quantity",
			input:        "100Mi",
			defaultValue: "100Mi",
			expected:     resource.MustParse("100Mi"),
		},
		{
			name:         "Invalid quantity, falling back to default",
			input:        "invalid",
			defaultValue: "100Mi",
			expected:     resource.MustParse("100Mi"),
		},
		{
			name:         "Empty input, falling back to default",
			input:        "",
			defaultValue: "100Mi",
			expected:     resource.MustParse("100Mi"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseQuantity(tt.input, tt.defaultValue)
			if !result.Equal(tt.expected) {
				t.Errorf("ParseQuantity(%q, %q) = %v; want %v", tt.input, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestApplyRegistryOverride(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		registry string
		expected string
	}{
		{
			name:     "empty registry - no change",
			image:    "ghcr.io/openclaw/openclaw:1.0.0",
			registry: "",
			expected: "ghcr.io/openclaw/openclaw:1.0.0",
		},
		{
			name:     "registry with repo path",
			image:    "ghcr.io/openclaw/openclaw:latest",
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/openclaw/openclaw:latest",
		},
		{
			name:     "registry with port",
			image:    "ghcr.io/openclaw/openclaw:v1.2.3",
			registry: "my-registry:5000",
			expected: "my-registry:5000/openclaw/openclaw:v1.2.3",
		},
		{
			name:     "docker hub official image with tag",
			image:    "nginx:1.27-alpine",
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/nginx:1.27-alpine",
		},
		{
			name:     "docker hub image - two path components",
			image:    "ollama/ollama:latest",
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/ollama/ollama:latest",
		},
		{
			name:     "image with digest",
			image:    "ghcr.io/openclaw/openclaw@sha256:abc123",
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/openclaw/openclaw@sha256:abc123",
		},
		{
			name:     "docker hub image with digest",
			image:    "ollama/ollama@sha256:def456",
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/ollama/ollama@sha256:def456",
		},
		{
			name:     "registry with trailing slash",
			image:    "nginx:latest",
			registry: "my-registry.example.com/",
			expected: "my-registry.example.com/nginx:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyRegistryOverride(tt.image, tt.registry)
			if got != tt.expected {
				t.Errorf("ApplyRegistryOverride(%q, %q) = %q, want %q", tt.image, tt.registry, got, tt.expected)
			}
		})
	}
}

func TestGetImage_WithRegistry(t *testing.T) {
	tests := []struct {
		name     string
		image    openclawv1alpha1.ImageSpec
		registry string
		expected string
	}{
		{
			name:     "default image with registry",
			image:    openclawv1alpha1.ImageSpec{},
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/openclaw/openclaw:latest",
		},
		{
			name: "custom image with registry",
			image: openclawv1alpha1.ImageSpec{
				Repository: "ghcr.io/custom/repo",
				Tag:        "v1.0.0",
			},
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/custom/repo:v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := newTestInstance("test")
			instance.Spec.Image = tt.image
			instance.Spec.Registry = tt.registry
			got := GetImage(instance)
			if got != tt.expected {
				t.Errorf("GetImage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetTailscaleImage_WithRegistry(t *testing.T) {
	tests := []struct {
		name     string
		image    openclawv1alpha1.TailscaleImageSpec
		registry string
		expected string
	}{
		{
			name:     "default image with registry",
			image:    openclawv1alpha1.TailscaleImageSpec{},
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/tailscale/tailscale:latest",
		},
		{
			name: "custom image with registry",
			image: openclawv1alpha1.TailscaleImageSpec{
				Repository: "ghcr.io/custom/tailscale",
				Tag:        "v1.50",
			},
			registry: "my-registry.example.com",
			expected: "my-registry.example.com/custom/tailscale:v1.50",
		},
		{
			name: "registry with trailing slash",
			image: openclawv1alpha1.TailscaleImageSpec{
				Repository: "tailscale/tailscale",
				Tag:        "v1.50",
			},
			registry: "my-registry.example.com/",
			expected: "my-registry.example.com/tailscale/tailscale:v1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := newTestInstance("test")
			instance.Spec.Tailscale.Image = tt.image
			instance.Spec.Registry = tt.registry
			got := GetTailscaleImage(instance)
			if got != tt.expected {
				t.Errorf("GetTailscaleImage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildStatefulSet_WithRegistry(t *testing.T) {
	instance := newTestInstance("test")
	instance.Spec.Registry = "my-registry.example.com"

	sts := BuildStatefulSet(instance, "test-secret", nil)

	// Check main container image
	var mainContainer *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "openclaw" {
			mainContainer = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if mainContainer == nil {
		t.Fatal("main container not found")
	}
	if !strings.HasPrefix(mainContainer.Image, "my-registry.example.com/") {
		t.Errorf("main container image = %q, want registry prefix", mainContainer.Image)
	}

	// Check gateway proxy container
	var proxyContainer *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "gateway-proxy" {
			proxyContainer = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if proxyContainer == nil {
		t.Fatal("gateway-proxy container not found")
	}
	if !strings.HasPrefix(proxyContainer.Image, "my-registry.example.com/") {
		t.Errorf("gateway-proxy image = %q, want registry prefix", proxyContainer.Image)
	}

	// Check init-uv container
	var initUvContainer *corev1.Container
	for i := range sts.Spec.Template.Spec.InitContainers {
		if sts.Spec.Template.Spec.InitContainers[i].Name == "init-uv" {
			initUvContainer = &sts.Spec.Template.Spec.InitContainers[i]
			break
		}
	}
	if initUvContainer == nil {
		t.Fatal("init-uv container not found")
	}
	if !strings.HasPrefix(initUvContainer.Image, "my-registry.example.com/") {
		t.Errorf("init-uv image = %q, want registry prefix", initUvContainer.Image)
	}
}
