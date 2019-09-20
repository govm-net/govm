package handler

import (
	"bytes"
	"github.com/lengzhao/govm/event"
	"log"
	"math/rand"
	"runtime/debug"
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
	mu       sync.Mutex
	wait     map[uint64]chan int
	Chains   map[uint64]chan int
	mineLock chan int
}

var procMgr tProcessMgr

func init() {
	procMgr.wait = make(map[uint64]chan int)
	procMgr.Chains = make(map[uint64]chan int)
	procMgr.mineLock = make(chan int, 2)

	time.AfterFunc(time.Second*5, timeoutFunc)
}

func timeoutFunc() {
	time.AfterFunc(time.Second*20, timeoutFunc)
	processEvent(1)
}

func getBestBlock(chain, index uint64) core.TReliability {
	var relia, rel core.TReliability
	var num int
	ib := core.ReadIDBlocks(chain, index)
	now := time.Now().Unix()
	for i, b := range ib.Blocks {
		if b.Empty() {
			continue
		}
		num++
		key := b[:]
		rel = core.ReadBlockReliability(chain, key)
		log.Printf("getBestBlock,chain:%d,index:%d,key:%x,i:%d,hp:%d\n", chain, index, b, i, rel.HashPower)
		if !rel.PreExist {
			data := core.ReadBlockData(chain, key)
			if data == nil {
				log.Printf("not exist block data.chain:%d,key:%x\n", chain, key)
				setBlockToIDBlocks(chain, index, b, 0)
				info := &messages.ReqBlock{Chain: chain, Index: index, Key: key}
				network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: info})
				continue
			}
			block := core.DecodeBlock(data)
			if block == nil {
				log.Printf("fail to DecodeBlock.chain:%d,key:%x\n", chain, key)
				setBlockToIDBlocks(chain, index, b, 0)
				continue
			}

			rel = block.GetReliability()

			var lost bool
			for _, t := range block.TransList {
				if core.IsExistTransaction(chain, t[:]) {
					continue
				}
				info := &messages.ReqTransaction{Chain: chain, Key: t[:]}
				network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: info})
				lost = true
				rel.HashPower--
				rel.PreExist = false
			}

			core.SaveBlockReliability(chain, key, rel)

			if lost {
				continue
			}
		}

		// time.Second
		if rel.Time > uint64(now+5)*1000 {
			continue
		}

		stat := core.ReadBlockRunStat(chain, key)
		if stat.RunTimes-stat.RunSuccessCount > 3 ||
			stat.RunTimes > 8 {
			log.Printf("delete idBlocks.chain:%d,index:%d,key:%x,rollback:%d,runTimes:%d,success:%d\n",
				chain, index, b, stat.RollbackCount,
				stat.RunTimes, stat.RunSuccessCount)
			setBlockToIDBlocks(chain, index, b, 0)
			core.SaveBlockRunStat(chain, rel.Key[:], core.BlockRunStat{})
			continue
		}
		hp := rel.HashPower
		hp -= 10 * uint64(stat.RunTimes-stat.RunSuccessCount)

		if rel.HashPower > ib.HashPower[i] {
			setBlockToIDBlocks(chain, index, b, rel.HashPower)
		}

		ch := core.GetChainHeight(chain, key)

		log.Printf("   rollback:%d,runTimes:%d,success:%d,height:%d,rhp:%d\n", stat.RollbackCount,
			stat.RunTimes, stat.RunSuccessCount, ch.Height, ch.HashPower)
		if ch.Height > 3 {
			hp += (ch.Height - 3)
			hp += ch.HashPower / 100
		}

		rel.HashPower = hp

		if rel.Cmp(relia) > 0 {
			relia = rel
		}
	}
	log.Printf("getBestBlock rst,num:%d,chain:%d,index:%d,hp:%d,key:%x\n", num, chain, index, relia.HashPower, relia.Key)
	return relia
}

//
func updateChainHeight(chain, from, num uint64) {
	for i := uint64(0); i < num; i++ {
		if from-i == 0 {
			return
		}
		ib := core.ReadIDBlocks(chain, from-i)
		for _, b := range ib.Blocks {
			if b.Empty() {
				continue
			}
			rel := core.ReadBlockReliability(chain, b[:])
			if rel.Previous.Empty() {
				continue
			}
			ch := core.GetChainHeight(chain, b[:])
			ch.Height++
			ch.HashPower += getHashPower(b[:])

			core.SaveChainHeight(chain, rel.Previous[:], ch)
		}
	}
}

