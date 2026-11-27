package budget

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) PriorityGroups(w http.ResponseWriter, r *http.Request) {
	slog.Debug("budget.Handler.PriorityGroups: called")
	groups := h.service.PriorityGroups()
	helper.RespondJSON(w, http.StatusOK, groups)
}

func (h *Handler) Remaining(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}

	slog.Debug("budget.Handler.Remaining: called", "user_id", userID)

	result, err := h.service.Remaining(r.Context(), userID)
	if err != nil {
		slog.Error("budget.Handler.Remaining: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	helper.RespondJSON(w, http.StatusOK, result)
}

// MaxImportBytes caps an import body (coregateway ADR-0008).
const MaxImportBytes = 25 << 20

// Export is the budget part, for the owner or a targeted read.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}

	part, err := h.service.Export(r.Context(), userID)
	if err != nil {
		slog.Error("budget.Handler.Export: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+exportFileName(r.Context(), time.Now()))
	helper.RespondJSON(w, http.StatusOK, part)
}

// exportFileName dates the file in the owner's zone from the token, UTC
// when it carries none.
func exportFileName(ctx context.Context, now time.Time) string {
	loc := time.UTC
	if tz := auth.TimezoneFromContext(ctx); tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	return "coregateway-" + PartName + "-" + now.In(loc).Format(dateLayout) + ".json"
}

// Import loads a budget part, or an unversioned export (version 0), into the
// caller's own data. ?dry_run=true reports without writing. A file it can't
// read is a 400; a file it refuses is a 422 carrying the failed result.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxImportBytes))
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			helper.RespondErrorJSON(w, http.StatusRequestEntityTooLarge, "import file is larger than 25 MB")
			return
		}
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	data, version, err := ReadPart(raw)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	dryRun := r.URL.Query().Get("dry_run") == "true"
	result, err := h.service.Import(r.Context(), userID, data, version, dryRun)
	if err != nil {
		slog.Error("budget.Handler.Import: failed", "error", err, "user_id", userID)
		helper.RespondServiceError(w, err)
		return
	}
	status := http.StatusOK
	if result.Status == StatusFailed {
		status = http.StatusUnprocessableEntity
	}
	helper.RespondJSON(w, status, result)
}
