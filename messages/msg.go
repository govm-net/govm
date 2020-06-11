package messages

import (
	"encoding/gob"
)

// ReqBlockInfo request block info
type ReqBlockInfo struct {
	Chain uint64
	Index uint64
}

// BlockInfo response block info
type BlockInfo struct {
	Chain     uint64
	Index     uint64
	HashPower uint64
	Key       []byte
	User      []byte
	PreKey    []byte
}

// ReqBlock request block data
type ReqBlock struct {
	Chain uint64
	Index uint64
	Key   []byte
}

// BlockData response block data
type BlockData struct {
	Chain uint64
	Key   []byte
	Data  []byte
}

// TransactionInfo transaction info
type TransactionInfo struct {
	Chain uint64
	Time  uint64
	Key   []byte
	User  []byte
}

// ReqTransList request transaction list
type ReqTransList struct {
	Chain uint64
	Key   []byte
}

// TransactionList response transaction list
type TransactionList struct {
	Chain uint64
	Key   []byte
	Data  []byte
}

// ReqTransaction request transaction data
type ReqTransaction struct {
	Chain uint64
	Key   []byte
}

// TransactionData response transaction data
type TransactionData struct {
	Chain uint64
	Key   []byte
	Data  []byte
}

// NodeInfo node info
type NodeInfo struct {
	Alias           string `json:"alias,omitempty"`
	Version         string `json:"version,omitempty"`
	RunTime         int64  `json:"run_time,omitempty"`
	NodesConnected  int    `json:"nodes_connected,omitempty"`
	MinersConnected int    `json:"miners_connected,omitempty"`
	Height          uint64 `json:"height,omitempty"`
}

func init() {
	gob.Register(&ReqBlockInfo{})
	gob.Register(&BlockInfo{})
	gob.Register(&ReqBlock{})
	gob.Register(&BlockData{})
	gob.Register(&ReqTransList{})
	gob.Register(&TransactionList{})
	gob.Register(&TransactionInfo{})
	gob.Register(&ReqTransaction{})
	gob.Register(&TransactionData{})
	gob.Register(&NodeInfo{})
}
