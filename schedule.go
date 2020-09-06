package main

import (
	"errors"
	"net/http"
	"time"

	jdscheduler "github.com/ede0m/jdgoscheduler"
	"github.com/go-chi/render"
	"go.mongodb.org/mongo-driver/bson"
)

// MasterSchedule is a jdscheudler output that is returned to all users
type MasterSchedule struct {
	Schedule  jdscheduler.Schedule `json:"schedule" bson:"schedule"`
	CreatedAt time.Time            `json:"createdAt" bson:"createdAt"`
	// TODO persist pick orders
	// TODO persist trade log
}

// MasterScheduleResponse is the response payload for MasterSchedule data model.
type MasterScheduleResponse struct {
	MasterSchedule MasterSchedule `json:"masterSchedule"`
}

// ScheduleResponse is the request payload for Scheudle data model.
type ScheduleResponse struct {
	Schedule jdscheduler.Schedule `json:"schedule"`
}

// ScheduleRequest is the request payload for generating schedules against the jdscheduler module
type ScheduleRequest struct {
	StartDate    time.Time // must pass in RFC3339 or UTC format
	SeasonWeeks  int
	Years        int
	Participants []string
	User         string
}

// NewMasterSchedule creates a new master schedule
func NewMasterSchedule(s jdscheduler.Schedule) *MasterSchedule {
	// TODO: get scheudle's scheudler pick order state, create trade log
	ms := &MasterSchedule{s, time.Now()}
	return ms
}

// NewMasterScheduleResponse creates a new master schedule
func NewMasterScheduleResponse(ms MasterSchedule) *MasterScheduleResponse {
	// TODO: get scheudle's scheudler pick order state, create trade log
	msr := &MasterScheduleResponse{MasterSchedule: ms}
	return msr
}

// NewScheduleResponse generates a new schedule resp out of a jdscheduler.scheudle
func NewScheduleResponse(sch jdscheduler.Schedule) *ScheduleResponse {
	resp := &ScheduleResponse{Schedule: sch}
	// any other properties i want to tag along on the response? ID, timestamp, etc.
	return resp
}

// Render is called in top-down order, like a http handler middleware chain.
func (rd *ScheduleResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	return nil
}

// Render is called in top-down order, like a http handler middleware chain.
func (rd *MasterScheduleResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	return nil
}

// Bind binds the http req to scheduleRequest type as the render
func (sr *ScheduleRequest) Bind(r *http.Request) error {
	// error handle
	if sr.StartDate.IsZero() || sr.Participants == nil {
		return errors.New("missing required StartDate, Participants fields")
	}
	// remove hh::mm::ss
	sr.StartDate = time.Date(sr.StartDate.Year(), sr.StartDate.Month(), sr.StartDate.Day(), 0, 0, 0, 0, sr.StartDate.Location())
	// defaults
	if sr.Years == 0 {
		sr.Years = 5
	}
	if sr.SeasonWeeks == 0 {
		sr.SeasonWeeks = 3
	}

	// TODO: handle user

	return nil
}

////////////  CONTROLLERS //////////////////

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
	err := mh.GetMasterSchedule(ms, bson.M{})
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, NewMasterScheduleResponse(*ms))
}
