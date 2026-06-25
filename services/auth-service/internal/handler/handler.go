package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/marketplace/auth-service/internal/middleware"
	"github.com/marketplace/auth-service/internal/repo"
	"github.com/marketplace/auth-service/internal/service"
)

type Handler struct {
	auth *service.AuthService
}

func New(auth *service.AuthService) *Handler { return &Handler{auth: auth} }

// ── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ── auth endpoints ───────────────────────────────────────────────────────────

type registerReq struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "email, password and display_name are required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	u, err := h.auth.Register(r.Context(), req.Email, req.Phone, req.Password, req.DisplayName)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "could not register user")
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	u, pair, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, err.Error())
		case errors.Is(err, service.ErrUserInactive):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "could not login")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "tokens": pair})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}
	pair, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}
	writeJSON(w, http.StatusOK, pair)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}
	if err := h.auth.Logout(r.Context(), req.RefreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "could not logout")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── user endpoints ───────────────────────────────────────────────────────────

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.auth.GetProfile(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

type updateProfileReq struct {
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}
	u, err := h.auth.UpdateProfile(r.Context(), middleware.UserID(r.Context()), req.DisplayName, req.AvatarURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// GetUser is an internal endpoint for sibling services to fetch a user by id.
// In production lock this down to the internal network / service auth.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.auth.GetProfile(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch user")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// ── admin endpoints (role=admin required) ────────────────────────────────────

func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if middleware.Role(r.Context()) != "admin" {
		writeError(w, http.StatusForbidden, "admin role required")
		return false
	}
	return true
}

// ListUsers returns a paginated list of users (admin console).
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	res, err := h.auth.ListUsers(r.Context(), page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type updateStatusReq struct {
	Status string `json:"status"`
}

// SetUserStatus bans/unbans a user.
func (h *Handler) SetUserStatus(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	var req updateStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	u, err := h.auth.SetUserStatus(r.Context(), chi.URLParam(r, "id"), req.Status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidStatus):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repo.ErrNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		default:
			writeError(w, http.StatusInternalServerError, "could not update status")
		}
		return
	}
	writeJSON(w, http.StatusOK, u)
}
