package main

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//TradeStatus defines the status of a trade
type TradeStatus int

// Status of a trade
const (
	Open TradeStatus = iota
	Executed
	Void
)

// Trade entry
type Trade struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CreatedAt       time.Time          `json:"createdAt" bson:"createdAt"`
	InitiatorEmail  string             `json:"initiatorEmail" bson:"initiatorEmail"`
	ExecutorEmail   string             `json:"executorEmail" bson:"executorEmail"`
	InitiatorTrades []uuid.UUID        `json:"initiatorTrades" bson:"initiatorTrades"`
	ExecutorTrades  []uuid.UUID        `json:"executorTrades" bson:"executorTrades"`
	Status          TradeStatus        `json:"status" bson:"status"`
}

// Ledger a log of trades assoicated with a schedule
type Ledger struct {
	ScheduleID primitive.ObjectID `json:"scheduleId" bson:"scheduleId"`
	Trades     []Trade            `json:"trades" bson:"trades"`
}

// TradeRequest for creating a new trade
type TradeRequest struct {
	ScheduleID      string   `json:"scheduleId"`
	InitiatorEmail  string   `json:"initiatorEmail"`
	ExecutorEmail   string   `json:"executorEmail"`
	InitiatorTrades []string `json:"initiatorTrades"`
	ExecutorTrades  []string `json:"executorTrades"`
}

// FinalizeTradeRequest for accepting an existing trade
type FinalizeTradeRequest struct {
	ScheduleID string `json:"scheduleId"`
	TradeID    string `json:"tradeId"`
	Action     bool   `json:"action"` // 0 decline, 1 accept
}

// NewTrade creates a new trade once passing domain validation checks
func NewTrade(tr *TradeRequest) (*Trade, error) {

	schID, err := primitive.ObjectIDFromHex(tr.ScheduleID)
	if err != nil {
		return nil, err
	}
	// check schedule exists
	sch := &MasterSchedule{}
	err = mh.GetMasterSchedule(sch, bson.M{"_id": schID})
	if err != nil {
		return nil, err
	}
	// check users exist
	initUser, execUser := &User{}, &User{}
	err = mh.GetUser(initUser, bson.M{"email": tr.InitiatorEmail})
	if err != nil {
		return nil, err
	}
	err = mh.GetUser(execUser, bson.M{"email": tr.ExecutorEmail})
	if err != nil {
		return nil, err
	}

	// users belong to group
	g := &Group{}
	err = mh.GetGroup(g, bson.M{"_id": sch.GroupID})
	if err != nil {
		return nil, err
	}
	if !g.HasUser(initUser.ID) || !g.HasUser(execUser.ID) {
		return nil, errors.New("one trade member does not belong to group")
	}

	schUnitMemberMap := make(map[string]string)
	initTrades, execTrades := []uuid.UUID{}, []uuid.UUID{}

	for _, guid := range tr.InitiatorTrades {
		initTrades = append(initTrades, uuid.MustParse(guid))
		schUnitMemberMap[guid] = tr.InitiatorEmail
	}
	for _, guid := range tr.ExecutorTrades {
		execTrades = append(execTrades, uuid.MustParse(guid))
		schUnitMemberMap[guid] = tr.ExecutorEmail
	}

	// TODO speed this up: maybe make a flat map right onto the doc?
	for _, s := range sch.Schedule.Seasons {
		for _, b := range s.Blocks {
			for _, unit := range b.Weeks {
				if owner, ok := schUnitMemberMap[unit.ID.String()]; ok {
					if unit.Participant != owner {
						return nil, errors.New(unit.ID.String() + " not owned by " + owner)
					}
				}
			}
		}
	}

	return &Trade{primitive.NilObjectID, time.Now(), tr.InitiatorEmail, tr.ExecutorEmail, initTrades, execTrades, Void}, nil
}

func FinalizeTrade() {

	// TODO: void out ALL/ANY other trades that share any traded units (uuids)
	// TODO: update this trade status.
	// TODO: reflect trade in schedule
}
