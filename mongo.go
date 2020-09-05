package main

import (
	"context"
	"time"

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

// GetOne scheudle doc by filter, options
func (mh *MongoHandler) GetOne(ms *MasterSchedule, filter interface{}, opts *options.FindOneOptions) error {
	//Will automatically create a collection if not available
	collection := mh.client.Database(mh.database).Collection("schedule")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error = nil
	if opts != nil {
		err = collection.FindOne(ctx, filter, opts).Decode(ms)
	} else {
		err = collection.FindOne(ctx, filter, opts).Decode(ms)
	}
	return err
}
