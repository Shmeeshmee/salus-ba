package worker

import (
	"log/slog"
	"testing"
)

// River's reindexer runs REINDEX INDEX CONCURRENTLY, which only a table's owner may run; the worker logs in
// as salus_app, which owns nothing. Index maintenance on River's tables is an owner-side ops task.
func TestConfig_ReindexesNoIndex(t *testing.T) {
	cfg := config(slog.New(slog.DiscardHandler))

	if cfg.ReindexerIndexNames == nil || len(cfg.ReindexerIndexNames) != 0 {
		t.Errorf("ReindexerIndexNames = %#v, want an empty, non-nil list (nil means River's defaults)", cfg.ReindexerIndexNames)
	}
}
