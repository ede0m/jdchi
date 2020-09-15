package main

// Configuration obj
type Configuration struct {
	MongoUser         string
	MongoPass         string
	MongoHost         string
	MongoPort         string
	MongoReplicaSet   string
	JWTSecret         string
	APIBaseURL        string
	APIPort           string
	APIMailerAddress  string
	APIMailerPassword string
	ClientBaseURL     string
}
