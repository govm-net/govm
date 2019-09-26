package a365d2b302434dac708688612b3b86a486d59c01071be7b2738eb8c6c028fd413

import (
	"encoding/hex"
	"github.com/lengzhao/govm/conf"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/govm/wallet"
	"log"
	"os"
	"time"
)

// StBlock StBlock
type StBlock struct {
	Block
	Key            Hash
	sign           []byte
	TransList      []Hash
	HashpowerLimit uint64
}

// TReliability Reliability of block
type TReliability struct {
	Key        Hash    `json:"key,omitempty"`
	Previous   Hash    `json:"previous,omitempty"`
	Parent     Hash    `json:"parent,omitempty"`
	LeftChild  Hash    `json:"left_child,omitempty"`
	RightChild Hash    `json:"right_child,omitempty"`
	Producer   Address `json:"producer,omitempty"`
	Time       uint64  `json:"time,omitempty"`
	Index      uint64  `json:"index,omitempty"`
	HashPower  uint64  `json:"hash_power,omitempty"`
	Miner      bool    `json:"miner,omitempty"`
	Ready      bool    `json:"ready,omitempty"`
}

const ldbReliability = "reliability" //blockKey:relia
var ldb *database.LDB

func init() {
	ldb = database.NewLDB("reliability.db", 2000)
	if ldb == nil {
		log.Println("fail to open ldb,local.db")
		os.Exit(2)
	}
	ldb.SetCache(ldbReliability)
}

// NewBlock new block
/*
	1. NewBlock
	2. SetTransList
	3. update StBlock.Size,PreCheckSum,HashPower,Parent...
	4. GetSignData
	5. SetSign
	6. Output
*/
func NewBlock(chain uint64, producer Address) *StBlock {
	var hashPowerLimit uint64
	var blockInterval uint64
	var pStat BaseInfo
	out := new(StBlock)
	getDataFormDB(chain, dbStat{}, []byte{StatBaseInfo}, &pStat)
	getDataFormDB(chain, dbStat{}, []byte{StatHashPower}, &hashPowerLimit)
	getDataFormDB(chain, dbStat{}, []byte{StatBlockInterval}, &blockInterval)

	hashPowerLimit /= 1000
	if hashPowerLimit < 10 {
		hashPowerLimit = 10
	}
	out.HashpowerLimit = hashPowerLimit

	if pStat.ID == 1 && chain > 1 {
		pStat.Time = pStat.Time + blockSyncMax + blockSyncMin + maxBlockInterval
	} else {
		pStat.Time += blockInterval
	}

	out.Previous = pStat.Key
	out.Producer = producer
	out.Time = pStat.Time

	out.Chain = chain
	out.Index = pStat.ID + 1

	out.TransList = make([]Hash, 0)

	if pStat.Chain > 1 {
		var key Hash
		var tmp BlockInfo
		getDataFormLog(chain/2, logBlockInfo{}, runtime.Encode(pStat.ParentID+1), &key)
		getDataFormLog(chain/2, logBlockInfo{}, key[:], &tmp)
		if !key.Empty() && out.Time > tmp.Time && out.Time-tmp.Time > blockSyncMin {
			out.Parent = key
		} else {
			getDataFormLog(chain/2, logBlockInfo{}, runtime.Encode(pStat.ParentID), &key)
			out.Parent = key
		}
	}
	if pStat.LeftChildID > 0 {
		var key Hash
		var tmp BlockInfo
		getDataFormLog(2*chain, logBlockInfo{}, runtime.Encode(pStat.LeftChildID+1), &key)
		getDataFormLog(2*chain, logBlockInfo{}, key[:], &tmp)
		if !key.Empty() && out.Time > tmp.Time && out.Time-tmp.Time > blockSyncMin {
			out.LeftChild = key
		} else {
			getDataFormLog(2*chain, logBlockInfo{}, runtime.Encode(pStat.LeftChildID), &key)
			out.LeftChild = key
		}
	}
	if pStat.RightChildID > 0 {
		var key Hash
		var tmp BlockInfo
		getDataFormLog(2*chain+1, logBlockInfo{}, runtime.Encode(pStat.RightChildID+1), &key)
		getDataFormLog(2*chain+1, logBlockInfo{}, key[:], &tmp)
		if !key.Empty() && out.Time > tmp.Time && out.Time-tmp.Time > blockSyncMin {
			out.RightChild = key
		} else {
			getDataFormLog(2*chain+1, logBlockInfo{}, runtime.Encode(pStat.RightChildID), &key)
			out.RightChild = key
		}
	}

	return out
}

func getTransHash(t1, t2 Hash) Hash {
	hashKey := Hash{}
	data := append(t1[:], t2[:]...)
	hash := runtime.GetHash(data)
	runtime.Decode(hash, &hashKey)
	return hashKey
}

