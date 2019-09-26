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

func init() {
	gob.Register(&ReqBlockInfo{})
	gob.Register(&BlockInfo{})
	gob.Register(&ReqBlock{})
	gob.Register(&BlockData{})
	gob.Register(&TransactionInfo{})
	gob.Register(&ReqTransaction{})
	gob.Register(&TransactionData{})
}
