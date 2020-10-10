package main

import (
	"context"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	jdchaimailer "github.com/ede0m/jdchai/mailer"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
	"github.com/tkanos/gonfig"
)

// mongo client
var mh = NewMongoHandler()
var tokenAuth *jwtauth.JWTAuth
var smtpAuth smtp.Auth
var host string
var port string
var clientBaseURL string

func main() {

	// config
	configuration := Configuration{}
	err := gonfig.GetConf(getConfigFileName(), &configuration)
	if err != nil {
		panic(err)
	}

	host = configuration.APIBaseURL
	port = configuration.APIPort
	clientBaseURL = configuration.ClientBaseURL

	// jwt setup
	tokenAuth = jwtauth.New("HS256", []byte(configuration.JWTSecret), nil)

	// mailer setup
	smtpAuth = smtp.PlainAuth("", configuration.APIMailerAddress, configuration.APIMailerPassword, "smtp.gmail.com")
	jdchaimailer.Init(smtpAuth, configuration.APIMailerAddress)

	defer mh.client.Disconnect(context.Background())
	r := chi.NewRouter()

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins:   []string{"*"},
		AllowOriginFunc:  AllowOriginFunc,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Timeout"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Use(middleware.Logger)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Route("/schedule", func(r chi.Router) {
		r.Post("/", GenerateSchedule)
		// Protected routes
		r.Group(func(r chi.Router) {
			// Seek, verify and validate JWT tokens
			r.Use(jwtauth.Verifier(tokenAuth))
			// Handle valid / invalid tokens.
			r.Use(jwtauth.Authenticator)

			r.Post("/master", CreateMasterSchedule)
			r.Get("/master/{groupID}", GetMasterSchedule)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)
		r.Route("/group", func(r chi.Router) {
			r.Post("/", CreateGroup)
			r.Post("/invitation", CreateInvites)
			r.Get("/{groupID}/user", GetGroupUsers)
		})
		r.Route("/user", func(r chi.Router) {
			r.Patch("/invitation", AcceptRegisterInvite)
			r.Get("/{userID}/trade", GetUserTrades)
		})
		r.Route("/trade", func(r chi.Router) {
			r.Post("/", CreateTrade)
			r.Patch("/", FinalizeTrade)
		})
	})

	r.Route("/session", func(r chi.Router) {
		r.Post("/", LoginUser)
	})

	http.ListenAndServe(host+":"+port, r)
}

// AllowOriginFunc logic for cors
func AllowOriginFunc(r *http.Request, origin string) bool {
	return true
}

func getConfigFileName() string {
	env := os.Getenv("ENV")
	if len(env) == 0 {
		env = "development"
	}
	filename := []string{"config/", "config.", env, ".json"}
	_, dirname, _, _ := runtime.Caller(0)
	filePath := path.Join(filepath.Dir(dirname), strings.Join(filename, ""))

	return filePath
}
