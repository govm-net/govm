package handler

import (
	"bytes"
	"encoding/hex"
	"github.com/lengzhao/govm/event"
	"log"
	"sync"
	"time"

	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/govm/wallet"
)

// chain->index->blockKey->reliability
type tProcessMgr struct {
	mu     sync.Mutex
	stop   bool
	wg     sync.WaitGroup
	Chains map[uint64]*sync.Mutex
}

var procMgr tProcessMgr

func init() {
	procMgr.Chains = make(map[uint64]*sync.Mutex)
	var i uint64
	for i = 0; i < 100; i++ {
		procMgr.Chains[i] = new(sync.Mutex)
	}
}

// GraceStop grace stop
func GraceStop() {
	procMgr.stop = true
	procMgr.wg.Wait()
}

func getBestBlock(chain, index uint64, preKey []byte) (core.TReliability,int) {
	var relia core.TReliability
	var num int
	ib := core.ReadIDBlocks(chain, index)
	now := time.Now().Unix()
	for i, b := range ib.Blocks {
		if b.Empty() {
			break
		}
		num++
		key := b[:]
		log.Printf("getBestBlock,chain:%d,index:%d,key:%x,%d\n", chain, index, b, i)
		data := core.ReadBlockData(chain, key)
		if data == nil {
			log.Printf("not exist block data.chain:%d,key:%x\n", chain, key)
			download(chain, hex.EncodeToString(key), nil, &messages.ReqBlock{Chain: chain, Key: key})
			continue
		}
		block := core.DecodeBlock(data)
		if block == nil {
			continue
		}
		if len(preKey) > 0 && bytes.Compare(preKey, block.Previous[:]) != 0 {
			continue
		}
		// time.Second
		if block.Time > uint64(now+5)*1000 {
			continue
		}
		var lost bool
		for _, t := range block.TransList {
			if core.IsExistTransaction(chain, t[:]) {
				continue
			}
			keyStr := hex.EncodeToString(t[:])
			msg := &messages.ReqTransaction{Chain: chain, Key: t[:]}
			download(chain, keyStr, nil, msg)
			lost = true
		}
		if lost {
			continue
		}
		rel := block.GetReliability()
		stat := core.ReadBlockRunStat(chain,key)
		
		if stat.RollbackTimes > 5 {
			continue
		} 
		if stat.RunTimes > stat.RunSuccessCount+3 {
			continue
		}
		if rel.Cmp(relia) > 0 {
			relia = rel
		}
	}
	return relia,num
}

func processEvent(chain uint64) {
	if procMgr.stop {
		return
	}
	defer func() {
		e := recover()
		if e != nil {
			log.Println("something error,", e)
		}
	}()

	procMgr.wg.Add(1)
	defer procMgr.wg.Done()
	procMgr.mu.Lock()
	c, ok := procMgr.Chains[chain]
	procMgr.mu.Unlock()
	if !ok {
		log.Println("error: fail to get lock of chain:", chain)
		return
	}
	c.Lock()
	defer c.Unlock()
	if procMgr.stop {
		return
	}
	index := core.GetLastBlockIndex(chain)
	ib := core.ReadIDBlocks(chain, index+1)
	if index == 0 {
		if chain > 1 {
			cInfo := core.GetChainInfo(chain / 2)
			if cInfo.LeftChildID == 1 || cInfo.RightChildID == 1 {
				c := conf.GetConf()
				data := core.ReadTransactionData(chain/2, c.FirstTransName)
				core.WriteTransaction(chain, data)
				data = core.ReadBlockData(chain/2, ib.Blocks[0][:])
				core.WriteTransaction(chain, data)
			} else {
				return
			}
		}

		core.CreateBiosTrans(chain)
	}
	var relia core.TReliability
	t := core.GetBlockTime(chain)
	now := time.Now().Unix()
	if t+120000 > uint64(now)*1000 {
		preKey := core.GetTheBlockKey(chain, index-7)
		relia,_ = getBestBlock(chain, index-6, preKey)
		key := core.GetTheBlockKey(chain, index-6)
		if !relia.Key.Empty() && bytes.Compare(key, relia.Key[:]) != 0 {
			log.Printf("processEvent,replace index-6. index:%d,key:%x,relia:%x\n", index, key, relia.Key)
			stat := core.ReadBlockRunStat(chain, preKey)
			stat.RollbackTimes++
			core.SaveBlockRunStat(chain, preKey, stat)
			dbRollBack(chain, index-6, key)
			log.Println("dbRollBack1")
			go processEvent(chain)
			return
		}
	}
	preKey := core.GetTheBlockKey(chain, index)
	relia,num := getBestBlock(chain, index+1, preKey)
	if relia.Key.Empty(){
		if num == 0{
			return
		}
		t := core.GetBlockTime(chain)
		t += 600000
		if t < uint64(now)*1000 {
			log.Printf("dbRollBack one block. block time:%d now:%d,index:%d,key:%x\n", t-600000, now, index, preKey)
			dbRollBack(chain, index, preKey)
			stat := core.ReadBlockRunStat(chain, preKey)
			stat.RollbackTimes++
			core.SaveBlockRunStat(chain, preKey, stat)
			go processEvent(chain)
		}
		return
	}
	stat := core.ReadBlockRunStat(chain, relia.Key[:])
	stat.RunTimes++
	log.Printf("start to process block,chain:%d,index:%d,key:%x\n", chain, relia.Index, relia.Key)
	err := blockRun(chain, relia.Key[:])
	if err != nil {
		log.Printf("fail to process block,chain:%d,index:%d,key:%x,error:%s\n", chain, index+1, relia.Key, err)
		core.SaveBlockRunStat(chain, relia.Key[:], stat)
		return
	}
	stat.RunSuccessCount++
	core.SaveBlockReliability(chain, relia.Key[:], relia)
	core.SaveBlockRunStat(chain, relia.Key[:], stat)

	info := messages.BlockInfo{}
	info.Chain = chain
	info.Index = index + 1
	info.Key = relia.Key[:]
	mgr.net.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})

	if relia.Time+200000 > uint64(now)*1000 {
		go doMine(chain)
	}

	cInfo := core.GetChainInfo(chain)
	if cInfo.LeftChildID == 1 {
		go processEvent(chain * 2)
	} else if cInfo.RightChildID == 1 {
		go processEvent(chain*2 + 1)
	}

	go processEvent(chain)
}

