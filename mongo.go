package main

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	clientOpts := options.Client().ApplyURI("mongodb://" + MongoHost + ":" + MongoPort).
		SetAuth(credential).
		SetReplicaSet(MongoReplicaSet)

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := collection.InsertOne(ctx, u)
	return result, err
}

// InsertGroupUsers creats users and adds them to group
func (mh *MongoHandler) InsertGroupUsers(users []*User, groupID primitive.ObjectID) (*mongo.InsertManyResult, error) {
	collectionGroup := mh.client.Database(mh.database).Collection("group")
	collectionUser := mh.client.Database(mh.database).Collection("user")
	var many []interface{}
	for _, u := range users {
		many = append(many, u)
	}
	var session mongo.Session
	var err error
	if session, err = mh.client.StartSession(); err != nil {
		return nil, errors.New("session error")
	}
	if err := session.StartTransaction(); err != nil {
		return nil, errors.New("tx group error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result *mongo.InsertManyResult
	if err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// create users
		result, err = collectionUser.InsertMany(ctx, many)
		if err != nil {
			return err
		}
		// add to group
		var update = bson.M{"$addToSet": bson.M{"members": bson.M{"$each": result.InsertedIDs}}}
		if _, err := collectionGroup.UpdateOne(sc, bson.M{"_id": groupID}, update); err != nil {
			return err
		}
		if err = session.CommitTransaction(sc); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	session.EndSession(ctx)
	return result, nil
}

// GetUser get a user
func (mh *MongoHandler) GetUser(u *User, filter interface{}) error {
	collection := mh.client.Database(mh.database).Collection("user")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, filter).Decode(u)
	return err
}

// GetUsers returns list of users specified by filter
func (mh *MongoHandler) GetUsers(filter interface{}) ([]*User, error) {
	collection := mh.client.Database(mh.database).Collection("user")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projection := bson.D{{"email", 1}, {"_id", 1}} // set field to 1 to project
	cur, err := collection.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var result []*User
	for cur.Next(ctx) {
		u := &User{}
		er := cur.Decode(u)
		if er != nil {
			log.Fatal(er)
		}
		result = append(result, u)
	}
	return result, nil
}

// GetGroup get a user
func (mh *MongoHandler) GetGroup(g *Group, filter interface{}) error {
	collection := mh.client.Database(mh.database).Collection("group")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, filter).Decode(g)
	return err
}

// InsertGroup create new group and adds group to all admin user's groups in transaction
func (mh *MongoHandler) InsertGroup(g *Group) (primitive.ObjectID, error) {
	collectionGroup := mh.client.Database(mh.database).Collection("group")
	collectionUser := mh.client.Database(mh.database).Collection("user")

	var session mongo.Session
	var err error
	if session, err = mh.client.StartSession(); err != nil {
		return primitive.NilObjectID, errors.New("session error")
	}
	if err := session.StartTransaction(); err != nil {
		return primitive.NilObjectID, errors.New("tx group error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	groupID := primitive.NilObjectID
	if err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// create the group
		result, err := collectionGroup.InsertOne(ctx, g)
		if err != nil {
			return err
		}
		// add group to all members
		groupID = result.InsertedID.(primitive.ObjectID)
		var update = bson.D{{Key: "$addToSet", Value: bson.D{{Key: "groups", Value: groupID}}}}
		for _, mID := range g.Members {
			if _, err := collectionUser.UpdateOne(sc, bson.M{"_id": mID}, update); err != nil {
				return err
			}
		}
		if err = session.CommitTransaction(sc); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return primitive.NilObjectID, err
	}
	session.EndSession(ctx)
	return groupID, nil
}
