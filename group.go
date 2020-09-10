package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/jwtauth"
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
	Name        string   `json:"name"`
	AdminEmails []string `json:"adminEmails"`
}

// GroupInviteRequest is a request to create users in a group and "invite" them
type GroupInviteRequest struct {
	GroupID      string   `json:"groupId"`
	MemberEmails []string `json:"memberEmails"`
}

// GroupInviteResponse is a request to create users in a group and "invite" them
type GroupInviteResponse struct {
}

// GroupResponse is a client response of a group
type GroupResponse struct {
	ID   primitive.ObjectID `json:"id"`
	Name string             `json:"name"`
}

// NewGroup creates a group with admins. will check that every listed admin exists
func NewGroup(gr GroupRequest) (*Group, error) {

	foundUsers, err := mh.GetUsers(bson.M{"email": bson.M{"$in": gr.AdminEmails}})
	if err != nil {
		return nil, err
	}
	if len(foundUsers) != len(gr.AdminEmails) {
		return nil, errors.New("one or more emails not found in system")
	}

	g := &Group{}
	err = mh.GetGroup(g, bson.M{"name": gr.Name})
	if err == nil {
		return nil, errors.New("group name: " + gr.Name + " aready exists")
	}

	var adminIds []primitive.ObjectID
	for _, u := range foundUsers {
		adminIds = append(adminIds, u.ID)
	}
	// set members as admins initially
	members := make([]primitive.ObjectID, len(adminIds))
	copy(members, adminIds)

	group := &Group{primitive.NilObjectID, gr.Name, adminIds, members}
	return group, nil
}

// NewGroupResponse returns a client response for a group
func NewGroupResponse(g Group) *GroupResponse {
	return &GroupResponse{g.ID, g.Name}
}

// NewGroupInviteResponse returns a client response for a group
func NewGroupInviteResponse() *GroupInviteResponse {
	return &GroupInviteResponse{}
}

// Render is called in top-down order, like a http handler middleware chain.
func (ur *GroupResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Render is called in top-down order, like a http handler middleware chain.
func (ur *GroupInviteResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Bind binds the http req to groupRequest type as the render
func (gr *GroupRequest) Bind(r *http.Request) error {

	if gr.Name == "" {
		return errors.New("group must have a name")
	}
	if len(gr.AdminEmails) > 5 {
		return errors.New("cannot have more than 5 admins")
	} else if len(gr.AdminEmails) == 0 {
		return errors.New("must have at least one admin")
	}
	return nil
}

// Bind binds the http req to GroupInviteRequest type as the render
func (gr *GroupInviteRequest) Bind(r *http.Request) error {
	if gr.GroupID == "" {
		return errors.New("must specify group id")
	}
	if len(gr.MemberEmails) == 0 {
		return errors.New("must have at least one member")
	}
	return nil
}

////////////  CONTROLLERS //////////////////

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
	result, err := mh.InsertGroup(g)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	g.ID = result
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewGroupResponse(*g))
}

/*CreateGroupInvite is used by a group admin to create accounts (that must not exist in the system)
and add those accounts to a group. It will fire off emails for the user to "update" their account password
and info (which will set activate)
*/
func CreateGroupInvite(w http.ResponseWriter, r *http.Request) {
	data := &GroupInviteRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	// find group
	gid, err := primitive.ObjectIDFromHex(data.GroupID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	g := &Group{}
	err = mh.GetGroup(g, bson.M{"_id": gid})
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	// must be made by admin of group to create
	_, claims, _ := jwtauth.FromContext(r.Context())
	uid, _ := primitive.ObjectIDFromHex(claims["userID"].(string))
	validAdmin := false
	for _, u := range g.Admins {
		if u == uid {
			validAdmin = true
			break
		}
	}
	if !validAdmin {
		render.Render(w, r, ErrAuth(errors.New("not authorized for this group")))
	}

	// setup users
	var users []*User
	rr := RegisterRequest{"", "", "", "password", gid}
	for _, m := range data.MemberEmails {
		// TODO: verify email
		rr.Email = m
		u, err := NewUser(rr)
		if err != nil {
			render.Render(w, r, ErrServer(err))
			return
		}
		users = append(users, u)
	}
	// create users and add them to groups
	userIDs, err := mh.InsertGroupUsers(users, gid)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	// TODO emails + jwt?
	for _, u := range userIDs.InsertedIDs {
		if u, ok := u.(primitive.ObjectID); ok {
			jwt := createTokenString(u.Hex(), 3*24*time.Hour) // expires in 3 days for "activate"
		}
		// TODO send email??
	}

	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewGroupInviteResponse())
}
