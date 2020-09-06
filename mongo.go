package main

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const DefaultDatabase = "jdscheduler"

// MongoHandler handles mongo client instance
type MongoHandler struct {
	client   *mongo.Client
	database string
}

//NewMongoHandler Constructor for MongoHandler
func NewMongoHandler() *MongoHandler {

	credential := options.Credential{
		Username: MongoUser,
		Password: MongoPass,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI("mongodb://" + MongoHost + ":" + MongoPort).SetAuth(credential)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		panic(err)
	}

	mh := &MongoHandler{
		client:   client,
		database: DefaultDatabase,
	}
	return mh
}

// scheudle handlers //

// InsertMasterSchedule inserts one master schedule into scheudle colletion
func (mh *MongoHandler) InsertMasterSchedule(ms *MasterSchedule) (*mongo.InsertOneResult, error) {
	collection := mh.client.Database(mh.database).Collection("schedule")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := collection.InsertOne(ctx, ms)
	return result, err
}

// GetMasterSchedule scheudle doc by filter, options
func (mh *MongoHandler) GetMasterSchedule(ms *MasterSchedule, filter interface{}) error {

	collection := mh.client.Database(mh.database).Collection("schedule")

	/*
		It is better to explain it using 2 parameters, e.g. search for a = 1 and b = 2.
		Your syntax will be: bson.M{"a": 1, "b": 2} or bson.D{{"a": 1}, {"b": 2}}

		bson.D respects order
	*/
	opts := options.FindOne()
	opts.SetSort(bson.D{{"createdAt", -1}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, filter, opts).Decode(ms)
	return err
}

// InsertUser inserts one master schedule into scheudle colletion
func (mh *MongoHandler) InsertUser(u *User) (*mongo.InsertOneResult, error) {
	collection := mh.client.Database(mh.database).Collection("user")

	// check email doesn't already exist
	ctxf, cancelf := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelf()
	findu := &User{}
	err := collection.FindOne(ctxf, bson.M{"email": u.Email}).Decode(findu)
	//If the filter does not match any documents, a SingleResult with an error set to ErrNoDocuments will be returned.
	if err == nil {
		// user exists
		return nil, errors.New("email already registered")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := collection.InsertOne(ctx, u)
	return result, err
}
