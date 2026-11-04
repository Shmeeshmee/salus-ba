//go:build e2e

// Package e2e tests the running stack end to end. Run with `make e2e`, which starts the stack, builds
// salusctl and points E2E_BASE_URL, E2E_MAILPIT_URL and E2E_SALUSCTL at the API, Mailpit and the operator
// command on the compose network.
package e2e_test

import "testing"

func TestStack_ReadyzReturns200(t *testing.T) {
	waitReady(t)
}
