package handler

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/runtime"
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
	ldbSyncBlocks   = "sync_blocks"      //index:blockKey
	ldbTransList    = "trans_list"       //blockKey:transList
	ldbTransInfo    = "trans_info"       //transKey:info
	ldbAllTransInfo = "all_trans_info"   //transKey:info
	ldbNewerTrans   = "newer_trans"      //timeKey:transKey
	ldbInputTrans   = "input_trans"      //receive transfer,timeKey:transKey
	ldbOutputTrans  = "output_trans"     //create by self,timeKey:transKey
	ldbBlacklist    = "user_blacklist"   //blacklist of user,user:info
	ldbMiner        = "miner_register"   //chain:index
	ldbHPLimit      = "hash_power_limit" //index:limit
	ldbBlockLocked  = "block_locked"     //key:n
	ldbDownloading  = "downloading"      //key:time
)

const downloadTimeout = 10

var ldb *database.LDB

func init() {
	ldb = database.NewLDB("local.db", 10000)
	if ldb == nil {
		log.Println("fail to open ldb,local.db")
		os.Exit(2)
	}
	ldb.SetNotDisk(ldbBlockRunStat, 10000)
	ldb.SetNotDisk(ldbIDBlocks, 10000)
	ldb.SetNotDisk(ldbSyncBlocks, 10000)
	ldb.SetCache(ldbTransList)
	ldb.SetCache(ldbTransInfo)
	ldb.SetCache(ldbBlacklist)
	ldb.SetCache(ldbMiner)
	ldb.SetNotDisk(ldbHPLimit, 1000)
	ldb.SetNotDisk(ldbBlockLocked, 10000)
	ldb.SetNotDisk(ldbDownloading, 2000)
}

// Exit os exit
func Exit() {
	ldb.Close()
	core.Exit()
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
	value := runtime.Encode(info)
	if conf.GetConf().DoMine {
		ldb.LSet(chain, ldbTransInfo, key, value)
	}
	ldb.LSet(chain, ldbAllTransInfo, key, value)
}

func readTransInfo(chain uint64, key []byte) transInfo {
	out := transInfo{}
	v := ldb.LGet(chain, ldbAllTransInfo, key)
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
		k, key := ldb.LGetNext(chain, ldbOutputTrans, preKey)
		if len(key) == 0 {
			break
		}
		preKey = k
		if len(key) == 1 {
			key = k
			ldb.LSet(chain, ldbOutputTrans, key, nil)
			var t uint64 = ^uint64(0)
			k = runtime.Encode(t)
			k = append(k, key[:16]...)
			ldb.LSet(chain, ldbOutputTrans, k, key)
		}
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
		k, key := ldb.LGetNext(chain, ldbInputTrans, preKey)
		if len(key) == 0 {
			break
		}
		preKey = k
		if len(key) == 1 {
			key = k
			ldb.LSet(chain, ldbInputTrans, key, nil)
			var t uint64 = ^uint64(0)
			k = runtime.Encode(t)
			k = append(k, key[:16]...)
			ldb.LSet(chain, ldbInputTrans, k, key)
		}
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
	info.Timeout = time.Now().Add(3 * time.Hour).Unix()
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
	runtime.Decode(v, value)
}

func getBlockLockNum(chain uint64, key []byte) uint64 {
	var out uint64
	rst := ldb.LGet(chain, ldbBlockLocked, key)
	if len(rst) >= 8 {
		runtime.Decode(rst, &out)
	}
	return out
}

func setBlockLockNum(chain uint64, key []byte, val uint64) {
	var old uint64
	rst := ldb.LGet(chain, ldbBlockLocked, key)
	if len(rst) >= 8 {
		runtime.Decode(rst, &old)
	}
	if val > old {
		ldb.LSet(chain, ldbBlockLocked, key, runtime.Encode(val))
	}
}

// get the time of download, if fresh, return false
func needDownload(chain uint64, key []byte) bool {
	var old int64
	rst := ldb.LGet(chain, ldbDownloading, key)
	if len(rst) >= 8 {
		runtime.Decode(rst, &old)
	}
	now := time.Now().Unix()
	if old+downloadTimeout < now {
		log.Printf("need download,chain:%d,key:%x\n", chain, key)
		ldb.LSet(chain, ldbDownloading, key, runtime.Encode(now))
		return true
	}
	log.Printf("not need download,chain:%d,key:%x\n", chain, key)
	return false
}

// get the time of request, if fresh, return false
func needRequstID(chain, index uint64) bool {
	return needDownload(chain, runtime.Encode(index))
}
