package handler

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/govm-net/govm/conf"
	core "github.com/govm-net/govm/core"
	"github.com/govm-net/govm/database"
	"github.com/govm-net/govm/event"
	"github.com/govm-net/govm/messages"
	"github.com/govm-net/govm/runtime"
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
	ldbBlockRunStat   = "block_run_stat"  //blockKey:stat
	ldbIDBlocks       = "id_blocks"       //index:blocks
	ldbSyncBlocks     = "sync_blocks"     //index:blockKey
	ldbTransList      = "trans_list"      //blockKey:transList
	ldbAllTransInfo   = "all_trans_info"  //transKey:info
	ldbInputTrans     = "input_trans"     //receive transfer,timeKey:transKey
	ldbOutputTrans    = "output_trans"    //create by self,timeKey:transKey
	ldbDownloading    = "downloading"     //key:time
	ldbReliability    = "reliability"     //blockKey:relia
	ldbBroadcastTrans = "broadcast_trans" //key:1
	ldbStatus         = "ldbStatus"       //status
)

const downloadTimeout = 10
const acceptTimeDeference = 120

var ldb *database.LDB
var timeDifference int64
var firstUpdateTime bool = true
var transForMinging map[uint64][]*transInfo

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
	ldb.SetNotDisk(ldbTransList, 1000)
	ldb.SetNotDisk(ldbDownloading, 2000)
	ldb.SetNotDisk(ldbReliability, 50000)
	ldb.SetNotDisk(ldbAllTransInfo, 50000)
	ldb.SetNotDisk(ldbBroadcastTrans, 50000)
	ldb.SetNotDisk(ldbStatus, 10000)
	transForMinging = make(map[uint64][]*transInfo)
	time.AfterFunc(time.Second*5, updateTimeDifference)
	time.AfterFunc(time.Second*2, startCheckBlock)
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
	v, _ := json.Marshal(value)
	ldb.LSet(chain, ldbTransList, key[:], v)
}

// GetTransList get trans list of block
func GetTransList(chain uint64, key []byte) []core.Hash {
	v := ldb.LGet(chain, ldbTransList, key[:])
	out := []core.Hash{}
	json.Unmarshal(v, &out)
	return out
}

type transInfo struct {
	core.TransactionHead
	Key      core.Hash
	Flag     int64
	Size     uint32
	Stat     uint32
	Selected uint32
}