// SetTransList SetTransList
func (b *StBlock) SetTransList(list []Hash) {
	n := len(list)
	if n == 0 {
		b.TransListHash = Hash{}
		return
	}
	b.TransList = make([]Hash, n)
	for i, t := range list {
		b.TransList[i] = t
	}
	tmpList := list
	for len(tmpList) > 1 {
		n := len(tmpList)
		if n%2 != 0 {
			tmpList = append(tmpList, Hash{})
			n++
			// log.Println("list number++:", n)
		}
		for i := 0; i < n/2; i++ {
			tmpList[i] = getTransHash(tmpList[2*i], tmpList[2*i+1])
		}
		tmpList = tmpList[:n/2]
	}
	b.TransListHash = tmpList[0]
	// log.Printf("TransListHash:%x", b.TransListHash)
}

//GetSignData GetSignData
func (b *StBlock) GetSignData() []byte {
	b.Nonce++
	data := runtime.Encode(b.Block)
	data = append(data, b.streamTransList()...)
	return data
}

// SetSign SetSign
func (b *StBlock) SetSign(sign []byte) error {
	//b.SignLen = uint16(len(sign))
	b.sign = sign
	return nil
}

func (b *StBlock) streamTransList() []byte {
	var out []byte
	for _, t := range b.TransList {
		out = append(out, t[:]...)
	}
	//log.Printf("streamTransList.chain:%d,trans:%d,%x\n", b.Chain, len(b.TransList), out)
	return out
}

// Output Output
func (b *StBlock) Output() []byte {
	data := make([]byte, 1, 1000)
	data[0] = uint8(len(b.sign))
	data = append(data, b.sign...)
	data = append(data, runtime.Encode(b.Block)...)
	data = append(data, b.streamTransList()...)
	k := runtime.GetHash(data)
	runtime.Decode(k, &b.Key)
	return data
}

// GetTransList GetTransList
func (b *StBlock) GetTransList() []string {
	var out []string
	for _, key := range b.TransList {
		keyStr := hex.EncodeToString(key[:])
		out = append(out, keyStr)
	}
	return out
}

// DecodeBlock decode data and check sign, check hash
func DecodeBlock(data []byte) *StBlock {
	out := new(StBlock)
	out.sign = data[1 : data[0]+1]
	bData := data[data[0]+1:]
	n := runtime.Decode(bData, &out.Block)
	stream := bData[n:]
	if len(stream)%HashLen != 0 {
		return nil
	}
	if out.Index < 1 {
		return nil
	}
	if out.Time > uint64(time.Now().Unix()+5)*1000 {
		return nil
	}

	rst := wallet.Recover(out.Producer[:], out.sign, bData)
	if !rst {
		log.Println("fail to recover block")
		return nil
	}
	h := runtime.GetHash(data)
	runtime.Decode(h, &out.Key)
	transList := make([]Hash, len(stream)/HashLen)
	for i := 0; i < len(transList); i++ {
		n = runtime.Decode(stream, &transList[i])
		stream = stream[n:]
	}
	listKey := Hash{}
	copy(listKey[:], out.TransListHash[:])

	out.SetTransList(transList)
	if listKey != out.TransListHash {
		return nil
	}

	return out
}

// GetReliability get block reliability
func (b *StBlock) GetReliability() TReliability {
	var power uint64
	var selfRel TReliability
	var miner Miner

	preRel := ReadBlockReliability(b.Chain, b.Previous[:])
	parent := ReadBlockReliability(b.Chain/2, b.Parent[:])
	getDataFormDB(b.Chain, dbMining{}, runtime.Encode(b.Index), &miner)

	for i := 0; i < minerNum; i++ {
		if miner.Miner[i] == b.Producer {
			power += uint64(minerNum-i) + 5
			selfRel.Miner = true
			break
		}
	}
	if b.Index == 1 {
		power += 1000
	}
	power += getHashPower(b.Key)
	power += parent.HashPower / 4
	power += preRel.HashPower
	power -= preRel.HashPower >> 30
	if b.Producer == preRel.Producer {
		power -= 7
	}

	selfRel.Key = b.Key
	selfRel.Index = b.Index
	selfRel.Previous = b.Previous
	selfRel.HashPower = power
	selfRel.Time = b.Time
	selfRel.Parent = b.Parent
	selfRel.LeftChild = b.LeftChild
	selfRel.RightChild = b.RightChild
	selfRel.Producer = b.Producer

	return selfRel
}

// Cmp compares x and y and returns:
//
//   +1 if x >  y
//   -1 if x <  y
//   0  if x =  y
func (x TReliability) Cmp(y TReliability) int {
	if x.HashPower > y.HashPower {
		return 1
	}
	if x.HashPower < y.HashPower {
		return -1
	}
	for i, b := range x.Key {
		if b > y.Key[i] {
			return -1
		}
		if b < y.Key[i] {
			return 1
		}
	}

	return 0
}

// IsExistBlock Determine whether block exists
func IsExistBlock(chain uint64, key []byte) bool {
	return runtime.DbExist(dbBlockData{}, chain, key)
}

