package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
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
	Cancelled
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
	Action     int    `json:"action"` // 0 decline/cancel, 1 accept
}

// TradeResponse client response for a created trade
type TradeResponse struct {
	Trade Trade `json:"trade"`
}

// UserTradesResponse shows a user's trades across groups
type UserTradesResponse struct {
	UserGroupTrades map[string]GroupTrades `json:"userGroupTrades"`
}

// GroupTrades monog aggregate query for user's group trades
type GroupTrades struct {
	ScheduleID primitive.ObjectID `json:"scheduleId" bson:"_id"`
	GroupID    primitive.ObjectID `json:"groupId" bson:"groupId"`
	Trades     []Trade            `json:"trades" bson:"trades"`
}

// Bind binds the http req to groupRequest type as the render
func (tr *TradeRequest) Bind(r *http.Request) error {

	if tr.ScheduleID == "" {
		return errors.New("missing scheduleID")
	} else if tr.InitiatorEmail == "" {
		return errors.New("missing initiator email")
	} else if tr.ExecutorEmail == "" {
		return errors.New("missing executor email")
	} else if len(tr.InitiatorTrades) == 0 {
		return errors.New("must have at least one trade away")
	} else if len(tr.ExecutorTrades) == 0 {
		return errors.New("must have at least one trade for")
	}
	return nil
}

// Bind binds the http req to groupRequest type as the render
func (ftr *FinalizeTradeRequest) Bind(r *http.Request) error {

	if ftr.ScheduleID == "" {
		return errors.New("missing scheduleID")
	} else if ftr.TradeID == "" {
		return errors.New("missing tradeID")
	} else if ftr.Action != 0 && ftr.Action != 1 {
		return errors.New("action should be 0 (decline/cancel) or 1 (accept)")
	}
	return nil
}

// Render is called in top-down order, like a http handler middleware chain.
func (tr *TradeResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Render is called in top-down order, like a http handler middleware chain.
func (tr *UserTradesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// NewTradeResponse returns a client response for a trade
func NewTradeResponse(t Trade) *TradeResponse {
	return &TradeResponse{t}
}

// NewUserTradesResponse creates a client response obj for a user's trade
func NewUserTradesResponse(gt []GroupTrades) *UserTradesResponse {

	userGroupTrades := make(map[string]GroupTrades)
	// TODO: parse to map
	for _, g := range gt {
		gid := g.GroupID.Hex()
		if _, found := userGroupTrades[gid]; !found {
			userGroupTrades[gid] = g
		}
	}
	return &UserTradesResponse{userGroupTrades}
}

// NewTrade creates a new trade once passing domain validation checks
func NewTrade(tr *TradeRequest, reqUserID string) (*Trade, error) {

	schID, err := primitive.ObjectIDFromHex(tr.ScheduleID)
	if err != nil {
		return nil, err
	}
	reqUID, err := primitive.ObjectIDFromHex(reqUserID)
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

	// check initiator is requestor
	if initUser.ID != reqUID {
		return nil, errors.New("trade must be made by initiator")
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

	// check that initiator trades belong to initiator
	initTrades, execTrades := []uuid.UUID{}, []uuid.UUID{}
	for _, guid := range tr.InitiatorTrades {
		initTrades = append(initTrades, uuid.MustParse(guid))
		if v, ok := sch.ScheduleUnitMap[guid]; ok {
			if v.Owner != tr.InitiatorEmail {
				return nil, errors.New(guid + " not owned by " + tr.InitiatorEmail)
			}
		}
	}
	// check that executor trades belong to executor
	for _, guid := range tr.ExecutorTrades {
		execTrades = append(execTrades, uuid.MustParse(guid))
		if v, ok := sch.ScheduleUnitMap[guid]; ok {
			if v.Owner != tr.ExecutorEmail {
				return nil, errors.New(guid + " not owned by " + tr.ExecutorEmail)
			}
		}
	}

	return &Trade{primitive.NewObjectID(), time.Now(), tr.InitiatorEmail, tr.ExecutorEmail, initTrades, execTrades, Open}, nil
}

// CreateTrade creates a new trade
func CreateTrade(w http.ResponseWriter, r *http.Request) {
	data := &TradeRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// request auth is from initiator
	_, claims, _ := jwtauth.FromContext(r.Context())
	trade, err := NewTrade(data, claims["userID"].(string))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	schID, err := primitive.ObjectIDFromHex(data.ScheduleID)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	if err = mh.InsertTrade(trade, schID); err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewTradeResponse(*trade))
}

/*FinalizeTrade will update a trade status based on execution or decline.
It can update competeing trades, and will reflect the trade in the assoicated schedule */
func FinalizeTrade(w http.ResponseWriter, r *http.Request) {
	data := &FinalizeTradeRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	_, claims, _ := jwtauth.FromContext(r.Context())
	uid, _ := primitive.ObjectIDFromHex(claims["userID"].(string))
	tid, _ := primitive.ObjectIDFromHex(data.TradeID)
	schid, _ := primitive.ObjectIDFromHex(data.ScheduleID)
	u := &User{}
	if err := mh.GetUser(u, bson.M{"_id": uid}); err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	t := &Trade{}
	if err := mh.GetTrade(t, tid, schid); err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	if t.Status == 2 || t.Status == 3 {
		render.Render(w, r, ErrInvalidRequest(errors.New("trade is void or cancelled")))
		return
	}

	// can the requestor participate in the trade?
	tradeFilter := bson.M{"_id": schid, "tradeLedger._id": tid}
	if t.ExecutorEmail == u.Email {
		// Executor
		if data.Action == 1 {
			// Accepted:
			// TODO: incorperate lock??
			sch := &MasterSchedule{}
			if err := mh.GetMasterSchedule(sch, bson.M{"_id": schid}); err != nil {
				render.Render(w, r, ErrNotFound(err))
				return
			}
			sch.Schedule, sch.ScheduleUnitMap = sch.tradeScheduleUnits(*t)
			if err := mh.ExecuteTrade(t, sch); err != nil {
				render.Render(w, r, ErrServer(err))
				return
			}
		} else {
			// Declined!
			if err := mh.UpdateTrade(tradeFilter, bson.M{"$set": bson.M{"status": 2}}); err != nil {
				render.Render(w, r, ErrNotFound(err))
				return
			}
		}
	} else if t.InitiatorEmail == u.Email {
		if data.Action == 1 {
			render.Render(w, r, ErrInvalidRequest(errors.New("initiator cannot preform this action")))
			return
		}
		// Cancelled!
		if err := mh.UpdateTrade(tradeFilter, bson.M{"$set": bson.M{"status": 3}}); err != nil {
			render.Render(w, r, ErrNotFound(err))
			return
		}
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("requestor not involved in trade")))
		return
	}
}

// GetUserTrades gets all trades belonging to a user's current groups
func GetUserTrades(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "userID")
	uID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return
	}
	// TODO: remove this query and combine with $lookup in GetActiveScheduleUserTrades
	user := &User{}
	if err := mh.GetUser(user, bson.M{"_id": uID}); err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}
	userGroupsTrades := mh.GetActiveScheduleUserTrades(user.Groups, user.Email)
	render.Status(r, http.StatusOK)
	render.Render(w, r, NewUserTradesResponse(userGroupsTrades))
}
