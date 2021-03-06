package main

import (
	"errors"
	"net/http"
	"time"

	jdscheduler "github.com/ede0m/jdgoscheduler"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MasterSchedule is a jdscheudler output that is returned to all users
type MasterSchedule struct {
	ID              primitive.ObjectID         `json:"id" bson:"_id,omitempty"`
	Schedule        jdscheduler.Schedule       `json:"schedule" bson:"schedule"`
	ScheduleUnitMap map[string]ScheduleMapUnit `json:"scheduleUnitMap" bson:"scheduleUnitMap"`
	TradeLedger     []Trade                    `json:"tradeLedger" bson:"tradeLedger"`
	CreatedAt       time.Time                  `json:"createdAt" bson:"createdAt"`
	GroupID         primitive.ObjectID         `json:"groupId" bson:"groupId"`

	// TODO persist pick orders
}

// ScheduleMapUnit is a value of the MasterSchedule's OwnerMap
type ScheduleMapUnit struct {
	Owner       string    `json:"owner" bson:"owner"`
	Start       time.Time `json:"start" bson:"start"`
	MapIndicies []int     `json:"mapIndicies" bson:"mapIndicies"`
}

// MasterScheduleResponse is the response payload for MasterSchedule data model.
type MasterScheduleResponse struct {
	ID        primitive.ObjectID   `json:"id"`
	Schedule  jdscheduler.Schedule `json:"schedule"`
	CreatedAt time.Time            `json:"createdAt"`
	GroupID   primitive.ObjectID   `json:"groupId" `
}

// ScheduleResponse is the request payload for Scheudle data model.
type ScheduleResponse struct {
	Schedule jdscheduler.Schedule `json:"schedule"`
}

// MasterScheduleRequest is the request payload for creating master schedules for a group
type MasterScheduleRequest struct {
	Schedule jdscheduler.Schedule `json:"schedule"`
	GroupID  string               `json:"groupId"`
}

// ScheduleRequest is the request payload for generating schedules against the jdscheduler module
type ScheduleRequest struct {
	Start        time.Time // must pass in RFC3339 or UTC format
	SeasonUnits  int
	Years        int
	Participants []string
}

// NewMasterSchedule creates a new master schedule
func NewMasterSchedule(sch jdscheduler.Schedule, groupID primitive.ObjectID) (*MasterSchedule, error) {

	ownerMap := make(map[string]ScheduleMapUnit)
	for i, s := range sch.Seasons {
		for j, b := range s.Blocks {
			for k, unit := range b.Units {
				scm := ScheduleMapUnit{unit.Participant, unit.Start, []int{i, j, k}}
				ownerMap[unit.ID.String()] = scm
			}
		}
	}
	// TODO: get scheudle's scheudler pick order state, create trade log
	ms := &MasterSchedule{primitive.NilObjectID, sch, ownerMap, []Trade{}, time.Now(), groupID}
	return ms, nil
}

// NewMasterScheduleResponse creates a new master schedule
func NewMasterScheduleResponse(ms MasterSchedule) *MasterScheduleResponse {
	msr := &MasterScheduleResponse{ID: ms.ID, Schedule: ms.Schedule, CreatedAt: ms.CreatedAt, GroupID: ms.GroupID}
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
	if sr.Start.IsZero() || sr.Participants == nil {
		return errors.New("missing required StartDate, Participants fields")
	}
	// remove hh::mm::ss
	sr.Start = time.Date(sr.Start.Year(), sr.Start.Month(), sr.Start.Day(), 0, 0, 0, 0, sr.Start.Location())
	// defaults
	if sr.Years == 0 {
		sr.Years = 5
	}
	if sr.SeasonUnits == 0 {
		sr.SeasonUnits = 3
	}

	// TODO: handle user

	return nil
}

// Bind binds the http req to scheduleRequest type as the render
func (msr *MasterScheduleRequest) Bind(r *http.Request) error {
	if msr.GroupID == "" {
		return errors.New("must bind schedule to group")
	}

	// TODO: verify schedule is populated?
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
	start, wksPSeason, nSeason, participants := data.Start, data.SeasonUnits, data.Years, data.Participants
	s, err := jdscheduler.NewSchedule(start, nSeason, wksPSeason, participants)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, NewScheduleResponse(*s))
}

// CreateMasterSchedule commits a schedule as master to schedule
func CreateMasterSchedule(w http.ResponseWriter, r *http.Request) {
	data := &MasterScheduleRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	s := data.Schedule
	// group must exist
	groupID, err := primitive.ObjectIDFromHex(data.GroupID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	g := &Group{}
	if err = mh.GetGroup(g, bson.M{"_id": groupID}); err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	// schedule must be created by admin of group
	_, claims, _ := jwtauth.FromContext(r.Context())
	uid, _ := primitive.ObjectIDFromHex(claims["userID"].(string))
	if ok := g.HasAdmin(uid); !ok {
		render.Render(w, r, ErrAuth(errors.New("not authorized for this group")))
		return
	}
	ms, err := NewMasterSchedule(s, groupID)
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	result, err := mh.InsertMasterSchedule(ms)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	ms.ID = result.InsertedID.(primitive.ObjectID)
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewMasterScheduleResponse(*ms))
}

// GetMasterSchedule retrieves the current (most recent) master scheudle
func GetMasterSchedule(w http.ResponseWriter, r *http.Request) {
	gid := chi.URLParam(r, "groupID")
	groupID, err := primitive.ObjectIDFromHex(gid)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	ms := &MasterSchedule{}
	err = mh.GetMasterSchedule(ms, bson.M{"groupId": groupID})
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, NewMasterScheduleResponse(*ms))
}

func (ms *MasterSchedule) tradeScheduleUnits(t Trade) (jdscheduler.Schedule, map[string]ScheduleMapUnit) {

	sch := ms.Schedule
	suMap := ms.ScheduleUnitMap

	for _, u := range t.InitiatorTrades {
		smu := suMap[u.ID.String()]
		smu.Owner = t.ExecutorEmail
		indicies := smu.MapIndicies
		if len(indicies) == 3 {
			sch.Seasons[indicies[0]].Blocks[indicies[1]].Units[indicies[2]].Participant = t.ExecutorEmail
		} else {
			panic(errors.New("schedule map unit indicies corrupt"))
		}
	}

	for _, u := range t.ExecutorTrades {
		smu := suMap[u.ID.String()]
		smu.Owner = t.InitiatorEmail
		indicies := smu.MapIndicies
		if len(indicies) == 3 {
			sch.Seasons[indicies[0]].Blocks[indicies[1]].Units[indicies[2]].Participant = t.InitiatorEmail
		} else {
			panic(errors.New("schedule map unit indicies corrupt"))
		}
	}

	return sch, suMap
}
