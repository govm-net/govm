package handler

import (
	"log"
	"math/rand"
	"time"

	"github.com/govm-net/govm/conf"
	core "github.com/govm-net/govm/core"
	"github.com/govm-net/govm/database"
	"github.com/govm-net/govm/messages"
	"github.com/govm-net/govm/runtime"
	"github.com/govm-net/govm/wallet"
)

var myHP *database.LRUCache
var myAddr core.Address

func init() {
	rand.Seed(time.Now().UnixNano())
	myHP = database.NewLRUCache(100 * blockHPNumber)

	runtime.Decode(conf.GetConf().WalletAddr, &myAddr)
}

func newBlockForMining(chain uint64) {
	var preKey []byte
	var size uint64
	var trans transInfo
	out := make([]core.Hash, 0)
	limit := core.GetBlockSizeLimit(chain)
	t := core.GetBlockInterval(chain)
	start := time.Now().Unix()
	block := core.NewBlock(chain, core.Address{})
	setBlockForMining(chain, block.Block)
	// c := conf.GetConf()
	lastID := core.GetLastBlockIndex(chain)
	err := core.CheckTransList(chain, func(chain uint64) core.Hash {
		if !trans.Key.Empty() {
			out = append(out, trans.Key)
			size += uint64(trans.Size)
		}
		for {
			now := time.Now().Unix()
			if uint64(now-start) > t/2000 {
				return core.Hash{}
			}

			trans = getNextTransInfo(chain, preKey)
			if trans.Key.Empty() {
				preKey = nil
				return trans.Key
			}

			preKey = trans.Key[:]
			info := core.GetTransInfo(chain, trans.Key[:])
			if info.BlockID > 0 {
				if info.BlockID+3 < lastID {
					deleteTransInfo(chain, trans.Key[:])
				}
				continue
			}
			if size+uint64(trans.Size) > limit {
				return core.Hash{}
			}
			break
		}
		return trans.Key
	})
	if err != nil {
		deleteTransInfo(chain, trans.Key[:])
	}
	if len(out) == 0 {
		return
	}

	block.TransListHash = core.GetHashOfTransList(out)
	setBlockForMining(chain, block.Block)
}

func doMining(chain uint64) {
	c := conf.GetConf()

	if !core.IsAdmin(chain, myAddr[:]) {
		return
	}

	old := GetBlockForMining(chain)
	if old != nil {
		if old.Time > getCoreTimeNow() {
			return
		}
	}

	block := core.NewBlock(chain, myAddr)
	if old != nil {
		if old.Previous == block.Previous && old.Parent == block.Parent {
			return
		}
	}

	block.Nonce = rand.Uint64()
	signData := block.GetSignData()
	sign := wallet.Sign(c.PrivateKey, signData)
	if len(sign) == 0 {
		return
	}
	if len(c.SignPrefix) > 0 {
		s := make([]byte, len(c.SignPrefix))
		copy(s, c.SignPrefix)
		sign = append(s, sign...)
	}

	block.SetSign(sign)
	data := block.Output()
	rel := getReliability(block)

	core.WriteBlock(chain, data)
	SaveBlockReliability(chain, block.Key[:], rel)
	setBlockToIDBlocks(chain, rel.Index, rel.Key, rel.HashPower)
	if !needBroadcastBlock(chain, rel) {
		return
	}
	info := messages.BlockInfo{}
	info.Chain = chain
	info.Index = rel.Index
	info.Key = rel.Key[:]
	info.HashPower = rel.HashPower
	info.PreKey = rel.Previous[:]
	network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
	log.Printf("mine one blok,chain:%d,index:%d,hashpower:%d,hp limit:%d,key:%x\n",
		chain, rel.Index, rel.HashPower, block.HashpowerLimit, rel.Key)

}

// GetMyHashPower get my average hashpower
func GetMyHashPower(chain uint64) uint64 {
	procMgr.mu.Lock()
	defer procMgr.mu.Unlock()
	var sum uint64
	var count uint64
	hpi := time.Now().Unix() / 60
	for i := hpi - blockHPNumber; i < hpi; i++ {
		v, ok := myHP.Get(keyOfBlockHP{chain, i})
		if ok {
			sum += v.(uint64)
			count++
		}
	}
	if count == 0 {
		return 0
	}

	return sum / count
}
