package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/marketplace/noti-service/internal/middleware"
	"github.com/marketplace/noti-service/internal/service"
)

type Handler struct {
	noti *service.NotiService
}

func New(noti *service.NotiService) *Handler { return &Handler{noti: noti} }

// ── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ── device-token endpoints ───────────────────────────────────────────────────

type registerDeviceReq struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// RegisterDevice upserts the caller's device token. POST /devices.
func (h *Handler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req registerDeviceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	dt, err := h.noti.RegisterDevice(r.Context(), middleware.UserID(r.Context()), req.Token, req.Platform)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "token is required")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not register device")
		return
	}
	writeJSON(w, http.StatusCreated, dt)
}

// UnregisterDevice removes the caller's device token. DELETE /devices/{token}.
func (h *Handler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	tokenStr := chi.URLParam(r, "token")
	if tokenStr == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	err := h.noti.UnregisterDevice(r.Context(), middleware.UserID(r.Context()), tokenStr)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "token is required")
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, "device token not found")
		default:
			writeError(w, http.StatusInternalServerError, "could not remove device")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
