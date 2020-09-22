package main

import (
	"errors"
	"net/http"
	"time"

	jdchaimailer "github.com/ede0m/jdchai/mailer"
	jdscheduler "github.com/ede0m/jdgoscheduler"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group defines a group for a scheudle
type Group struct {
	ID      primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name    string               `json:"name" bson:"name"`
	Admins  []primitive.ObjectID `json:"admins" bson:"admins"`
	Members []primitive.ObjectID `json:"members" bson:"members"`
}

// GroupRequest is a request to create a new group
type GroupRequest struct {
	Name         string               `json:"name"`
	AdminEmails  []string             `json:"adminEmails"`
	MemberEmails []string             `json:"memberEmails"`
	Schedule     jdscheduler.Schedule `json:"schedule"`
}

// GroupResponse is a client response of a group
type GroupResponse struct {
	ID            primitive.ObjectID `json:"id"`
	Name          string             `json:"name"`
	NParticipants int                `json:"nParticipants"`
}

// GroupUsersResponse response for all users in a group
type GroupUsersResponse struct {
	Members []GroupUserResponse `json:"members"`
}

// NewGroup creates a group with admins. will check that every listed admin exists
func NewGroup(gr GroupRequest) (*Group, error) {

	foundUsers, err := mh.GetUsers(bson.M{"email": bson.M{"$in": gr.AdminEmails}})
	if err != nil {
		return nil, err
	}
	if len(foundUsers) != len(gr.AdminEmails) {
		return nil, errors.New("one or more admin emails not found in system")
	}

	g := &Group{}
	err = mh.GetGroup(g, bson.M{"name": gr.Name})
	if err == nil {
		return nil, errors.New("group name: " + gr.Name + " aready exists")
	}

	adminIds := make([]primitive.ObjectID, 0)
	for _, u := range foundUsers {
		adminIds = append(adminIds, u.ID)
	}
	// members empty initially because we may need to create new users
	memberIds := make([]primitive.ObjectID, 0)
	group := &Group{primitive.NilObjectID, gr.Name, adminIds, memberIds}
	return group, nil
}

// NewGroupResponse returns a client response for a group
func NewGroupResponse(g Group) *GroupResponse {
	return &GroupResponse{g.ID, g.Name, len(g.Members)}
}

// NewGroupUsersResponse groupUser representation from user slice
func NewGroupUsersResponse(users []*User) *GroupUsersResponse {
	var groupUsers []GroupUserResponse
	for _, u := range users {
		groupUsers = append(groupUsers, *NewGroupUserResponse(*u))
	}
	return &GroupUsersResponse{groupUsers}
}

// Render is called in top-down order, like a http handler middleware chain.
func (gr *GroupResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Render is called in top-down order, like a http handler middleware chain.
func (gur *GroupUsersResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Bind binds the http req to groupRequest type as the render
func (gr *GroupRequest) Bind(r *http.Request) error {

	if gr.Name == "" {
		return errors.New("group must have a name")
	} else if len(gr.AdminEmails) > 5 {
		return errors.New("cannot have more than 5 admins")
	} else if len(gr.AdminEmails) == 0 {
		return errors.New("must have at least one admin")
	} else if len(gr.MemberEmails) == 0 {
		return errors.New("must have at least one member")
	} else if len(gr.Schedule.Participants) == 0 {
		return errors.New("must submit with valid schedule")
	}
	return nil
}

////////////  CONTROLLERS //////////////////

// GetGroupUsers gets users in a group
func GetGroupUsers(w http.ResponseWriter, r *http.Request) {
	gid := chi.URLParam(r, "groupID")
	groupID, err := primitive.ObjectIDFromHex(gid)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	g := &Group{}
	err = mh.GetGroup(g, bson.M{"_id": groupID})
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	users, err := mh.GetUsers(bson.M{"_id": bson.M{"$in": g.Members}})
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewGroupUsersResponse(users))
}

// CreateGroup creates a new group
func CreateGroup(w http.ResponseWriter, r *http.Request) {

	data := &GroupRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	g, err := NewGroup(*data)
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}

	ms, err := NewMasterSchedule(data.Schedule, primitive.NilObjectID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// get users already in system
	existingUsers := make([]*User, 0)
	newUsers := make([]*User, 0)
	rr := RegisterRequest{"", "", "", "mehpwd", primitive.NilObjectID} //TODO randomize
	for _, addr := range data.MemberEmails {
		// TODO: verify email
		rr.Email = addr
		u, err := NewUser(rr)
		if err != nil {
			if u != nil {
				// user exists
				existingUsers = append(existingUsers, u)
				continue
			} else {
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}
		}
		// user not in system, so we create
		newUsers = append(newUsers, u)
	}

	result, err := mh.InsertGroup(g, ms, newUsers, existingUsers)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}

	group := &Group{}
	mh.GetGroup(group, bson.M{"_id": result})

	// send out invites
	for _, u := range existingUsers {
		go jdchaimailer.SendGroupInvite(group.Name, u.FirstName, u.Email)
	}
	for _, u := range newUsers {
		err := mh.GetUser(u, bson.M{"email": u.Email})
		if err == nil {
			jwt := createTokenString(u.ID.Hex(), 30*24*time.Hour) // expires in 30 days for "activate"
			link := clientBaseURL + "register?token=" + jwt + "&group=" + group.Name + "&groupID=" + group.ID.Hex()
			go jdchaimailer.SendWelcomRegistration(group.Name, u.Email, link)
		}
	}
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewGroupResponse(*group))
}

// HasAdmin checks whether or not a user is admin of a group
func (g Group) HasAdmin(uid primitive.ObjectID) bool {
	for _, u := range g.Admins {
		if u == uid {
			return true
		}
	}
	return false
}
