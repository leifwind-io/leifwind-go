// SPDX-License-Identifier: MPL-2.0

package leifwindtest

import (
	"os"
	"sync"
	"testing"
)

// The package shares one booted stack across tests (each boot costs ~60-90s).
// Boot is lazy: hermetic tests (contract/attach unit tests) must run without
// Docker, so the stack boots on the first sharedStack call, not in TestMain.
// Tests that exercise the stack already isolate via per-test orgs (see
// NewOrg), so sharing one stack here is safe — the same pattern the client
// and acctest packages use.
var (
	mainOnce    sync.Once
	mainStack   *Stack
	mainCleanup func()
	mainErr     error
)

// sharedStack returns the package-shared stack, booting it on first use; it
// skips the test without a backend and fails it if the boot failed (Require).
func sharedStack(t testing.TB) *Stack {
	t.Helper()
	mainOnce.Do(func() {
		mainStack, mainCleanup, mainErr = StartMain()
	})
	Require(t, mainErr)
	return mainStack
}

func TestMain(m *testing.M) {
	code := m.Run()
	if mainCleanup != nil {
		mainCleanup()
	}
	os.Exit(code)
}
