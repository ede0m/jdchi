package main

import (
	"time"

	"github.com/google/uuid"
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
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	InitiatorEmail string             `json:"initiatorEmail" bson:"initiatorEmail"`
	ExecutorEmail  string             `json:"executorEmail" bson:"executorEmail"`
	InitatorTrades []uuid.UUID        `json:"initatorTrades" bson:"initatorTrades"`
	ExecutorTrades []uuid.UUID        `json:"executorTrades" bson:"executorTrades"`
	Status         TradeStatus        `json:"status" bson:"status"`
}

// Ledger a log of trades assoicated with a schedule
type Ledger struct {
	ScheduleID primitive.ObjectID `json:"scheduleId" bson:"scheduleId"`
	Trades     []Trade            `json:"trades" bson:"trades"`
}
