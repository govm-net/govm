package handler

import (
	"encoding/json"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/runtime"
	"log"
	"os"
	"time"
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
	ldbBlockRunStat = "block_run_stat"   //blockKey:stat
	ldbIDBlocks     = "id_blocks"        //index:blocks
	ldbMineHistory  = "mine_history"     //blockKey:count
	ldbSyncBlocks   = "sync_blocks"      //index:blockKey
	ldbTransList    = "trans_list"       //blockKey:transList
	ldbTransInfo    = "trans_info"       //transKey:info
	ldbInputTrans   = "input_trans"      //receive transfer
	ldbOutputTrans  = "output_trans"     //create by self
	ldbBlacklist    = "user_blacklist"   //blacklist of user,user:info
	ldbMiner        = "miner_register"   //chain:index
	ldbHPLimit      = "hash_power_limit" //index:limit
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
	ldb.SetCache(ldbTransInfo)
	ldb.SetCache(ldbBlacklist)
	ldb.SetCache(ldbMiner)
	ldb.SetCache(ldbHPLimit)
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
		v = v[core.HashLen:]
	}
	return out
}

type transInfo struct {
	core.TransactionHead
	Key      core.Hash
	Size     uint32
	Stat     uint32
	Selected uint32
}

func saveTransInfo(chain uint64, key []byte, info transInfo) {
	ldb.LSet(chain, ldbTransInfo, key, runtime.Encode(info))
}

func readTransInfo(chain uint64, key []byte) transInfo {
	out := transInfo{}
	v := ldb.LGet(chain, ldbTransInfo, key)
	if len(v) > 0 {
		runtime.Decode(v, &out)
	}
	return out
}

func getNextTransInfo(chain uint64, preKey []byte) transInfo {
	out := transInfo{}
	_, v := ldb.LGetNext(chain, ldbTransInfo, preKey)
	if len(v) > 0 {
		runtime.Decode(v, &out)
	}
	return out
}

func deleteTransInfo(chain uint64, key []byte) {
	ldb.LSet(chain, ldbTransInfo, key, nil)
}

// GetOutputTrans get output transaction by self
func GetOutputTrans(chain uint64, preKey []byte) []core.Hash {
	out := make([]core.Hash, 0)
	for i := 0; i < 10; i++ {
		key, _ := ldb.LGetNext(chain, ldbOutputTrans, preKey)
		if len(key) == 0 {
			break
		}
		preKey = key
		it := core.Hash{}
		runtime.Decode(key, &it)
		out = append(out, it)
	}

	return out
}

// GetInputTrans get input transaction
func GetInputTrans(chain uint64, preKey []byte) []core.Hash {
	out := make([]core.Hash, 0)
	for i := 0; i < 10; i++ {
		key, _ := ldb.LGetNext(chain, ldbInputTrans, preKey)
		if len(key) == 0 {
			break
		}
		preKey = key
		it := core.Hash{}
		runtime.Decode(key, &it)
		out = append(out, it)
	}

	return out
}

// BlackItem  the info of blacklist
type blackItem struct {
	Timeout int64
	Count   uint64
	Other   uint64
}

func saveBlackItem(chain uint64, key []byte) {
	info := blackItem{}
	v := ldb.LGet(chain, ldbBlacklist, key)
	if len(v) > 0 {
		runtime.Decode(v, &info)
	}
	info.Count++
	if info.Count%5 == 0 {
		info.Timeout = time.Now().Add(3 * time.Hour).Unix()
	}
	ldb.LSet(chain, ldbBlacklist, key, runtime.Encode(info))
}

func believable(chain uint64, key []byte) bool {
	info := blackItem{}
	v := ldb.LGet(chain, ldbBlacklist, key)
	if len(v) == 0 {
		return true
	}
	runtime.Decode(v, &info)
	if info.Timeout < time.Now().Unix() {
		return true
	}
	return false
}

func getData(chain uint64, tb string, key []byte, value interface{}) {
	v := ldb.LGet(chain, tb, key)
	if len(v) == 0 {
		return
	}
	runtime.Decode(v, &value)
}
