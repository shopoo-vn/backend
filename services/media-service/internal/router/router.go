package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/marketplace/media-service/internal/handler"
	"github.com/marketplace/media-service/internal/middleware"
	"github.com/marketplace/media-service/internal/token"
)

func New(h *handler.Handler, v *token.Verifier) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/media", func(r chi.Router) {
		// Public read: object URLs are public anyway.
		r.Get("/{id}", h.Get)

		// Authenticated write endpoints.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(v))
			r.Post("/upload", h.Upload)
			r.Delete("/{id}", h.Delete)
		})
	})

	return r
}
