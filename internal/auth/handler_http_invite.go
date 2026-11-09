package auth

import (
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// inviteJSON is the Invite schema: no token, no inviter, no clinic (the admin's own).
type inviteJSON struct {
	ExpiresAt  time.Time `json:"expiresAt"`
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Role       Role      `json:"role"`
	GivenName  string    `json:"givenName,omitempty"`
	FamilyName string    `json:"familyName,omitempty"`
}

func toInviteJSON(inv Invite) inviteJSON {
	return inviteJSON{ID: inv.ID, Email: inv.Email, Role: inv.Role, GivenName: inv.GivenName, FamilyName: inv.FamilyName, ExpiresAt: inv.ExpiresAt}
}

func (h *Handler) invite(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req struct {
		Email      string `json:"email"`
		Role       Role   `json:"role"`
		GivenName  string `json:"givenName"`
		FamilyName string `json:"familyName"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	inv, err := h.svc.Invite(r.Context(), sess, Invite{Email: req.Email, Role: req.Role, GivenName: req.GivenName, FamilyName: req.FamilyName})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toInviteJSON(inv))
}

func (h *Handler) listInvites(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	invites, err := h.svc.ListInvites(r.Context(), sess)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	body := struct {
		Invites []inviteJSON `json:"invites"`
	}{Invites: make([]inviteJSON, 0, len(invites))}
	for _, inv := range invites {
		body.Invites = append(body.Invites, toInviteJSON(inv))
	}
	httpx.JSON(w, http.StatusOK, body)
}

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		languagesJSON
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	language, detected, ok := decodeSignIn(w, r, &req, &req.languagesJSON)
	if !ok {
		return
	}
	acc, err := h.svc.AcceptInvite(r.Context(), req.Token, req.Password, language, detected)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusCreated, map[string]string{"challenge": acc.Challenge, "secret": acc.Enrolment.Secret, "uri": acc.Enrolment.URI})
}
