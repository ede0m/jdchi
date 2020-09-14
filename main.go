package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
)

// mongo client
var mh = NewMongoHandler()

func main() {

	defer mh.client.Disconnect(context.Background())

	r := chi.NewRouter()

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins:   []string{"*"},
		AllowOriginFunc:  AllowOriginFunc,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Timeout"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Use(middleware.Logger)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Route("/schedule", func(r chi.Router) {
		r.Post("/", GenerateSchedule)
		// Protected routes
		r.Group(func(r chi.Router) {
			// Seek, verify and validate JWT tokens
			r.Use(jwtauth.Verifier(tokenAuth))
			// Handle valid / invalid tokens.
			r.Use(jwtauth.Authenticator)

			r.Post("/master", CreateMasterSchedule)
			r.Get("/master/{groupID}", GetMasterSchedule)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)
		r.Route("/group", func(r chi.Router) {
			r.Post("/", CreateGroup)
			r.Post("/invitation", CreateInvites)
		})
		r.Route("/user", func(r chi.Router) {
			r.Patch("/invitation", AcceptRegisterInvite)
		})
	})

	r.Route("/session", func(r chi.Router) {
		r.Post("/", LoginUser)
	})

	http.ListenAndServe(":3000", r)
}

// AllowOriginFunc logic for cors
func AllowOriginFunc(r *http.Request, origin string) bool {
	return true
}
