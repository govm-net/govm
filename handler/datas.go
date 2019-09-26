package handler

import (
	"encoding/json"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/runtime"
	"log"
	"os"
)

// BlockRunStat stat of block
type BlockRunStat struct {
	RunTimes        int    `json:"run_times,omitempty"`
	RunSuccessCount int    `json:"run_success_count,omitempty"`
	RollbackCount   int    `json:"rollback_count,omitempty"`
	RollbackTime    int64  `json:"rollback_time,omitempty"`
	SelectedCount   uint64 `json:"selected_count,omitempty"`
}

const (
	ldbBlockRunStat = "block_run_stat" //blockKey:stat
	ldbIDBlocks     = "id_blocks"      //index:blocks
	ldbMineHistory  = "mine_history"   //blockKey:count
	ldbSyncBlocks   = "sync_blocks"    //index:blockKey
	ldbTransList    = "trans_list"     //blockKey:transList
)

var ldb *database.LDB

func init() {
	ldb = database.NewLDB("local.db", 10000)
	if ldb == nil {
		log.Println("fail to open ldb,local.db")
		os.Exit(2)
	}
	ldb.SetCache(ldbBlockRunStat)
	ldb.SetCache(ldbIDBlocks)
	ldb.SetCache(ldbMineHistory)
	ldb.SetCache(ldbSyncBlocks)
	ldb.SetCache(ldbTransList)
}

// SaveBlockRunStat save block stat
func SaveBlockRunStat(chain uint64, key []byte, rb BlockRunStat) {
	if chain == 0 {
		return
	}
	data, err := json.Marshal(rb)
	if err != nil {
		log.Fatal(err)
	}
	ldb.LSet(chain, ldbBlockRunStat, key, data)
}

// ReadBlockRunStat get stat of block
func ReadBlockRunStat(chain uint64, key []byte) (cl BlockRunStat) {
	if chain == 0 {
		return
	}
	stream := ldb.LGet(chain, ldbBlockRunStat, key)
	if stream != nil {
		json.Unmarshal(stream, &cl)
	}

	return
}

// ItemBlock Item of IDBlocks
type ItemBlock struct {
	Key       core.Hash
	HashPower uint64
}

// IDBlocks the blocks of same index
type IDBlocks struct {
	Items     []ItemBlock
	MaxHeight uint64
}

// SaveIDBlocks save blocks of the index
func SaveIDBlocks(chain, index uint64, ib IDBlocks) {
	if chain == 0 {
		return
	}
	key := runtime.Encode(index)
	data, err := json.Marshal(ib)
	if err != nil {
		log.Println("fail to Marshal IDBlocks.", err)
		return
	}
	ldb.LSet(chain, ldbIDBlocks, key, data)
}

// ReadIDBlocks get blocks of the index
func ReadIDBlocks(chain, index uint64) (ib IDBlocks) {
	if chain == 0 {
		return
	}
	key := runtime.Encode(index)
	stream := ldb.LGet(chain, ldbIDBlocks, key)
	if stream != nil {
		json.Unmarshal(stream, &ib)
	}

	return
}

// GetMineCount get mine count
func GetMineCount(chain uint64, key []byte) uint64 {
	var out uint64
	stream := ldb.LGet(chain, ldbMineHistory, key)
	if stream != nil {
		runtime.Decode(stream, &out)
	}
	return out
}

// SetMineCount set mine count
func SetMineCount(chain uint64, key []byte, count uint64) {
	ldb.LSet(chain, ldbMineHistory, key, runtime.Encode(count))
}

// SetSyncBlock set sync block, index:key
func SetSyncBlock(chain, index uint64, key []byte) {
	ldb.LSet(chain, ldbSyncBlocks, runtime.Encode(index), key)
}

// GetSyncBlock get sync block by index
func GetSyncBlock(chain, index uint64) []byte {
	return ldb.LGet(chain, ldbSyncBlocks, runtime.Encode(index))
}

// SaveTransList save trans list of block
func SaveTransList(chain uint64, key []byte, value []core.Hash) {
	var v []byte
	for _, it := range value {
		v = append(v, it[:]...)
	}
	ldb.LSet(chain, ldbTransList, key[:], v)
}

// GetTransList get trans list of block
func GetTransList(chain uint64, key []byte) []core.Hash {
	v := ldb.LGet(chain, ldbTransList, key[:])
	var out []core.Hash
	for len(v) > 0 {
		var tmp core.Hash
		runtime.Decode(v, &tmp)
		out = append(out, tmp)
		v = v[len(tmp[:]):]
	}
	return out
}
