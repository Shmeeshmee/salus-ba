package auth_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

func TestPurgers_DeleteWhatEndedBeforeTheCutOffAndCountIt(t *testing.T) {
	before := time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC)
	repo := &fakeRepository{purged: 3}

	sessions, err := auth.NewSessionPurger(repo).Purge(t.Context(), before)
	if err != nil || sessions != 3 {
		t.Errorf("session Purge() = %d, %v; want 3, nil", sessions, err)
	}
	invites, err := auth.NewInvitePurger(repo).Purge(t.Context(), before)
	if err != nil || invites != 3 {
		t.Errorf("invite Purge() = %d, %v; want 3, nil", invites, err)
	}
	if want := []purge{{table: "session", before: before}, {table: "invite", before: before}}; !slices.Equal(repo.purges, want) {
		t.Errorf("deletes %v, want %v", repo.purges, want)
	}
}

func TestPurgers_ReturnTheRepositoryError(t *testing.T) {
	repo := &fakeRepository{err: errors.New("synthetic failure")}

	if _, err := auth.NewSessionPurger(repo).Purge(t.Context(), cutOff); err == nil {
		t.Error("session Purge() error = nil")
	}
	if _, err := auth.NewInvitePurger(repo).Purge(t.Context(), cutOff); err == nil {
		t.Error("invite Purge() error = nil")
	}
}
