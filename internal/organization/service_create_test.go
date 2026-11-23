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

func TestCreate_AddsAClinicAndAuditsItAsItsAgent(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}

	got, err := newService(repo, audit).Create(t.Context(), auditevent.AgentSignup, "  Synthetic clinic C ")

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
		AgentName:       auditevent.AgentSignup,
		EntityReference: "Organization/" + createdClinic,
		OrganizationID:  createdClinic,
	}
	if len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit events %+v,\nwant %+v", audit.events, want)
	}
}

func TestCreate_RejectsABlankOrOverlongName(t *testing.T) {
	for _, name := range []string{" ", strings.Repeat("x", 201), "Synthetic\rclinic", "http://example.test"} {
		repo := newFakeRepository()
		if _, err := newService(repo, &fakeAuditor{}).Create(t.Context(), auditevent.AgentOperator, name); !errors.Is(err, organization.ErrInvalidName) || len(repo.calls) != 0 {
			t.Errorf("Create(%d characters) = %v after %v, want ErrInvalidName", len(name), err, repo.calls)
		}
	}
}

func TestCreate_FailsOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Create(t.Context(), auditevent.AgentOperator, "Synthetic clinic C"); err == nil {
		t.Error("Create() with a failing repository: error = nil")
	}
	if _, err := newService(newFakeRepository(), &fakeAuditor{err: errors.New("audit down")}).Create(t.Context(), auditevent.AgentOperator, "Synthetic clinic C"); err == nil {
		t.Error("Create() with a failing audit: error = nil")
	}
}

func (r *fakeRepository) List(ctx context.Context, id string) (bool, error) {
	r.seen(ctx, "List")
	if r.err != nil {
		return false, r.err
	}
	o, ok := r.orgs[id]
	if !ok || o.Listed {
		return false, nil
	}
	o.Listed = true
	r.orgs[id] = o
	return true, nil
}

// The operator lists a self-signed-up clinic in the public directory (salusctl signup approve, docs/clinic-cms O1).
func TestList_ListsTheClinicOnceAndAuditsItAsTheOperator(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.orgs[createdClinic] = organization.Organization{ID: createdClinic, Name: "Synthetic clinic C", VersionID: 1}
	svc := newService(repo, audit)

	if err := svc.List(t.Context(), createdClinic); err != nil || !repo.orgs[createdClinic].Listed {
		t.Fatalf("List() = %v, listed %v; want the clinic listed", err, repo.orgs[createdClinic].Listed)
	}
	if len(audit.events) != 1 || audit.events[0].Action != auditevent.ActionUpdate || audit.events[0].AgentName != auditevent.AgentOperator ||
		audit.events[0].EntityReference != "Organization/"+createdClinic {
		t.Errorf("audit events %+v, want one update by the operator", audit.events)
	}
	if err := svc.List(t.Context(), createdClinic); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("List() of a listed clinic = %v, want ErrNotFound", err)
	}
	if err := svc.List(t.Context(), "c0000000-0000-4000-8000-000000000000"); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("List() of an unknown clinic = %v, want ErrNotFound", err)
	}
}
