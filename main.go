package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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
		r.Post("/master", CreateMasterSchedule)
		r.Get("/master", GetMasterSchedule)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", RegisterUser)
	})

	http.ListenAndServe(":3000", r)
}
