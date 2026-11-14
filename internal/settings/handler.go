// Package settings stores per-user settings (#9). Today that is the user's
// timezone, which decides the date rule entries land on.
package settings

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

// DefaultTimezone is reported for users who haven't saved one. Their rule
// entries keep the server clock's date, which is UTC in every deploy.
const DefaultTimezone = "UTC"

// Settings is the GET/PUT body. Saved is false while the user still has the
// default, so the web can adopt the device's timezone once.
type Settings struct {
	Timezone string `json:"timezone"`
	Saved    bool   `json:"saved"`
}

type putRequest struct {
	Timezone string `json:"timezone"`
}

type Handler struct {
	queries db.Querier
}

func NewHandler(queries db.Querier) *Handler {
	return &Handler{queries: queries}
}

// validTimezone reports whether tz is an IANA zone name such as Asia/Manila.
// LoadLocation also accepts "" and "Local" (the server's zone), which mean
// nothing for a user, so those are refused.
func validTimezone(tz string) bool {
	if tz == "" || tz == "Local" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	row, err := h.queries.GetUserSettings(r.Context(), userID)
	if errors.Is(err, pgx.ErrNoRows) {
		helper.RespondJSON(w, http.StatusOK, Settings{Timezone: DefaultTimezone})
		return
	}
	if err != nil {
		slog.Error("settings.Get: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, "couldn't load settings")
		return
	}
	helper.RespondJSON(w, http.StatusOK, Settings{Timezone: row.Timezone, Saved: true})
}

func (h *Handler) Put(w http.ResponseWriter, r *http.Request) {
	var req putRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validTimezone(req.Timezone) {
		helper.RespondErrorJSON(w, http.StatusBadRequest, "timezone must be an IANA zone such as Asia/Manila")
		return
	}
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	row, err := h.queries.PutUserSettings(r.Context(), db.PutUserSettingsParams{UserID: userID, Timezone: req.Timezone})
	if err != nil {
		slog.Error("settings.Put: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, "couldn't save settings")
		return
	}
	slog.Info("settings.Put: timezone saved", "user_id", userID, "timezone", row.Timezone)
	helper.RespondJSON(w, http.StatusOK, Settings{Timezone: row.Timezone, Saved: true})
}
