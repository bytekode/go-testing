package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	// register middleware
	mux.Use(middleware.Recoverer)
	// mux.Use(app.enableCORS)

	// authentication routes - auth handler, refresh token
	mux.Post("/auth", app.authenticate)
	mux.Post("/refresh-token", app.refresh)

	// test handler

	// protected routes
	mux.Route("/users", func(r chi.Router) {
		// use auth middleware
		r.Get("/", app.allUsers)
		r.Get("/{userID}", app.getUser)
		r.Delete("/{userID}", app.deleteUser)
		r.Put("/{userID}", app.insertUser)
		r.Patch("/{userID}", app.updateUser)
	})

	return mux
}
