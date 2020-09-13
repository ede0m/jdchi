package main

import (
	"errors"
	"net/http"
	"time"

	jdchaimailer "github.com/ede0m/jdchai/mailer"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SystemInviteRequest is a request by a group admin to create users in a group and "invite" them to the system
type SystemInviteRequest struct {
	GroupID      string   `json:"groupId"`
	MemberEmails []string `json:"memberEmails"`
}

// AcceptSystemInviteRequest is a request by an invitee to sign up in the system and group
type AcceptSystemInviteRequest struct {
	GroupID   string `json:"groupId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

// SystemInviteResponse is a request to create users in a group and "invite" them to system
type SystemInviteResponse struct {
}

// AcceptInviteResponse is a request to create users in a group and "invite" them
type AcceptInviteResponse struct {
}

// Bind binds the http req to SystemInviteRequest type as the render
func (si *SystemInviteRequest) Bind(r *http.Request) error {
	if si.GroupID == "" {
		return errors.New("must specify group id to send system invite")
	}
	if len(si.MemberEmails) == 0 {
		return errors.New("must have at least one member")
	}
	return nil
}

// Bind binds the http req to AcceptSystemInviteRequest type as the render
func (si *AcceptSystemInviteRequest) Bind(r *http.Request) error {
	if si.GroupID == "" {
		return errors.New("must specify group id to accept invite")
	}
	if si.Password == "" {
		return errors.New("no password received")
	}
	if si.FirstName == "" || si.LastName == "" {
		return errors.New("enter a first name and last name")
	}
	return nil
}

// NewSystemInviteResponse returns a client response for a group
func NewSystemInviteResponse() *SystemInviteResponse {
	return &SystemInviteResponse{}
}

// Render is called in top-down order, like a http handler middleware chain.
func (ur *SystemInviteResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

/*CreateInvites is used by a group admin to create accounts (that must not exist in the system)
and add those accounts to a group. It will fire off emails for the user to "update" their account password
and info (which will set activate)
*/
func CreateInvites(w http.ResponseWriter, r *http.Request) {
	data := &SystemInviteRequest{}
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
	if ok := g.HasAdmin(uid); !ok {
		render.Render(w, r, ErrAuth(errors.New("not authorized for this group")))
		return
	}

	// create users and email in mailer thread
	rr := RegisterRequest{"", "", "", "password", gid}
	for _, m := range data.MemberEmails {
		// TODO: verify email
		rr.Email = m
		u, err := NewUser(rr)
		if err != nil {
			if u != nil {
				// user exists
				if !u.inGroup(gid) {
					var update = bson.M{"$addToSet": bson.M{"groups": gid}}
					if err := mh.UpdateUsers(bson.M{"_id": u.ID}, update); err != nil {
						render.Render(w, r, ErrServer(err))
						return
					}
					go jdchaimailer.SendGroupInvite(g.Name, u.FirstName, u.Email)
				}
				continue
			} else {
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}
		}
		// user not in system, so we create and send welcome registration
		uid, err := mh.InsertUser(u)
		if uID, ok := uid.InsertedID.(primitive.ObjectID); ok {
			jwt := createTokenString(uID.Hex(), 30*24*time.Hour) // expires in 30 days for "activate"
			go jdchaimailer.SendWelcomRegistration(g.Name, u.Email, jwt)
		}
	}
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewSystemInviteResponse())
}

func AcceptSystemInvite(w http.ResponseWriter, r *http.Request) {
	data := &AcceptSystemInviteRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// TODO: set firstName, lastName, password, activatedAt,

}
