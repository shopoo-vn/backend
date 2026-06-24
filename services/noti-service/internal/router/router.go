package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/marketplace/noti-service/internal/handler"
	"github.com/marketplace/noti-service/internal/middleware"
	"github.com/marketplace/noti-service/internal/token"
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

	// Device-token management — all behind Bearer auth.
	r.Route("/devices", func(r chi.Router) {
		r.Use(middleware.Auth(v))
		r.Post("/", h.RegisterDevice)
		r.Delete("/{token}", h.UnregisterDevice)
	})

	return r
}
