package main

import (
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
)

var tokenAuth *jwtauth.JWTAuth = jwtauth.New("HS256", []byte(JWTSecret), nil)

// User for login/register
type User struct {
	Email     string    `json:"email" bson:"email"`
	Password  string    `json:"password" bson:"password"`
	FirstName string    `json:"firstName" bson:"firstName"`
	LastName  string    `json:"lastName" bson:"lastName"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}

// RegisterRequest request
type RegisterRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// LoginRequest for logins
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResponse a typical response for login/register
type UserResponse struct {
	FirstName string `json:"firstName"`
	Token     string `json:"token"`
}

// NewUser constructor for a new User. hash password
func NewUser(rr RegisterRequest) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rr.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("register failure genp")
	}
	createdAt := time.Now()
	return &User{rr.Email, string(hashedPassword), rr.FirstName, rr.LastName, createdAt}, nil
}

// NewUserResponse constructor for UserResponse
func NewUserResponse(u User) *UserResponse {
	return &UserResponse{u.FirstName, createTokenString(u.Email)}
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

func createTokenString(email string) string {
	_, tokenString, _ := tokenAuth.Encode(jwt.MapClaims{"email": email, "exp": jwtauth.ExpireIn(15 * time.Minute)})
	return tokenString
}
