package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/location"
)

// roomRoutes are a location's rooms and devices (package RM1): the clinic's staff read them, the desk included; only
// its admin writes.
func roomRoutes(svc *location.Service, accounts *auth.Service, log *slog.Logger) func(*http.ServeMux) {
	clinicCaller := organizationCaller(accounts)
	caller := func(ctx context.Context) (location.Staff, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return location.Staff{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	return location.NewRoomHandler(svc, caller, log).Routes
}