// WriteBlock write block data to database
func WriteBlock(chain uint64, data []byte) error {
	key := runtime.GetHash(data)
	exist := runtime.DbExist(dbBlockData{}, chain, key)
	if exist {
		return nil
	}

	return runtime.AdminDbSet(dbBlockData{}, chain, key, data, 2<<50)
}

// DeleteBlock delete block
func DeleteBlock(chain uint64, key []byte) error {
	return runtime.AdminDbSet(dbBlockData{}, chain, key, nil, 0)
}

func getDataFormLog(chain uint64, db interface{}, key []byte, out interface{}) {
	if chain == 0 {
		return
	}
	stream, _ := runtime.LogRead(db, chain, key)
	if stream != nil {
		runtime.Decode(stream, out)
	}
}

// ReadBlockData read block data
func ReadBlockData(chain uint64, key []byte) []byte {
	stream, _ := runtime.DbGet(dbBlockData{}, chain, key)
	return stream
}

// GetChainInfo get chain info
func GetChainInfo(chain uint64) *BaseInfo {
	var pStat BaseInfo
	getDataFormDB(chain, dbStat{}, []byte{StatBaseInfo}, &pStat)
	return &pStat
}

// GetLastBlockIndex get the index of last block
func GetLastBlockIndex(chain uint64) uint64 {
	var pStat BaseInfo
	getDataFormDB(chain, dbStat{}, []byte{StatBaseInfo}, &pStat)
	return pStat.ID
}

// GetTheBlockKey get block key,if index==0,return last key
func GetTheBlockKey(chain, index uint64) []byte {
	var key Hash
	if chain == 0 {
		return nil
	}
	if index == 0 {
		var pStat BaseInfo
		getDataFormDB(chain, dbStat{}, []byte{StatBaseInfo}, &pStat)
		return pStat.Key[:]
	}
	getDataFormLog(chain, logBlockInfo{}, runtime.Encode(index), &key)
	if key.Empty() {
		return nil
	}
	return key[:]
}

// GetBlockTime get block time
func GetBlockTime(chain uint64) uint64 {
	var pStat BaseInfo
	getDataFormDB(chain, dbStat{}, []byte{StatBaseInfo}, &pStat)
	return pStat.Time
}

// IsMiner check miner
func IsMiner(chain, index uint64, user []byte) bool {
	var miner Miner
	var addr Address
	if len(user) != AddressLen {
		return false
	}
	runtime.Decode(user, &addr)
	getDataFormDB(chain, dbMining{}, runtime.Encode(index), &miner)

	for i := 0; i < minerNum; i++ {
		if miner.Miner[i] == addr {
			return true
		}
	}

	return false
}

// GetMinerInfo get miner info
func GetMinerInfo(chain, index uint64) Miner {
	var miner Miner
	var guerdon uint64
	getDataFormDB(chain, dbStat{}, []byte{StatGuerdon}, &guerdon)
	getDataFormDB(chain, dbMining{}, runtime.Encode(index), &miner)
	guerdon = 3*guerdon - 1
	for i := 0; i < minerNum; i++ {
		if miner.Cost[i] == 0 {
			miner.Cost[i] = guerdon
		}
	}

	return miner
}

// CreateBiosTrans CreateBiosTrans
func CreateBiosTrans(chain uint64) {
	c := conf.GetConf()
	err := database.OpenFlag(chain, c.FirstTransName)
	if err != nil {
		log.Println("fail to open Flag,", err)
		return
	}
	defer database.Cancel(chain, c.FirstTransName)
	data, _ := runtime.DbGet(dbTransactionData{}, chain, c.FirstTransName)
	trans := DecodeTrans(data)

	appCode := trans.Data
	appName := runtime.GetHash(appCode)
	log.Printf("first app: %x\n", appName)
	appCode[6] = appCode[6] | AppFlagRun
	runtime.NewApp(chain, appName, appCode)
}

// SaveBlockReliability save block reliability
func SaveBlockReliability(chain uint64, key []byte, rb TReliability) {
	if chain == 0 {
		return
	}
	ldb.LSet(chain, ldbReliability, key, runtime.Encode(rb))
}

// ReadBlockReliability get Reliability of block from db
func ReadBlockReliability(chain uint64, key []byte) (cl TReliability) {
	if chain == 0 {
		return
	}
	stream := ldb.LGet(chain, ldbReliability, key)
	if stream != nil {
		runtime.Decode(stream, &cl)
	}
	return
}

// DeleteBlockReliability delete reliability of block
func DeleteBlockReliability(chain uint64, key []byte) {
	if chain == 0 {
		return
	}
	ldb.LSet(chain, ldbReliability, key, nil)
}

func GetBlockInterval(chain uint64) uint64 {
	var out uint64
	getDataFormDB(chain, dbStat{}, []byte{StatBlockInterval}, &out)
	return out
}
