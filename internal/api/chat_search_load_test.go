//go:build loadtest

package api

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/communication"
)

// searches measures the chat's search (SAL-30) on TestChatLoad's table, through the service (the role's scope, the
// service's own access check, the audit): within a conversation and across the practitioner's, who takes part in all
// 50 010 (the worst case: a real doctor has hundreds), for a word in about one message in 5000 and for one in every
// message. No target in the plan: the numbers go in docs/plans/chat.PROGRESS.md.
func searches(t *testing.T, owner *pgxpool.Pool, svc *communication.Service, long, small func() string) {
	t.Helper()
	ctx := context.Background()
	rare := func() string { return fmt.Sprintf("w%d", rand.IntN(loadWords)) }
	every := func() string { return "placeholder" }
	for _, q := range []string{"'w4321':*", "'placeholder':*"} {
		plan(t, owner, `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM communication
			WHERE to_tsvector('public.chat_search'::regconfig, body) @@ to_tsquery('public.chat_search'::regconfig, $1)
			ORDER BY sent_at DESC, seq DESC LIMIT 50`, q)
		plan(t, owner, `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM communication
			WHERE conversation_id = $2 AND seq > 0
			  AND to_tsvector('public.chat_search'::regconfig, body) @@ to_tsquery('public.chat_search'::regconfig, $1)
			ORDER BY sent_at DESC, seq DESC LIMIT 50`, q, long())
	}
	for _, tt := range []struct {
		conv  func() string // nil: across the practitioner's conversations
		query func() string
		name  string
		n     int
	}{
		{long, rare, "search a long conversation, a rare word", loadSamples / 4},
		{long, every, "search a long conversation, a word in every message", loadSamples / 4},
		{small, rare, "search a small conversation, a rare word", loadSamples / 4},
		{nil, rare, "search across the practitioner's 50 010 conversations, a rare word", loadSamples / 10},
		{nil, every, "search across the practitioner's 50 010 conversations, a word in every message", loadSamples / 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var d []time.Duration
			for range tt.n {
				began := loadWall.Now()
				var err error
				if tt.conv == nil {
					_, err = svc.SearchAll(ctx, loadPract, tt.query())
				} else {
					_, err = svc.Search(ctx, loadPract, tt.conv(), tt.query())
				}
				d = append(d, loadWall.Now().Sub(began))
				if err != nil {
					t.Fatal(err)
				}
			}
			report(t, tt.name, d, 0)
		})
	}
}

// plan logs the plan of query with args.
func plan(t *testing.T, owner *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	rows, err := owner.Query(t.Context(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		_ = rows.Scan(&line)
		lines = append(lines, line)
	}
	t.Logf("plan of the search with %v:\n%s", args, strings.Join(lines, "\n"))
}
