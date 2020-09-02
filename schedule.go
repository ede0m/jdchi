package main

import (
	"errors"
	"net/http"
	"time"

	jdscheduler "github.com/ede0m/jdgoscheduler"
)

// ScheduleRequest is the request payload for Scheudle data model.
type ScheduleRequest struct {
	StartDate    time.Time // must pass in RFC3339 or UTC format
	SeasonWeeks  int
	Years        int
	Participants []string
}

// ScheduleResponse is the request payload for Scheudle data model.
type ScheduleResponse struct {
	Schedule  jdscheduler.Schedule
	SchString string
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
	return nil
}

// NewScheduleResponse generates a new schedule resp out of a jdscheduler.scheudle
func NewScheduleResponse(sch jdscheduler.Schedule) *ScheduleResponse {
	resp := &ScheduleResponse{Schedule: sch, SchString: sch.String()}
	// any other properties i want to tag along on the response? ID, timestamp, etc.
	return resp
}

// Render is called in top-down order, like a http handler middleware chain.
func (rd *ScheduleResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	return nil
}
