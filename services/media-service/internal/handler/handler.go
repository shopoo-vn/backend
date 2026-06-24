package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/marketplace/media-service/internal/middleware"
	"github.com/marketplace/media-service/internal/service"
)

type Handler struct {
	media          *service.MediaService
	maxUploadBytes int64
}

func New(media *service.MediaService, maxUploadBytes int64) *Handler {
	return &Handler{media: media, maxUploadBytes: maxUploadBytes}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ── endpoints ────────────────────────────────────────────────────────────────

// Upload accepts a multipart form with a single "file" image part, validates
// mime + size, produces the derived sizes and returns {id, urls}.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// Cap the whole request body so an oversized upload can't exhaust memory.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes+(1<<20))

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing multipart form field \"file\"")
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadBytes {
		writeError(w, http.StatusBadRequest, "file exceeds maximum size of 10MB")
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, h.maxUploadBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read uploaded file")
		return
	}
	if int64(len(data)) > h.maxUploadBytes {
		writeError(w, http.StatusBadRequest, "file exceeds maximum size of 10MB")
		return
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, "uploaded file is empty")
		return
	}

	// Determine the content type from the bytes (don't trust the client header)
	// and require an image.
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		writeError(w, http.StatusBadRequest, "unsupported content type: image required")
		return
	}

	m, err := h.media.Upload(r.Context(), middleware.UserID(r.Context()), data, header.Filename, contentType)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedType) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "could not process upload")
		return
	}

	view := h.media.ToView(m)
	writeJSON(w, http.StatusCreated, map[string]any{"id": view.ID, "urls": view.URLs})
}

// Get returns the full media record by id.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	m, err := h.media.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "media not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch media")
		return
	}
	writeJSON(w, http.StatusOK, h.media.ToView(m))
}

// Delete removes the media (owner only) and returns 204.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.media.Delete(r.Context(), chi.URLParam(r, "id"), middleware.UserID(r.Context()))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, "media not found")
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not allowed to delete this media")
		default:
			writeError(w, http.StatusInternalServerError, "could not delete media")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