func processEvent(chain uint64) {
	if chain == 0 {
		return
	}
	if network == nil {
		return
	}

	defer func() {
		e := recover()
		if e != nil {
			log.Println("something error,", e)
			log.Println(string(debug.Stack()))
		}
	}()

	procMgr.mu.Lock()
	wait, ok := procMgr.wait[chain]
	if !ok {
		procMgr.wait[chain] = make(chan int, 2)
		wait = procMgr.wait[chain]
	}
	cl, ok := procMgr.Chains[chain]
	if !ok {
		procMgr.Chains[chain] = make(chan int, 1)
		cl = procMgr.Chains[chain]
	}
	procMgr.mu.Unlock()

	select {
	case wait <- 1:
	default:
		return
	}
	defer func() { <-wait }()

	cl <- 1
	log.Println("start processEvent")
	defer func() {
		log.Println("finish processEvent")
		<-cl
	}()

	index := core.GetLastBlockIndex(chain)
	if index == 0 {
		// first block
		c := conf.GetConf()
		if chain > 1 {
			cInfo := core.GetChainInfo(chain / 2)
			if cInfo.LeftChildID == 1 || cInfo.RightChildID == 1 {
				data := core.ReadTransactionData(chain/2, c.FirstTransName)
				core.WriteTransaction(chain, data)
				ib := core.ReadIDBlocks(chain/2, 1)
				data = core.ReadBlockData(chain/2, ib.Blocks[0][:])
				core.WriteTransaction(chain, data)
			} else {
				return
			}
		}
		if !core.IsExistTransaction(chain, c.FirstTransName) {
			info := messages.ReqBlockInfo{}
			info.Chain = chain
			info.Index = 1
			network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: &info})
			return
		}

		core.CreateBiosTrans(chain)
	}

	// get the last index(processed)
	ek := core.Hash{}
	er := core.ReadBlockReliability(chain, ek[:])

	updateChainHeight(chain, index+13, 20)

	var relia core.TReliability
	now := time.Now().Unix()
	//check the last 6 block,if exist better block,rollback
	for i := er.Index - 6; i < er.Index; i++ {
		if i > index {
			continue
		}
		relia = getBestBlock(chain, i)
		key := core.GetTheBlockKey(chain, i)
		if relia.Key.Empty() || bytes.Compare(key, relia.Key[:]) == 0 {
			continue
		}
		log.Printf("processEvent,dbRollBack %d. index:%d,key:%x,relia:%x\n", i, index, key, relia.Key)

		dbRollBack(chain, index, key)
		go processEvent(chain)
		return
	}
	log.Printf("try to get next block key,chain:%d,index:%d\n", chain, index+1)

	relia = getBestBlock(chain, index+1)
	if relia.Key.Empty() {
		log.Printf("no next block key,chain:%d,index:%d\n", chain, index+1)
		t := core.GetBlockTime(chain)
		t += 200000
		if t >= uint64(now)*1000 {
			go doMine(chain)
			return
		}
		info := messages.ReqBlockInfo{Chain: chain, Index: index + 1}
		network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})

		if t+2000000 < uint64(now)*1000 {
			info := messages.ReqBlockInfo{Chain: chain, Index: index + 30}
			network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
		}
		return
	}
	preKey := core.GetTheBlockKey(chain, index)
	if bytes.Compare(relia.Previous[:], preKey) != 0 {
		log.Printf("dbRollBack one block. index:%d,key:%x,next block:%x\n", index, preKey, relia.Key)

		dbRollBack(chain, index, preKey)
		go processEvent(chain)

		t := core.GetBlockTime(chain)
		if t+200000 > uint64(now)*1000 {
			return
		}

		// too long,clear IDBlocks
		if t+2000000 < uint64(now)*1000 {
			ib := core.IDBlocks{}
			core.SaveIDBlocks(chain, index, ib)
			core.SaveIDBlocks(chain, index+1, ib)
		}
		info1 := messages.ReqBlockInfo{Chain: chain, Index: index + 10}
		network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info1})
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

	// save the last index
	if er.Index < index+1 {
		er.Index = index + 1
		er.Time = uint64(time.Now().Unix())
		core.SaveBlockReliability(chain, ek[:], er)
	}

	info := messages.BlockInfo{}
	info.Chain = chain
	info.Index = index + 1
	info.Key = relia.Key[:]
	network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
	info1 := messages.ReqBlockInfo{Chain: chain, Index: index + 2}
	network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: &info1})

	if relia.Time+200000 > uint64(now)*1000 {
		go doMine(chain)
	}

	cInfo := core.GetChainInfo(chain)
	if cInfo.LeftChildID == 1 {
		go processEvent(chain * 2)
	} else if cInfo.RightChildID == 1 {
		go processEvent(chain*2 + 1)
	}

	ib := core.IDBlocks{}
	core.SaveIDBlocks(chain, index-10, ib)

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

	transList, size := getTransListForMine(chain)

	block.SetTransList(transList)
	block.Size = uint32(size)
	block.Nonce = rand.Uint64()
	var key core.Hash
	var oldHP uint64
	to := time.Now().Unix()

	for {
		now := time.Now().Unix()
		if to+60 < now {
			break
		}
		if block.Time < uint64(now)*1000 && oldHP > 0 {
			// log.Printf("block.time(%d) > now(%d)\n", block.Time, now)
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

	if oldHP == 0 || key.Empty() {
		log.Printf("fail to doMine,error oldHP")
		return
	}

	log.Printf("mine one blok,chain:%d,index:%d,hashpower:%d,hp limit:%d,key:%x\n",
		chain, block.Index, oldHP, block.HashpowerLimit, key[:])
	info := messages.BlockInfo{}
	info.Chain = chain
	info.Index = block.Index
	info.Key = key[:]

	network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
	// go doMine(chain)
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

// hp=0,delete;hp>1,add and update; hp=1,add
func setBlockToIDBlocks(chain, index uint64, key core.Hash, hp uint64) {
	ib := core.ReadIDBlocks(chain, index)
	for i, b := range ib.Blocks {
		if key == b {
			if hp > ib.HashPower[i] {
				ib.HashPower[i] = hp
			} else if hp == 0 {
				ib.HashPower[i] = 0
				ib.Blocks[i] = core.Hash{}
			}
			hp = 0
			continue
		}
		if hp > ib.HashPower[i] {
			// log.Printf("IDBlocks switch,index:%d,i:%d,old:%x,new:%x\n", block.Index, i, b, key)
			hp, ib.HashPower[i] = ib.HashPower[i], hp
			key, ib.Blocks[i] = b, key
		}
	}
	core.SaveIDBlocks(chain, index, ib)
}