func getHashPower(in []byte) uint64 {
	var out uint64
	for _, item := range in {
		out += 8
		if item != 0 {
			for item > 0 {
				out--
				item = item >> 1
			}
			return out
		}
	}
	return out
}

func doMine(chain uint64) {
	procMgr.wg.Add(1)
	defer procMgr.wg.Done()
	procMgr.mu.Lock()
	cl, ok := procMgr.Chains[chain]
	if !ok {
		procMgr.Chains[chain] = new(sync.Mutex)
	}
	procMgr.mu.Unlock()
	cl.Lock()
	defer cl.Unlock()

	c := conf.GetConf()
	addr := core.Address{}
	runtime.Decode(c.WalletAddr, &addr)
	block := core.NewBlock(chain, addr)

	transList,size := getTransListForMine(chain)

	block.SetTransList(transList)
	block.Size = uint32(size)
	var key core.Hash
	var oldHP uint64

	for {
		if procMgr.stop {
			return
		}
		now := time.Now().Unix() * 1000
		if block.Time < uint64(now) {
			break
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
		// rst := wallet.Recover(c.WalletAddr, sign, signData)
		// if !rst {
		// 	log.Printf("fail to recover, address: %x, sign:%x\n", c.WalletAddr, sign)
		// 	panic("fail to recover,block mine")
		// }
		block.SetSign(sign)
		data := block.Output()

		hp := getHashPower(block.Key[:])
		if hp <= block.HashpowerLimit {
			block.Nonce++
			// log.Printf("drop hash:%x,data:%x\n", key, signData[:6])
			continue
		}
		if hp > oldHP {
			core.WriteBlock(chain, data)
			key = block.Key
			oldHP = hp
		}
	}

	if oldHP == 0 {
		log.Printf("fail to doMine,error oldHP")
		return
	}

	log.Printf("mine one blok,chain:%d,index:%d,hashpower:%d,hp limit:%d,key:%x\n",
		chain, block.Index, oldHP, block.HashpowerLimit, key[:])

	ib := core.ReadIDBlocks(chain, block.Index)
	hp := oldHP
	for i, b := range ib.Blocks {
		if key == b {
			break
		}
		if hp > ib.HashPower[i] {
			// log.Printf("IDBlocks switch,index:%d,i:%d,old:%x,new:%x\n", block.Index, i, b, key)
			hp, ib.HashPower[i] = ib.HashPower[i], hp
			key, ib.Blocks[i] = b, key
		}
	}
	core.SaveIDBlocks(chain, block.Index, ib)

	go processEvent(chain)
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
	index := core.GetLastBlockIndex(chain)
	index += 50
	miner := core.GetMinerInfo(chain, index)
	if c.CostOfRegMiner < miner.Cost[5] {
		return
	}
	cAddr := core.Address{}
	runtime.Decode(c.WalletAddr, &cAddr)
	trans := core.NewTransaction(chain, cAddr)
	trans.Time = core.GetBlockTime(chain)
	trans.CreateRegisterMiner(0, index, c.CostOfRegMiner)
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
