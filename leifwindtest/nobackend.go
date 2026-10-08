// SPDX-License-Identifier: MPL-2.0

package leifwindtest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

// ErrNoBackend means the stack cannot start here: LEIFWIND_BACKEND_IMAGE is
// unset, there is no Docker daemon, or the image cannot be pulled. The backend
// image is not public, so outside the upstream CI that is the normal case, and
// Start and Require skip the test instead of failing it.
var ErrNoBackend = errors.New("backend image not accessible; integration tests run in the private upstream CI")

// pullTimeout bounds the pull check, so a registry that does not answer skips
// the tests instead of hanging them.
const pullTimeout = 3 * time.Minute

// backendImage is the image the stack starts: LEIFWIND_BACKEND_IMAGE, or
// ErrNoBackend when it is unset.
func backendImage(getenv func(string) string) (string, error) {
	if image := getenv("LEIFWIND_BACKEND_IMAGE"); image != "" {
		return image, nil
	}
	return "", fmt.Errorf("%w: LEIFWIND_BACKEND_IMAGE is not set", ErrNoBackend)
}

// checkBackend makes sure the Docker daemon has the image: one it already has
// (a local build) is used as is, any other is pulled within pullTimeout. Any
// failure, a missing daemon included, is ErrNoBackend.
func checkBackend(ctx context.Context, image string) (err error) {
	defer func() {
		// testcontainers panics when it finds no Docker host at all.
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: no Docker daemon: %v", ErrNoBackend, r)
		}
	}()
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		return fmt.Errorf("%w: no Docker daemon: %w", ErrNoBackend, err)
	}
	defer func() { _ = provider.Close() }()
	ctx, cancel := context.WithTimeout(ctx, pullTimeout)
	defer cancel()
	if err := provider.Health(ctx); err != nil {
		return fmt.Errorf("%w: no Docker daemon: %w", ErrNoBackend, err)
	}
	if _, err := provider.Client().ImageInspect(ctx, image); err == nil {
		return nil
	}
	if err := provider.PullImage(ctx, image); err != nil {
		return fmt.Errorf("%w: pulling %s: %w", ErrNoBackend, image, err)
	}
	return nil
}

// skips reports whether a stack error skips the test: ErrNoBackend does,
// unless LEIFWIND_REQUIRE_BACKEND is 1 (the upstream CI, whose integration
// tests must run).
func skips(err error, getenv func(string) string) bool {
	return errors.Is(err, ErrNoBackend) && getenv("LEIFWIND_REQUIRE_BACKEND") != "1"
}

// Require ends the test when the stack could not start: it skips the test for
// ErrNoBackend (see skips) and fails it for anything else. A nil error does
// nothing. For tests that boot the stack in TestMain with StartMain.
func Require(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if skips(err, os.Getenv) {
		t.Skip(err.Error())
	}
	t.Fatalf("leifwindtest: %v", err)
}
