package handler

import (
	"bytes"
	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/event"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/govm/wallet"
	"log"
	"math/rand"
	"time"
)

func getTransListForMine(chain uint64) ([]core.Hash, uint64) {
	var preKey []byte
	var size uint64
	var trans transInfo
	out := make([]core.Hash, 0)
	limit := core.GetBlockSizeLimit(chain)
	t := core.GetBlockInterval(chain)
	start := time.Now().Unix()
	c := conf.GetConf()
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
			info := core.GetTransInfo(chain, trans.Key[:])
			if info.BlockID > 0 {
				if info.BlockID+6 < lastID {
					deleteTransInfo(chain, trans.Key[:])
				}
				continue
			}

			preKey = trans.Key[:]
			if size+uint64(trans.Size) > limit {
				continue
			}
			if !believable(chain, trans.User[:]) && bytes.Compare(trans.User[:], c.WalletAddr) != 0 {
				continue
			}
			break
		}
		return trans.Key
	})
	if err != nil {
		saveBlackItem(chain, trans.User[:])
		deleteTransInfo(chain, trans.Key[:])
	}

	return out, size
}

func doMine(chain uint64, force bool) {
	c := conf.GetConf()
	if !c.DoMine {
		return
	}
	if c.ChainOfMine != 0 && c.ChainOfMine != chain {
		return
	}

	select {
	case procMgr.mineLock <- 1:
		log.Println("start to doMine:", chain)
	default:
		return
	}
	defer func() {
		<-procMgr.mineLock
		log.Println("finish doMine:", chain)
	}()

	addr := core.Address{}
	runtime.Decode(c.WalletAddr, &addr)
	block := core.NewBlock(chain, addr)
	var transList []core.Hash
	var size uint64

	if !force {
		if block.Time+tHour < uint64(time.Now().Unix())*1000 {
			return
		}

		count := GetMineCount(chain, block.Previous[:])
		if count > 1 {
			return
		}
		SetMineCount(chain, block.Previous[:], count+1)

		transList, size = getTransListForMine(chain)
	}

	block.SetTransList(transList)
	block.Size = uint32(size)
	block.Nonce = rand.Uint64()

	var oldRel core.TReliability

	timeout := time.Now().Unix() + 20

	for {
		now := time.Now().Unix()
		if timeout < now && block.Time < uint64(now)*1000 {
			if !force {
				break
			}
			if force && oldRel.HashPower != 0 {
				break
			}
		}
		signData := block.GetSignData()
		sign := wallet.Sign(c.PrivateKey, signData)
		if len(sign) == 0 {
			continue
		}
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}

		block.SetSign(sign)
		data := block.Output()

		hp := getHashPower(block.Key[:])
		if hp < block.HashpowerLimit {
			block.Nonce++
			// log.Printf("drop hash:%x,data:%x\n", key, signData[:6])
			continue
		}
		rel := block.GetReliability()
		if rel.Cmp(oldRel) > 0 {
			oldRel = rel
			core.WriteBlock(chain, data)
			core.SaveBlockReliability(chain, block.Key[:], rel)
			info := messages.BlockInfo{}
			info.Chain = chain
			info.Index = rel.Index
			info.Key = rel.Key[:]
			info.HashPower = rel.HashPower
			info.PreKey = rel.Previous[:]
			network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
			log.Printf("mine one blok,chain:%d,index:%d,hashpower:%d,hp limit:%d,trans:%d,key:%x\n",
				chain, rel.Index, rel.HashPower, block.HashpowerLimit, len(transList), rel.Key)
		}
	}

	if oldRel.HashPower == 0 {
		log.Printf("fail to doMine,error oldHP,limit:%d\n", block.HashpowerLimit)
		count := GetMineCount(chain, block.Previous[:])
		if count > 0 {
			SetMineCount(chain, block.Previous[:], count-1)
		}
		return
	}
}

func autoRegisterMiner(chain uint64) {
	c := conf.GetConf()
	if c.CostOfRegMiner < 100 {
		return
	}
	if chain != c.ChainOfMine && c.ChainOfMine != 0 {
		return
	}
	cost := core.GetUserCoin(chain, c.WalletAddr)
	if cost < c.CostOfRegMiner {
		return
	}
	t := core.GetBlockTime(chain)
	if t+5*tMinute < uint64(time.Now().Unix())*1000 {
		return
	}
	index := core.GetLastBlockIndex(chain)
	index += 50
	miner := core.GetMinerInfo(chain, index)
	if c.CostOfRegMiner < miner.Cost[core.MinerNum-1] {
		return
	}

	id := runtime.Encode(index)
	stream := ldb.LGet(chain, ldbMiner, id)
	if len(stream) > 0 {
		return
	}
	ldb.LSet(chain, ldbMiner, id, runtime.Encode(c.CostOfRegMiner))

	cAddr := core.Address{}
	runtime.Decode(c.WalletAddr, &cAddr)
	trans := core.NewTransaction(chain, cAddr)
	trans.Time = core.GetBlockTime(chain)
	trans.CreateRegisterMiner(0, index, c.CostOfRegMiner)
	trans.Energy = c.EnergyOfTrans
	td := trans.GetSignData()
	sign := wallet.Sign(c.PrivateKey, td)
	if len(c.SignPrefix) > 0 {
		s := make([]byte, len(c.SignPrefix))
		copy(s, c.SignPrefix)
		sign = append(s, sign...)
	}
	trans.SetSign(sign)
	td = trans.Output()

	msg := new(messages.NewTransaction)
	msg.Chain = chain
	msg.Key = trans.Key[:]
	msg.Data = td
	event.Send(msg)
	// log.Println("SendInternalMsg autoRegisterMiner:", msg)
}
