// SPDX-License-Identifier: MPL-2.0

package leifwindtest

import "testing"

func TestBackendImageDefaultsToEdge(t *testing.T) {
	if got := backendImage(mapEnv(nil)); got != BackendImage {
		t.Fatalf("backendImage() = %q, want the default %q", got, BackendImage)
	}
}

func TestBackendImageFromTheEnvironment(t *testing.T) {
	// CI tests the backend image its own pipeline built (go-client:test).
	const image = "registry.example.invalid/leifwind-stream-backend/ci:12345"
	got := backendImage(mapEnv(map[string]string{"LEIFWIND_BACKEND_IMAGE": image}))
	if got != image {
		t.Fatalf("backendImage() = %q, want %q", got, image)
	}
}
