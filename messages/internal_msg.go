package messages

// BaseMsg base message
type BaseMsg struct {
	Type string
	Msg  interface{}
}

func (m *BaseMsg) String() string {
	return ""
}

// GetType get message type
func (m *BaseMsg) GetType() string {
	return m.Type
}

// GetMsg get message
func (m *BaseMsg) GetMsg() interface{} {
	return m.Msg
}

// NewTransaction internal message
type NewTransaction struct {
	BaseMsg
	Chain uint64
	Key   []byte
	Data  []byte
}

// Mine Mine
type Mine struct {
	BaseMsg
	Chain uint64
	Index uint64
}

// Rollback Rollback
// type Rollback struct {
// 	BaseMsg
// 	Chain uint64
// 	Index uint64
// 	Key   []byte
// }

// ChainEvent ChainEvent
type ChainEvent struct {
	BaseMsg
	Chain uint64
	Who   string
	Event string
	Param string
}

// NewNode new node
type NewNode struct {
	BaseMsg
	Peer string
}

// RawData raw data
type RawData struct {
	BaseMsg
	Chain     uint64
	Key       []byte
	Broadcast bool
	IsTrans   bool
	Data      []byte
}

// ReceiveTrans receive new transaction
type ReceiveTrans struct {
	BaseMsg
	Chain uint64
	Key   []byte
}

// Internal msg keys
const (
	BroadcastMsg = "broadcast"
	RandsendMsg  = "randsend"
)
