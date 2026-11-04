package organization_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

const createdClinic = "c0000000-0000-4000-8000-00000000c0de"

func (r *fakeRepository) Create(ctx context.Context, name string) (organization.Organization, error) {
	r.seen(ctx, "Create")
	if r.err != nil {
		return organization.Organization{}, r.err
	}
	o := organization.Organization{ID: createdClinic, Name: name, VersionID: 1, Active: true, LastUpdated: now}
	r.orgs[o.ID] = o
	return o, nil
}

func TestCreate_AddsAClinicAndAuditsItAsTheOperatorCommand(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}

	got, err := newService(repo, audit).Create(t.Context(), "  Synthetic clinic C ")

	if want := repo.orgs[createdClinic]; err != nil || got != want || got.Name != "Synthetic clinic C" {
		t.Fatalf("Create() = %+v, %v; want the new clinic, its name trimmed", got, err)
	}
	if !slices.Equal(repo.calls, []string{"Create"}) {
		t.Errorf("calls %v, want the insert inside a transaction", repo.calls)
	}
	want := auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          auditevent.ActionCreate,
		AgentName:       auditevent.AgentOperator,
		EntityReference: "Organization/" + createdClinic,
		OrganizationID:  createdClinic,
	}
	if len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit events %+v,\nwant %+v", audit.events, want)
	}
}

func TestCreate_RejectsABlankOrOverlongName(t *testing.T) {
	for _, name := range []string{" ", strings.Repeat("x", 201)} {
		repo := newFakeRepository()
		if _, err := newService(repo, &fakeAuditor{}).Create(t.Context(), name); !errors.Is(err, organization.ErrInvalidName) || len(repo.calls) != 0 {
			t.Errorf("Create(%d characters) = %v after %v, want ErrInvalidName", len(name), err, repo.calls)
		}
	}
}

func TestCreate_FailsOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Create(t.Context(), "Synthetic clinic C"); err == nil {
		t.Error("Create() with a failing repository: error = nil")
	}
	if _, err := newService(newFakeRepository(), &fakeAuditor{err: errors.New("audit down")}).Create(t.Context(), "Synthetic clinic C"); err == nil {
		t.Error("Create() with a failing audit: error = nil")
	}
}
