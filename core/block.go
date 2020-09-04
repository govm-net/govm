package zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/govm-net/govm/conf"
	"github.com/govm-net/govm/database"
	"github.com/govm-net/govm/runtime"
	"github.com/govm-net/govm/wallet"
)

// const minHPLimit = 15
// todo
const minHPLimit = 2

// StBlock StBlock
type StBlock struct {
	Block
	Key            Hash
	HashpowerLimit uint64
	sign           []byte
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

	hashPowerLimit = hashPowerLimit / 1000
	if hashPowerLimit < minHPLimit {
		hashPowerLimit = minHPLimit
	}

	out.HashpowerLimit = hashPowerLimit

	if pStat.ID == 1 && chain > 1 {
		pStat.Time = pStat.Time + blockSyncMax + blockSyncMin + TimeSecond
	} else {
		pStat.Time += blockInterval
	}

	out.Previous = pStat.Key
	out.Producer = producer
	out.Time = pStat.Time

	out.Chain = chain
	out.Index = pStat.ID + 1

	if pStat.Chain > 1 {
		var key Hash
		var tmp BlockInfo
		getDataFormLog(chain/2, logBlockInfo{}, runtime.Encode(pStat.ParentID+1), &key)
		getDataFormLog(chain/2, logBlockInfo{}, key[:], &tmp)
		if out.Index != 2 && !key.Empty() && out.Time > tmp.Time && out.Time-tmp.Time > blockSyncMin {
			var key2 Hash
			getDataFormLog(chain/2, logBlockInfo{}, runtime.Encode(pStat.ParentID+2), &key2)
			getDataFormLog(chain/2, logBlockInfo{}, key2[:], &tmp)
			if !key2.Empty() && out.Time > tmp.Time && out.Time-tmp.Time > blockSyncMin {
				out.Parent = key2
			} else {
				out.Parent = key
			}
			// assert(out.Time-tmp.Time <= blockSyncMax)
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
			var key2 Hash
			getDataFormLog(2*chain, logBlockInfo{}, runtime.Encode(pStat.LeftChildID+2), &key2)
			getDataFormLog(2*chain, logBlockInfo{}, key2[:], &tmp)
			if !key2.Empty() && out.Time > tmp.Time && out.Time-tmp.Time > blockSyncMin {
				out.LeftChild = key2
			} else {
				out.LeftChild = key
			}
			// assert(out.Time-tmp.Time <= blockSyncMax)
		} else if pStat.LeftChildID == 1 {
			getDataFormLog(chain, logBlockInfo{}, runtime.Encode(pStat.LeftChildID), &key)
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
			var key2 Hash
			getDataFormLog(2*chain+1, logBlockInfo{}, runtime.Encode(pStat.RightChildID+2), &key2)
			getDataFormLog(2*chain+1, logBlockInfo{}, key2[:], &tmp)
			if !key2.Empty() && out.Time > tmp.Time && out.Time-tmp.Time > blockSyncMin {
				out.RightChild = key2
			} else {
				out.RightChild = key
			}
			// assert(out.Time-tmp.Time <= blockSyncMax)
		} else if pStat.RightChildID == 1 {
			getDataFormLog(chain, logBlockInfo{}, runtime.Encode(pStat.RightChildID), &key)
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
// func (b *StBlock) SetTransList(list []Hash) {
// 	if len(list) == 0 {
// 		return
// 	}
// 	hk := GetHashOfTransList(list)
// 	b.TransListHash = hk
// 	var data []byte
// 	for _, it := range list {
// 		data = append(data, it[:]...)
// 	}
// 	runtime.AdminDbSet(dbTransList{}, b.Chain, hk[:], data, 2<<50)
// }

//GetSignData GetSignData
func (b *StBlock) GetSignData() []byte {
	b.Nonce++
	data := runtime.Encode(b.Block)
	return data
}

// SetSign SetSign
func (b *StBlock) SetSign(sign []byte) error {
	//b.SignLen = uint16(len(sign))
	b.sign = sign
	return nil
}

// Output Output
func (b *StBlock) Output() []byte {
	data := make([]byte, 1, 1000)
	data[0] = uint8(len(b.sign))
	data = append(data, b.sign...)
	data = append(data, runtime.Encode(b.Block)...)
	k := runtime.GetHash(data)
	runtime.Decode(k, &b.Key)
	return data
}

// ParseTransList parse transaction list
func ParseTransList(data []byte) []Hash {
	if len(data) == 0 {
		return nil
	}
	size := len(data)
	num := size / HashLen
	if num*HashLen != size {
		return nil
	}

	transList := make([]Hash, num)
	for i := 0; i < num; i++ {
		n := runtime.Decode(data, &transList[i])
		data = data[n:]
	}
	return transList
}

// TransListToBytes transList to bytes
func TransListToBytes(in []Hash) []byte {
	if len(in) == 0 {
		return nil
	}
	var out []byte
	for _, it := range in {
		out = append(out, it[:]...)
	}
	return out
}

// WriteTransList write transList to db
func WriteTransList(chain uint64, transList []Hash) error {
	hk := GetHashOfTransList(transList)
	data := TransListToBytes(transList)
	return runtime.AdminDbSet(dbTransList{}, chain, hk[:], data, 2<<50)
}

// ReadTransList read trans list from db
func ReadTransList(chain uint64, key []byte) []byte {
	data, _ := runtime.DbGet(dbTransList{}, chain, key)
	return data
}

// TransListExist return true when transList exist
func TransListExist(chain uint64, key []byte) bool {
	return runtime.DbExist(dbTransList{}, chain, key)
}

// GetHashPowerLimit ge hash power limit
func GetHashPowerLimit(chain uint64) uint64 {
	var hashPowerLimit uint64
	getDataFormDB(chain, dbStat{}, []byte{StatHashPower}, &hashPowerLimit)
	return hashPowerLimit / 1000
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

	rst := wallet.Recover(out.Producer[:], out.sign, bData)
	if !rst {
		log.Println("fail to recover block")
		return nil
	}
	h := runtime.GetHash(data)
	runtime.Decode(h, &out.Key)

	return out
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
	if len(stream) > 0 {
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
func IsMiner(chain uint64, user []byte) bool {
	var startBlock uint64
	var pStat BaseInfo
	getDataFormDB(chain, dbStat{}, []byte{StatBaseInfo}, &pStat)
	getDataFormDB(chain, dbMiner{}, user, &startBlock)

	if startBlock > 0 && startBlock <= pStat.ID {
		return true
	}

	return false
}

// IsAdmin is admin
func IsAdmin(chain uint64, user []byte) bool {
	var adminList [AdminNum]Address
	var a Address
	runtime.Decode(user, &a)
	getDataFormDB(chain, dbStat{}, []byte{StatAdmin}, &adminList)
	for _, it := range adminList {
		if it == a {
			return true
		}
	}
	return false
}

// BlockOnTheChain return true if the block is on the chain
func BlockOnTheChain(chain uint64, key []byte) bool {
	var block BlockInfo
	getDataFormLog(chain, logBlockInfo{}, key[:], &block)
	return block.Index > 0
}

// CreateBiosTrans CreateBiosTrans
func CreateBiosTrans(chain uint64) {
	c := conf.GetConf()
	runtime.NewApp(chain, c.CorePackName, nil)
}

// ProcessBlockOfChain process block
func ProcessBlockOfChain(chain uint64, key []byte) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			log.Printf("fail to process block,chain:%d,key:%x,err:%s\n", chain, key, e)
			log.Println(string(debug.Stack()))
			err = fmt.Errorf("recover:%s", e)
		}
	}()
	client := database.GetClient()
	err = client.OpenFlag(chain, key)
	if err != nil {
		log.Println("fail to open Flag,", err)
		f := client.GetLastFlag(chain)
		client.Cancel(chain, f)
		client.Rollback(chain, f)
		return err
	}
	defer client.Cancel(chain, key)

	run(chain, key)
	client.Commit(chain, key)
	return err
}

// GetBlockInterval get the interval time of between blocks
func GetBlockInterval(chain uint64) uint64 {
	var out uint64
	getDataFormDB(chain, dbStat{}, []byte{StatBlockInterval}, &out)
	return out
}

// GetBlockSizeLimit get the limit size of block
func GetBlockSizeLimit(chain uint64) uint64 {
	var out uint64
	getDataFormDB(chain, dbStat{}, []byte{StatBlockSizeLimit}, &out)
	return out
}

// GetParentBlockOfChain get parent block of chain
func GetParentBlockOfChain(chain uint64) Hash {
	var out Hash
	getDataFormDB(chain, dbStat{}, []byte{StatParentKey}, &out)
	return out
}
