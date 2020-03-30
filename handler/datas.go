package handler

import (
	"encoding/hex"
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
	ldbBlockRunStat = "block_run_stat" //blockKey:stat
	ldbIDBlocks     = "id_blocks"      //index:blocks
	ldbSyncBlocks   = "sync_blocks"    //index:blockKey
	ldbTransList    = "trans_list"     //blockKey:transList
	ldbTransInfo    = "trans_info"     //transKey:info
	ldbAllTransInfo = "all_trans_info" //transKey:info
	ldbInputTrans   = "input_trans"    //receive transfer,timeKey:transKey
	ldbOutputTrans  = "output_trans"   //create by self,timeKey:transKey
	ldbBlacklist    = "user_blacklist" //blacklist of user,user:info
	ldbMiner        = "miner_register" //chain:index
	ldbBlockLocked  = "block_locked"   //key:n
	ldbDownloading  = "downloading"    //key:time
	ldbReliability  = "reliability"    //blockKey:relia
)

const downloadTimeout = 5

var ldb *database.LDB

// Init init
func Init() {
	ldb = database.NewLDB("local.db", 10000)
	if ldb == nil {
		log.Println("fail to open ldb,local.db")
		os.Exit(2)
	}
	ldb.SetNotDisk(ldbBlockRunStat, 10000)
	ldb.SetNotDisk(ldbIDBlocks, 10000)
	ldb.SetNotDisk(ldbSyncBlocks, 10000)
	ldb.SetNotDisk(ldbBlacklist, 10000)
	ldb.SetNotDisk(ldbTransList, 1000)
	ldb.SetNotDisk(ldbTransInfo, 10000)
	ldb.SetNotDisk(ldbMiner, 1000)
	ldb.SetNotDisk(ldbBlockLocked, 10000)
	ldb.SetNotDisk(ldbDownloading, 2000)
	ldb.SetNotDisk(ldbReliability, 50000)
	ldb.SetNotDisk(ldbAllTransInfo, 50000)
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
	Items []ItemBlock
	// MaxHeight uint64
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

// HistoryItem history item
type HistoryItem struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// GetOutputTrans get output transaction by self
func GetOutputTrans(chain uint64, preKey []byte) []HistoryItem {
	out := make([]HistoryItem, 0)
	for i := 0; i < 10; i++ {
		k, v := ldb.LGetNext(chain, ldbOutputTrans, preKey)
		if len(v) == 0 {
			break
		}
		preKey = k
		it := HistoryItem{}
		it.Key = hex.EncodeToString(k)
		it.Value = hex.EncodeToString(v)
		out = append(out, it)
	}

	return out
}

// GetInputTrans get input transaction
func GetInputTrans(chain uint64, preKey []byte) []HistoryItem {
	out := make([]HistoryItem, 0)
	for i := 0; i < 10; i++ {
		k, v := ldb.LGetNext(chain, ldbInputTrans, preKey)
		if len(v) == 0 {
			break
		}
		preKey = k
		it := HistoryItem{}
		it.Key = hex.EncodeToString(k)
		it.Value = hex.EncodeToString(v)
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
	type record struct {
		Time [3]int64
	}
	var info record
	rst := ldb.LGet(chain, ldbDownloading, key)
	if len(rst) > 0 {
		json.Unmarshal(rst, &info)
	}
	now := time.Now().Unix()
	for i := range info.Time {
		if info.Time[i]+downloadTimeout < now {
			info.Time[i] = now
			data, _ := json.Marshal(info)
			ldb.LSet(chain, ldbDownloading, key, data)
			return true
		}
	}
	return false
}

// get the time of request, if fresh, return false
func needRequstID(chain, index uint64) bool {
	return needDownload(chain, runtime.Encode(index))
}

// TReliability Reliability of block
type TReliability struct {
	Key        core.Hash    `json:"key,omitempty"`
	Previous   core.Hash    `json:"previous,omitempty"`
	Parent     core.Hash    `json:"parent,omitempty"`
	LeftChild  core.Hash    `json:"left_child,omitempty"`
	RightChild core.Hash    `json:"right_child,omitempty"`
	Producer   core.Address `json:"producer,omitempty"`
	Time       uint64       `json:"time,omitempty"`
	Index      uint64       `json:"index,omitempty"`
	HashPower  uint64       `json:"hash_power,omitempty"`
	Miner      bool         `json:"miner,omitempty"`
	Ready      bool         `json:"ready,omitempty"`
}

// SaveBlockReliability save block reliability
func SaveBlockReliability(chain uint64, key []byte, rb TReliability) {
	if chain == 0 {
		return
	}
	if rb.Index == 0 {
		ldb.LSet(chain, ldbReliability, key, nil)
	} else {
		ldb.LSet(chain, ldbReliability, key, runtime.Encode(rb))
	}
}

// ReadBlockReliability get Reliability of block from db
func ReadBlockReliability(chain uint64, key []byte) TReliability {
	var out TReliability
	if chain == 0 {
		return out
	}
	stream := ldb.LGet(chain, ldbReliability, key)
	if stream != nil {
		runtime.Decode(stream, &out)
	}
	return out
}

// Cmp compares x and y and returns:
//
//   +1 if x >  y
//   -1 if x <  y
//   0  if x =  y
func (r TReliability) Cmp(y TReliability) int {
	if r.HashPower > y.HashPower {
		return 1
	}
	if r.HashPower < y.HashPower {
		return -1
	}
	for i, b := range r.Key {
		if b > y.Key[i] {
			return -1
		}
		if b < y.Key[i] {
			return 1
		}
	}

	return 0
}

// Recalculation recalculation
func (r *TReliability) Recalculation(chain uint64) {
	var power uint64
	var miner core.Miner
	var parent, preRel TReliability

	if r.Index > 1 {
		preRel = ReadBlockReliability(chain, r.Previous[:])
		if chain > 1 {
			parent = ReadBlockReliability(chain/2, r.Parent[:])
		}
	}

	miner = core.GetMinerInfo(chain, r.Index)

	if r.Index == 1 {
		power = 1000
	}

	hp := getHashPower(r.Key[:])
	for i := 0; i < core.MinerNum; i++ {
		if miner.Miner[i] == r.Producer {
			hp = hp + hp*uint64(core.MinerNum-i+5)/50
			weight := miner.Cost[i] / core.MaxGuerdon / 5
			if weight < 100 {
				hp += weight
			} else {
				hp += 100
			}
			r.Miner = true
			break
		}
	}

	power += hp
	power += (parent.HashPower / 4)
	power += preRel.HashPower
	power -= (preRel.HashPower >> 40)
	if r.Producer == preRel.Producer {
		power -= 7
	}

	r.HashPower = power
}

func getReliability(b *core.StBlock) TReliability {
	var selfRel TReliability

	selfRel.Key = b.Key
	selfRel.Index = b.Index
	selfRel.Previous = b.Previous
	selfRel.Time = b.Time
	selfRel.Parent = b.Parent
	selfRel.LeftChild = b.LeftChild
	selfRel.RightChild = b.RightChild
	selfRel.Producer = b.Producer

	selfRel.Recalculation(b.Chain)
	return selfRel
}
