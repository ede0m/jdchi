package main

import (
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
)

var nilTime time.Time

// User for login/register
type User struct {
	ID          primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Email       string               `json:"email" bson:"email"`
	Password    string               `json:"password" bson:"password"`
	FirstName   string               `json:"firstName" bson:"firstName"`
	LastName    string               `json:"lastName" bson:"lastName"`
	CreatedAt   time.Time            `json:"createdAt" bson:"createdAt"`
	ActivatedAt time.Time            `json:"activatedAt" bson:"activatedAt"`
	Groups      []primitive.ObjectID `json:"groups" bson:"groups"`
}

// RegisterRequest request
type RegisterRequest struct {
	FirstName string             `json:"firstName"`
	LastName  string             `json:"lastName"`
	Email     string             `json:"email"`
	Password  string             `json:"password"`
	GroupID   primitive.ObjectID `json:"group"`
}

// LoginRequest for logins
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResponse a typical response for login/register
type UserResponse struct {
	FirstName string          `json:"firstName"`
	LastName  string          `json:"lastName"`
	Groups    []GroupResponse `json:"groups"`
	Token     string          `json:"token"`
}

// GroupUserResponse is a group's representation of a user
type GroupUserResponse struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

// NewUser constructor for a new User. hash password
func NewUser(rr RegisterRequest) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rr.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("register failure genp")
	}
	u := &User{}
	err = mh.GetUser(u, bson.M{"email": rr.Email})
	if err == nil {
		// user exists
		return u, errors.New("email " + rr.Email + " already registered")
	}
	createdAt := time.Now()
	// first group
	groups := make([]primitive.ObjectID, 1)
	groups[0] = rr.GroupID
	return &User{primitive.NilObjectID, rr.Email, string(hashedPassword), rr.FirstName, rr.LastName, createdAt, nilTime, groups}, nil
}

// NewUserResponse constructor for UserResponse
func NewUserResponse(u User) *UserResponse {
	var groups []GroupResponse
	for _, gid := range u.Groups {
		group := &Group{}
		mh.GetGroup(group, bson.M{"_id": gid})
		groups = append(groups, *NewGroupResponse(*group))
	}
	jwt := createTokenString(u.ID.Hex(), 15*time.Minute) // expires in 15 mins
	return &UserResponse{u.FirstName, u.LastName, groups, jwt}
}

// NewGroupUserResponse returns group user from a user
func NewGroupUserResponse(u User) *GroupUserResponse {
	return &GroupUserResponse{u.FirstName, u.LastName, u.Email}
}

// Render is called in top-down order, like a http handler middleware chain.
func (ur *UserResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Bind binds the http req to scheduleRequest type as the render
func (rr *RegisterRequest) Bind(r *http.Request) error {
	// TODO: check email is valid?
	if rr.Email == "" {
		return errors.New("improper email")
	}
	if rr.FirstName == "" || rr.LastName == "" {
		return errors.New("enter a first name and last name")
	}
	return nil
}

// Bind binds the http req to scheduleRequest type as the render
func (rr *LoginRequest) Bind(r *http.Request) error {
	// TODO: check email is valid?
	if rr.Email == "" {
		return errors.New("improper email")
	}
	return nil
}

////////////  CONTROLLERS ////////////////////

// RegisterUser registers user to system
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	data := &RegisterRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	u, err := NewUser(*data)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	_, err = mh.InsertUser(u)
	if err != nil {
		render.Render(w, r, ErrConflict(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewUserResponse(*u))
}

// LoginUser logs in a user after checking credentials. generates a new jwt for auth
func LoginUser(w http.ResponseWriter, r *http.Request) {
	data := &LoginRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	user := &User{}
	err := mh.GetUser(user, bson.M{"email": data.Email})
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.Password))
	if err != nil {
		render.Render(w, r, ErrAuth(err))
		return
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, NewUserResponse(*user))
}

func (u User) inGroup(gid primitive.ObjectID) bool {
	for _, gID := range u.Groups {
		if gID == gid {
			return true
		}
	}
	return false
}

func createTokenString(userID string, expiresIn time.Duration) string {
	_, tokenString, _ := tokenAuth.Encode(jwt.MapClaims{"userID": userID, "exp": jwtauth.ExpireIn(expiresIn)})
	return tokenString
}
