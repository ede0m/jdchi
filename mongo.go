package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/tkanos/gonfig"
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

	// config
	configuration := Configuration{}
	err := gonfig.GetConf(getConfigFileName(), &configuration)
	if err != nil {
		panic(err)
	}

	credential := options.Credential{
		Username: configuration.MongoUser,
		Password: configuration.MongoPass,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI("mongodb://" + configuration.MongoHost + ":" + configuration.MongoPort).
		SetAuth(credential).
		SetReplicaSet(configuration.MongoReplicaSet)

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
	collectionSch := mh.client.Database(mh.database).Collection("schedule")

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
	var result *mongo.InsertOneResult
	if err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// create master schedule
		result, err = collectionSch.InsertOne(sc, ms)
		if err != nil {
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
	opts.SetSort(bson.M{"createdAt": -1})
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

// UpdateUsers updates many with filter and condition
func (mh *MongoHandler) UpdateUsers(filter interface{}, update interface{}) error {
	collection := mh.client.Database(mh.database).Collection("user")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := collection.UpdateMany(ctx, filter, update)
	return err
}

// GetUsers returns list of users specified by filter
func (mh *MongoHandler) GetUsers(filter interface{}) ([]*User, error) {
	collection := mh.client.Database(mh.database).Collection("user")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	projection := bson.M{"email": 1, "_id": 1, "firstName": 1, "lastName": 1} // set field to 1 to project
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

// InsertGroup create new users if needed, creates a group with all members, adds groups to each member,
//  then creates the group schedule in transaction
func (mh *MongoHandler) InsertGroup(g *Group, sch *MasterSchedule, newUsers []*User, existingUsers []*User) (primitive.ObjectID, error) {
	collectionGroup := mh.client.Database(mh.database).Collection("group")
	collectionUser := mh.client.Database(mh.database).Collection("user")
	collectionSchedule := mh.client.Database(mh.database).Collection("schedule")

	var manyNew []interface{}
	for _, u := range newUsers {
		manyNew = append(manyNew, u)
	}
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

	var users []primitive.ObjectID
	groupID := primitive.NilObjectID
	if err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {

		// create new users
		if manyNew != nil {
			usersNew, err := collectionUser.InsertMany(ctx, manyNew)
			if err != nil {
				return err
			}
			// merge new users with existing
			for _, nID := range usersNew.InsertedIDs {
				nOID := nID.(primitive.ObjectID)
				users = append(users, nOID)
			}
		}

		for _, u := range existingUsers {
			users = append(users, u.ID)
		}

		// create the group
		group, err := collectionGroup.InsertOne(sc, g)
		if err != nil {
			return err
		}

		// add group to members
		groupID = group.InsertedID.(primitive.ObjectID)
		var update = bson.M{"$addToSet": bson.M{"groups": groupID}}
		for _, mID := range users {
			if _, err := collectionUser.UpdateOne(sc, bson.M{"_id": mID}, update); err != nil {
				return err
			}
		}

		// add members to group
		update = bson.M{"$addToSet": bson.M{"members": bson.M{"$each": users}}}
		if _, err := collectionGroup.UpdateOne(sc, bson.M{"_id": groupID}, update); err != nil {
			return err
		}

		// create the schedule
		sch.GroupID = groupID
		if _, err := collectionSchedule.InsertOne(sc, sch); err != nil {
			return err
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

// InsertTrade inserts one master schedule into ledger colletion
func (mh *MongoHandler) InsertTrade(t *Trade, schID primitive.ObjectID) error {
	collection := mh.client.Database(mh.database).Collection("schedule")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	update := bson.M{"$addToSet": bson.M{"tradeLedger": t}}
	if _, err := collection.UpdateOne(ctx, bson.M{"_id": schID}, update); err != nil {
		return err
	}
	return nil
}

// GetActiveScheduleUserTrades returns a user's trades for all active user groups in groupIDs
func (mh *MongoHandler) GetActiveScheduleUserTrades(groupIDs []primitive.ObjectID, email string) []GroupTrades {

	// TODO: use $lookup with userID to remove getUser lookup

	collection := mh.client.Database(mh.database).Collection("schedule")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	match := bson.M{"$match": bson.M{"groupId": bson.M{"$in": groupIDs}}}
	sort := bson.M{"$sort": bson.M{"groupId": -1, "createdAt": -1}}
	group := bson.M{"$group": bson.M{
		"_id":    "$groupId",
		"schId":  bson.M{"$first": "$_id"},
		"trades": bson.M{"$first": "$tradeLedger"},
	}}
	project := bson.M{"$project": bson.M{
		"_id":     "$schId",
		"groupId": "$_id",
		"trades": bson.M{"$filter": bson.M{
			"input": "$trades",
			"as":    "trade",
			"cond": bson.M{"$or": bson.A{
				bson.M{"$eq": bson.A{"$$trade.initiatorEmail", email}},
				bson.M{"$eq": bson.A{"$$trade.executorEmail", email}},
			}},
		}},
	}}
	pipeline := []bson.M{match, sort, group, project}
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		panic(err)
	}
	var groupsTrades []GroupTrades
	for cursor.Next(ctx) {
		gt := &GroupTrades{}
		cursor.Decode(gt)
		groupsTrades = append(groupsTrades, *gt)
	}
	if err := cursor.Close(ctx); err != nil {
		panic(err)
	}
	return groupsTrades
}

// GetTrade get's a trade by ID within the schedule tradeLedger using mongo aggregate framework
func (mh *MongoHandler) GetTrade(t *Trade, tradeID, schID primitive.ObjectID) error {
	collection := mh.client.Database(mh.database).Collection("schedule")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	matchSch := bson.M{"$match": bson.M{"_id": schID}}
	unwind := bson.M{"$unwind": "$tradeLedger"}
	matchT := bson.M{"$match": bson.M{"tradeLedger._id": tradeID}}
	project := bson.M{"$project": bson.M{
		"_id":             "$tradeLedger._id",
		"createdAt":       "$tradeLedger.createdAt",
		"initiatorEmail":  "$tradeLedger.initiatorEmail",
		"executorEmail":   "$tradeLedger.executorEmail",
		"initiatorTrades": "$tradeLedger.initiatorTrades",
		"executorTrades":  "$tradeLedger.executorTrades",
		"status":          "$tradeLedger.status",
	}}
	pipeline := []bson.M{matchSch, unwind, matchT, project}
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	// the aggregate will only return one trade sub doc because of match on id
	cursor.Next(ctx)
	err = cursor.Decode(t) // do i need to call Next() before this?
	return err
}

// UpdateTrade updates a subdoc trade in a schedule with a status
func (mh *MongoHandler) UpdateTrade(filter interface{}, update interface{}) error {
	collection := mh.client.Database(mh.database).Collection("schedule")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// ExecuteTrade will execute a trade, void competeing trades and reflect it in the schedule
func (mh *MongoHandler) ExecuteTrade(t *Trade, sch *MasterSchedule) error {
	collection := mh.client.Database(mh.database).Collection("schedule")

	var unitIDs []uuid.UUID
	for _, tu := range t.InitiatorTrades {
		unitIDs = append(unitIDs, tu.ID)
	}
	for _, tu := range t.ExecutorTrades {
		unitIDs = append(unitIDs, tu.ID)
	}

	var session mongo.Session
	var err error
	if session, err = mh.client.StartSession(); err != nil {
		return errors.New("session error")
	}
	if err := session.StartTransaction(); err != nil {
		return errors.New("tx group error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {

		// void out ALL/ANY other trades that share any traded units (uuids)
		filter := bson.M{"_id": sch.ID}
		update := bson.M{"$set": bson.M{"tradeLedger.$[trade].status": 2}} // 2 is void

		arrayFiltersOpts := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{bson.M{
				"$or": bson.A{
					bson.M{"trade.initiatorTrades": bson.M{"$in": unitIDs}},
					bson.M{"trade.executorTrades": bson.M{"$in": unitIDs}},
				},
			}},
		})
		if _, err := collection.UpdateOne(sc, filter, update, arrayFiltersOpts); err != nil {
			return err
		}
		// update this trade status executed
		// reflect trade in schedule
		filter = bson.M{"_id": sch.ID, "tradeLedger._id": t.ID}
		update = bson.M{"$set": bson.M{
			"tradeLedger.$.status": 1, // 1 is executed
			"schedule":             sch.Schedule,
			"scheduleUnitMap":      sch.ScheduleUnitMap,
		}}
		if _, err := collection.UpdateOne(sc, filter, update); err != nil {
			return err
		}
		if err = session.CommitTransaction(sc); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	session.EndSession(ctx)
	return nil
}
