package communication

import (
	"context"
	"errors"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// GroupChats is what the group consult's handler calls (Service; docs/plans/chat.md, SAL-28).
type GroupChats interface {
	AddParticipant(ctx context.Context, c Caller, id string, in AddParticipant) (Participant, error)
	RemoveParticipant(ctx context.Context, c Caller, id, participantID string) error
	Invitations(ctx context.Context, c Caller) ([]Invitation, error)
	DecideInvitation(ctx context.Context, c Caller, id string, d Decision) error
	RevokeInvitation(ctx context.Context, c Caller, id string) error
	OpenThread(ctx context.Context, c Caller, id string) (Conversation, error)
	Threads(ctx context.Context, c Caller, id string) ([]Conversation, error)
}

// GroupHandler serves the group consult's routes (api/openapi.yaml, tag chat) beside HTTPHandler's, with its caller,
// log and error codes.
type GroupHandler struct {
	svc  GroupChats
	base *HTTPHandler
}

// NewGroupHandler returns the handler on svc, answering as base does.
func NewGroupHandler(svc GroupChats, base *HTTPHandler) *GroupHandler {
	return &GroupHandler{svc: svc, base: base}
}

// Routes declares the group consult's routes on mux.
func (h *GroupHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /conversations/{id}/participants", h.add)
	mux.HandleFunc("DELETE /conversations/{id}/participants/{participant_id}", h.remove)
	mux.HandleFunc("GET /me/chat-invitations", h.invitations)
	mux.HandleFunc("POST /me/chat-invitations/{id}/decision", h.decide)
	mux.HandleFunc("POST /me/chat-invitations/{id}/revoke", h.revoke)
	mux.HandleFunc("POST /conversations/{id}/threads", h.openThread)
	mux.HandleFunc("GET /conversations/{id}/threads", h.threads)
}

func (h *GroupHandler) add(w http.ResponseWriter, r *http.Request) {
	c, ok := h.base.signedIn(w, r)
	if !ok {
		return
	}
	var in AddParticipant
	if !decode(w, r, &in) {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	p, err := h.svc.AddParticipant(r.Context(), c, r.PathValue("id"), in)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *GroupHandler) remove(w http.ResponseWriter, r *http.Request) {
	c, ok := h.base.signedIn(w, r)
	if !ok {
		return
	}
	h.done(w, r, h.svc.RemoveParticipant(r.Context(), c, r.PathValue("id"), r.PathValue("participant_id")))
}

func (h *GroupHandler) invitations(w http.ResponseWriter, r *http.Request) {
	c, ok := h.base.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Invitations(r.Context(), c)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if list == nil {
		list = []Invitation{}
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, struct {
		Invitations []Invitation `json:"invitations"`
	}{list})
}

func (h *GroupHandler) decide(w http.ResponseWriter, r *http.Request) {
	c, ok := h.base.signedIn(w, r)
	if !ok {
		return
	}
	var in struct {
		Decision Decision `json:"decision"`
	}
	if !decode(w, r, &in) || in.Decision == "" {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	h.done(w, r, h.svc.DecideInvitation(r.Context(), c, r.PathValue("id"), in.Decision))
}

func (h *GroupHandler) revoke(w http.ResponseWriter, r *http.Request) {
	c, ok := h.base.signedIn(w, r)
	if !ok {
		return
	}
	h.done(w, r, h.svc.RevokeInvitation(r.Context(), c, r.PathValue("id")))
}

// openThread takes no body: a side thread has nothing of its own but its messages.
func (h *GroupHandler) openThread(w http.ResponseWriter, r *http.Request) {
	c, ok := h.base.signedIn(w, r)
	if !ok {
		return
	}
	thread, err := h.svc.OpenThread(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	noStore(w)
	httpx.JSON(w, http.StatusCreated, thread)
}

func (h *GroupHandler) threads(w http.ResponseWriter, r *http.Request) {
	c, ok := h.base.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Threads(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if list == nil {
		list = []Conversation{}
	}
	noStore(w)
	httpx.JSON(w, http.StatusOK, listJSON{Conversations: list})
}

// done answers 204 for a call without a body, or its error.
func (h *GroupHandler) done(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeError maps the group consult's own error, then answers as HTTPHandler does.
func (h *GroupHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrAlreadyParticipant) {
		writeCode(w, http.StatusConflict, "already_participant")
		return
	}
	h.base.writeError(w, r, err)
}
