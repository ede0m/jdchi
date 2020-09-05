package main

import (
	"context"
	"net/http"

	jdscheduler "github.com/ede0m/jdgoscheduler"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	http.ListenAndServe(":3000", r)
}

// GenerateSchedule just generates a scheudle with given query parameteres
func GenerateSchedule(w http.ResponseWriter, r *http.Request) {
	data := &ScheduleRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	start, wksPSeason, nSeason, participants := data.StartDate, data.SeasonWeeks, data.Years, data.Participants
	s := jdscheduler.NewSchedule(start, nSeason, wksPSeason, participants)

	render.Status(r, http.StatusOK)
	render.Render(w, r, NewScheduleResponse(*s))
}

// CreateMasterSchedule commits a schedule as master to schedule
func CreateMasterSchedule(w http.ResponseWriter, r *http.Request) {
	data := &ScheduleRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	start, wksPSeason, nSeason, participants := data.StartDate, data.SeasonWeeks, data.Years, data.Participants
	s := jdscheduler.NewSchedule(start, nSeason, wksPSeason, participants)
	ms := NewMasterSchedule(*s)
	_, err := mh.InsertMasterSchedule(ms)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewMasterScheduleResponse(*ms))
}

// GetMasterSchedule retrieves the current (most recent) master scheudle
func GetMasterSchedule(w http.ResponseWriter, r *http.Request) {
	ms := &MasterSchedule{}

	opts := options.FindOne()
	opts.SetSort(bson.D{{"createdat", -1}})
	/*
		It is better to explain it using 2 parameters, e.g. search for a = 1 and b = 2.
		Your syntax will be: bson.M{"a": 1, "b": 2} or bson.D{{"a": 1}, {"b": 2}}

		bson.D respects order
	*/

	err := mh.GetOne(ms, bson.M{}, nil)
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, NewMasterScheduleResponse(*ms))
}
