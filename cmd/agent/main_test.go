package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun_LogsStartAndStopAndReturnsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var logs bytes.Buffer

	if err := run(ctx, &logs); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
	for _, msg := range []string{`"msg":"started"`, `"msg":"stopped"`} {
		if !strings.Contains(logs.String(), msg) {
			t.Errorf("logs lack %s:\n%s", msg, logs.String())
		}
	}
}
