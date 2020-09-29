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

func NewTrade(tr *TradeRequest) *Trade {
	// TODO check schedule exists
	// TODO check participants exist and that they belong to schedule
	// TODO check all initiator trades belong to initiator
	// TODO check all executor trades belong to executor
	return nil
}

func FinalizeTrade() {

	// TODO: void out ALL/ANY other trades that share any traded units (uuids)
	// TODO: update this trade status.
	// TODO: reflect trade in schedule
}
