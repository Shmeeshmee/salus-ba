package terminology

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// GroupReviewService is what the GroupReviewHandler calls: GroupReviews.
type GroupReviewService interface {
	List(ctx context.Context, c Caller, language string) ([]GroupReviewEntry, error)
	EditGroupName(ctx context.Context, c Caller, group, language, text string, version int32) (GroupName, error)
	ReviewGroupName(ctx context.Context, c Caller, group, language string, d Decision, version int32) (GroupName, error)
}

// GroupReviewHandler serves the review panel's diagnosis group names (docs/plans/analytics.md §3), with the review
// panel's request reading and error codes.
type GroupReviewHandler struct {
	*ReviewHandler
	groups GroupReviewService
}

// NewGroupReviewHandler returns the handler on svc; caller is who is signed in (internal/api), false for nobody.
func NewGroupReviewHandler(svc GroupReviewService, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *GroupReviewHandler {
	return &GroupReviewHandler{ReviewHandler: &ReviewHandler{caller: caller, log: log}, groups: svc}
}

// Routes declares the group names' review routes on mux.
func (h *GroupReviewHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /terminology/review/diagnosis-groups", h.list)
	mux.HandleFunc("PUT /terminology/diagnosis-groups/{code}/names/{language}", h.edit)
	mux.HandleFunc("POST /terminology/diagnosis-groups/{code}/names/{language}/review", h.review)
}

// groupNameJSON is the DiagnosisGroupName schema.
type groupNameJSON struct {
	Code      string     `json:"code"`
	Parent    string     `json:"parent,omitempty"`
	Language  string     `json:"language"`
	Text      string     `json:"text"`
	Review    reviewJSON `json:"review"`
	VersionID int32      `json:"versionId"`
}

func groupNameOut(code, parent string, n GroupName) groupNameJSON {
	return groupNameJSON{Code: code, Parent: parent, Language: n.Language, Text: n.Text, Review: reviewOut(n.Review), VersionID: n.VersionID}
}

func (h *GroupReviewHandler) list(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.groups.List(r.Context(), c, r.URL.Query().Get("language"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]groupNameJSON, len(list))
	for i, g := range list {
		out[i] = groupNameOut(g.Code, g.Parent, g.Name)
	}
	httpx.JSON(w, http.StatusOK, map[string][]groupNameJSON{"groups": out})
}

func (h *GroupReviewHandler) edit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text *string `json:"text"`
	}
	c, version, ok := h.write(w, r, &req, func() bool { return req.Text != nil })
	if !ok {
		return
	}
	n, err := h.groups.EditGroupName(r.Context(), c, r.PathValue("code"), r.PathValue("language"), *req.Text, version)
	h.writeName(w, r, n, err)
}

func (h *GroupReviewHandler) review(w http.ResponseWriter, r *http.Request) {
	var req decisionJSON
	c, version, ok := h.write(w, r, &req, func() bool { return req.Status != nil })
	if !ok {
		return
	}
	n, err := h.groups.ReviewGroupName(r.Context(), c, r.PathValue("code"), r.PathValue("language"), req.decision(), version)
	h.writeName(w, r, n, err)
}

func (h *GroupReviewHandler) writeName(w http.ResponseWriter, r *http.Request, n GroupName, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, groupNameOut(r.PathValue("code"), "", n))
}
