package main

import (
	"net/http"

	jdscheduler "github.com/ede0m/jdgoscheduler"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

func main() {

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Route("/schedule", func(r chi.Router) {
		r.Post("/", GenerateSchedule)
	})

	http.ListenAndServe(":3000", r)
}

// GenerateSchedule just generates a scheudle with given query parameteres
func GenerateSchedule(w http.ResponseWriter, r *http.Request) {
	data := &ScheduleRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	start, wksPSeason, nSeason := data.StartDate, data.SeasonWeeks, data.Years
	participants := data.Participants
	s := jdscheduler.NewSchedule(start, nSeason, wksPSeason, participants)

	render.Status(r, http.StatusOK)
	render.Render(w, r, NewScheduleResponse(*s))
}