func saveTransInfo(chain uint64, key []byte, info transInfo) {
	value := runtime.Encode(info)
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

func pushTransInfo(chain uint64, info *transInfo) {
	procMgr.mu.Lock()
	defer procMgr.mu.Unlock()
	lst := transForMinging[chain]
	if lst == nil {
		lst = make([]*transInfo, 0)
	}
	lst = append(lst, info)
	transForMinging[chain] = lst
	// log.Println("transInfo list1:", len(lst))
}

func popTransInfo(chain uint64) *transInfo {
	procMgr.mu.Lock()
	defer procMgr.mu.Unlock()
	lst := transForMinging[chain]
	// log.Println("transInfo list2:", len(lst))
	if len(lst) == 0 {
		return nil
	}
	out := lst[0]
	transForMinging[chain] = lst[1:]
	return out
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

func getData(chain uint64, tb string, key []byte, value interface{}) {
	v := ldb.LGet(chain, tb, key)
	if len(v) == 0 {
		return
	}
	runtime.Decode(v, value)
}

var dlMutex sync.Mutex

// get the time of download, if fresh, return false
func needDownload(chain uint64, key []byte) bool {
	var lastTime int64
	dlMutex.Lock()
	defer dlMutex.Unlock()
	rst := ldb.LGet(chain, ldbDownloading, key)
	if len(rst) > 0 {
		runtime.Decode(rst, &lastTime)
	}
	now := time.Now().Unix()
	if lastTime+downloadTimeout >= now {
		return false
	}
	ldb.LSet(chain, ldbDownloading, key, runtime.Encode(now))
	return true
}

// get the time of request, if fresh, return false
func needRequstID(chain, index uint64) bool {
	return needDownload(chain, runtime.Encode(index))
}

// TReliability Reliability of block
type TReliability struct {
	Key           core.Hash    `json:"key,omitempty"`
	Previous      core.Hash    `json:"previous,omitempty"`
	Parent        core.Hash    `json:"parent,omitempty"`
	LeftChild     core.Hash    `json:"left_child,omitempty"`
	RightChild    core.Hash    `json:"right_child,omitempty"`
	Producer      core.Address `json:"producer,omitempty"`
	TransListHash core.Hash    `json:"trans_list_hash,omitempty"`
	Time          uint64       `json:"time,omitempty"`
	Index         uint64       `json:"index,omitempty"`
	HashPower     uint64       `json:"hash_power,omitempty"`
	Miner         bool         `json:"miner,omitempty"`
	Ready         bool         `json:"ready,omitempty"`
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
	var parent, preRel TReliability

	if r.Index > 1 {
		preRel = ReadBlockReliability(chain, r.Previous[:])
		if chain > 1 {
			parent = ReadBlockReliability(chain/2, r.Parent[:])
		}
	}

	if r.Index == 1 {
		power = 1000
	}
	admins := core.GetAdminList(chain)

	hp := getHashPower(r.Key[:]) * 10
	for i, it := range admins {
		if it == r.Producer {
			id := r.Index % core.AdminNum
			if id < uint64(i) {
				id += core.AdminNum
			}
			hp = id - uint64(i) + 50
			r.Miner = true
			break
		}
	}

	power += hp
	power += (parent.HashPower / 4)
	power += preRel.HashPower
	power -= (preRel.HashPower >> 40)
	if r.Producer == preRel.Producer && power > core.AdminNum {
		power -= core.AdminNum
	}
	// log.Printf("rel,chain:%d,key:%x,index:%d,hp:%d\n", chain, r.Key, r.Index, power)

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
	selfRel.TransListHash = b.TransListHash

	selfRel.Recalculation(b.Chain)
	return selfRel
}

// some node time is not right, if it have
func updateTimeDifference() {
	server := conf.GetConf().TrustedServer
	if server == "" {
		log.Println("updateTimeDifference,server is null")
		return
	}
	time.AfterFunc(time.Hour*25, updateTimeDifference)
	start := time.Now().Unix()
	resp, err := http.Get(server + "/api/v1/time")
	if err != nil {
		log.Println("fail to updateTimeDifference,", err)
		return
	}
	end := time.Now().Unix()
	if end < start {
		end = start
	}
	if resp.StatusCode != http.StatusOK {
		log.Println("error response of updateTimeDifference,", resp.Status)
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("updateTimeDifference,fail to read body:", err)
		return
	}
	serverTime, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		log.Println("updateTimeDifference error:", string(data), err)
		return
	}
	selfTime := (start + end) / 2
	if serverTime > selfTime+acceptTimeDeference || selfTime > serverTime+acceptTimeDeference {
		log.Println("updateTimeDifference error, time difference over 120s.",
			serverTime, selfTime, acceptTimeDeference)
		if firstUpdateTime {
			fmt.Println("error: system time error,update system time or change trusted_server in conf/conf")
			os.Exit(2)
		}
		return
	}
	firstUpdateTime = false
	timeDifference = serverTime - selfTime
	log.Println("updateTimeDifference", timeDifference, start, end)
}

func getCoreTimeNow() uint64 {
	now := time.Now().Unix() + timeDifference
	return uint64(now) * 1000
}

type bcBlock struct {
	Key core.Hash
	HP  uint64
}

func needBroadcastBlock(chain uint64, rel TReliability) bool {
	var best TReliability
	var dbKey = []byte("best")

	if rel.Time+2*tMinute < getCoreTimeNow() {
		return false
	}

	stream := ldb.LGet(chain, ldbStatus, dbKey)
	if len(stream) > 0 {
		json.Unmarshal(stream, &best)
	}
	if rel.Cmp(best) <= 0 {
		return false
	}

	log.Printf("new best,chain:%d,key:%x,index:%d,hp:%d,old hp:%d,old_id:%d,old_key:%x\n",
		chain, rel.Key, rel.Index, rel.HashPower,
		best.HashPower, best.Index, best.Key)
	stream, _ = json.Marshal(rel)
	ldb.LSet(chain, ldbStatus, dbKey, stream)

	return true
}

// GetBlockForMining get block info for mine
func GetBlockForMining(chain uint64) *core.StBlock {
	var out core.StBlock
	var dbKey = []byte("mining")
	data := ldb.LGet(chain, ldbStatus, dbKey)
	if len(data) == 0 {
		return nil
	}
	err := json.Unmarshal(data, &out)
	if err != nil {
		log.Println("fail to unmarshal.", err)
		return nil
	}
	if out.Time+tMinute < getCoreTimeNow() {
		return nil
	}
	return &out
}

func setBlockForMining(chain uint64, block core.StBlock) {
	var dbKey = []byte("mining")
	data, err := json.Marshal(block)
	if err != nil {
		log.Println(err)
	}
	ldb.LSet(chain, ldbStatus, dbKey, data)
	msg := new(messages.BlockForMining)
	msg.Chain = chain
	msg.Data = data
	event.Send(msg)
}
