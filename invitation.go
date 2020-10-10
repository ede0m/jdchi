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
	"golang.org/x/crypto/bcrypt"
)

// InviteRequest is a request by a group admin to create users in a group and "invite" them to the system
type InviteRequest struct {
	GroupID      string   `json:"groupId"`
	MemberEmails []string `json:"memberEmails"`
}

// AcceptRegisterInviteRequest is a request by an invitee to sign up in the system and group
type AcceptRegisterInviteRequest struct {
	GroupID   string `json:"groupId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

// InviteResponse is a request to create users in a group and "invite" them to system
type InviteResponse struct{}

// AcceptRegisterInviteResponse is a request to create users in a group and "invite" them
type AcceptRegisterInviteResponse struct{}

// Bind binds the http req to SystemInviteRequest type as the render
func (si *InviteRequest) Bind(r *http.Request) error {
	if si.GroupID == "" {
		return errors.New("must specify group id to send system invite")
	}
	if len(si.MemberEmails) == 0 {
		return errors.New("must have at least one member")
	}
	return nil
}

// Bind binds the http req to AcceptSystemInviteRequest type as the render
func (si *AcceptRegisterInviteRequest) Bind(r *http.Request) error {
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

// NewInviteResponse returns a client response for a group
func NewInviteResponse() *InviteResponse {
	return &InviteResponse{}
}

// Render is called in top-down order, like a http handler middleware chain.
func (ir *InviteResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

//NewAcceptRegisterInviteResponse returns empty obj
func NewAcceptRegisterInviteResponse() *AcceptRegisterInviteResponse {
	return &AcceptRegisterInviteResponse{}
}

// Render is called in top-down order, like a http handler middleware chain.
func (ir *AcceptRegisterInviteResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

/*CreateInvites is used by a group admin to create or update accounts
with a valid group. It will fire off emails depending on the user's current system status.
*/
func CreateInvites(w http.ResponseWriter, r *http.Request) {
	data := &InviteRequest{}
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

	// create or update users and email in mailer thread
	rr := RegisterRequest{"", "", "", "password", gid}
	for _, m := range data.MemberEmails {
		// TODO: verify email
		rr.Email = m
		u, err := NewUser(rr)
		if err != nil {
			if u != nil {
				// user exists
				if !u.inGroup(gid) {
					update := bson.M{"$addToSet": bson.M{"groups": gid}}
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
			link := clientBaseURL + "register?token=" + jwt + "&group=" + g.Name + "&groupID=" + g.ID.Hex()
			go jdchaimailer.SendWelcomRegistration(g.Name, u.Email, link)
		}
	}
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewInviteResponse())
}

// AcceptRegisterInvite udates a user in the system with registeration details
func AcceptRegisterInvite(w http.ResponseWriter, r *http.Request) {
	data := &AcceptRegisterInviteRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	// must be made by admin of group to create
	_, claims, _ := jwtauth.FromContext(r.Context())
	uid, _ := primitive.ObjectIDFromHex(claims["userID"].(string))

	pwd, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(errors.New("register failure genp")))
		return
	}
	update := bson.M{"$set": bson.M{"password": pwd, "firstName": data.FirstName, "lastName": data.LastName, "activatedAt": time.Now()}}
	if err := mh.UpdateUsers(bson.M{"_id": uid}, update); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, NewAcceptRegisterInviteResponse())
}
