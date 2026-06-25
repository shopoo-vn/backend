package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/marketplace/auth-service/internal/handler"
	"github.com/marketplace/auth-service/internal/middleware"
	"github.com/marketplace/auth-service/internal/token"
)

func New(h *handler.Handler, tm *token.Manager) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	})

	r.Route("/users", func(r chi.Router) {
		// Internal service-to-service lookup (chi prioritises the static
		// "/me" route over this param route, so they don't collide).
		r.Get("/{id}", h.GetUser)

		// Authenticated endpoints.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(tm))
			r.Get("/me", h.Me)
			r.Patch("/me", h.UpdateMe)
			// Admin-only (handlers enforce role=admin).
			r.Get("/", h.ListUsers)
			r.Patch("/{id}/status", h.SetUserStatus)
		})
	})

	return r
}
