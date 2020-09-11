package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
)

// mongo client
var mh = NewMongoHandler()

func main() {

	defer mh.client.Disconnect(context.Background())

	// set up routes
	r := chi.NewRouter()
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
			r.Post("/invitation", CreateGroupInvite)
		})
		r.Route("/user", func(r chi.Router) {
			//r.Put("/", UpdateUser)
		})
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", LoginUser)
	})

	http.ListenAndServe(":3000", r)
}
