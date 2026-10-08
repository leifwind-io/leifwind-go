// SPDX-License-Identifier: MPL-2.0

package leifwindtest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/moby/moby/client"
	"github.com/testcontainers/testcontainers-go"
)

func TestNoImageMeansNoBackend(t *testing.T) {
	// No default image: the backend's is not public, so a test outside the
	// upstream CI has none to pull.
	if _, err := backendImage(mapEnv(nil)); !errors.Is(err, ErrNoBackend) {
		t.Fatalf("backendImage() error = %v, want ErrNoBackend", err)
	}
}

func TestBackendImageFromTheEnvironment(t *testing.T) {
	// CI tests the backend image its own pipeline built (go-client:test).
	const image = "registry.example.invalid/leifwind-stream-backend/ci:12345"
	got, err := backendImage(mapEnv(map[string]string{"LEIFWIND_BACKEND_IMAGE": image}))
	if err != nil || got != image {
		t.Fatalf("backendImage() = %q, %v; want %q", got, err, image)
	}
}

func TestAnUnreachableDaemonMeansNoBackend(t *testing.T) {
	// testcontainers resolves the Docker host once per process (a sync.Once).
	// A dead DOCKER_HOST set in this process would stay for every later test of
	// the package wherever DOCKER_HOST is the only way to the daemon, as in CI,
	// so the check runs in a child process of its own.
	if os.Getenv("LEIFWINDTEST_DEAD_DAEMON") == "1" {
		err := checkBackend(context.Background(), "example.invalid/backend:x")
		if !errors.Is(err, ErrNoBackend) {
			t.Fatalf("checkBackend() = %v, want ErrNoBackend", err)
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAnUnreachableDaemonMeansNoBackend$", "-test.count=1")
	cmd.Env = append(os.Environ(), "LEIFWINDTEST_DEAD_DAEMON=1", "DOCKER_HOST=tcp://127.0.0.1:1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the check in a child process failed: %v\n%s", err, out)
	}
}

func TestAnImageTheDaemonHasIsNotPulled(t *testing.T) {
	// A local build that no registry knows: the stack uses it as testcontainers
	// always did. Needs a Docker daemon; skipped without one.
	ctx := context.Background()
	if err := checkBackend(ctx, "postgres:18-alpine"); err != nil {
		t.Skipf("needs a Docker daemon that can pull postgres:18-alpine: %v", err)
	}
	const local = "leifwindtest.invalid/local-only:test"
	tagLocal(t, "postgres:18-alpine", local)
	if err := checkBackend(ctx, local); err != nil {
		t.Fatalf("checkBackend(%s) = %v, want nil for an image the daemon has", local, err)
	}
}

func TestNoBackendSkipsUnlessRequired(t *testing.T) {
	noBackend := fmt.Errorf("%w: no Docker daemon", ErrNoBackend)
	cases := []struct {
		name string
		err  error
		env  map[string]string
		skip bool
	}{
		{"no backend", noBackend, nil, true},
		{"no backend, but required", noBackend, map[string]string{"LEIFWIND_REQUIRE_BACKEND": "1"}, false},
		{"another error", errors.New("zitadel: timeout"), nil, false},
	}
	for _, c := range cases {
		if got := skips(c.err, mapEnv(c.env)); got != c.skip {
			t.Errorf("%s: skips() = %v, want %v", c.name, got, c.skip)
		}
	}
}

func TestRequireSkipsWithoutABackend(t *testing.T) {
	t.Setenv("LEIFWIND_REQUIRE_BACKEND", "")
	var inner *testing.T
	t.Run("no backend", func(t *testing.T) {
		inner = t
		Require(t, fmt.Errorf("%w: no Docker daemon", ErrNoBackend))
		t.Error("Require returned instead of skipping")
	})
	if !inner.Skipped() {
		t.Fatal("Require did not skip the test")
	}
}

// tagLocal gives the daemon's image SRC a second name that exists nowhere else.
func tagLocal(t *testing.T, src, name string) {
	t.Helper()
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = provider.Close() }()
	ctx := context.Background()
	if _, err := provider.Client().ImageTag(ctx, client.ImageTagOptions{Source: src, Target: name}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = provider.Client().ImageRemove(ctx, name, client.ImageRemoveOptions{})
	})
}
