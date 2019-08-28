package handler

import (
	"bytes"
	"github.com/lengzhao/govm/event"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/depend"
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

	first := depend.First(chain)
	if first == nil {
		//log.Println("fail to get first depend:", chain)
		return
	}
	element := first.(*depend.DfElement)
	log.Printf("start to proxy,key:%x,isBlock:%t,timeout:%d\n", element.Key, element.IsBlock, element.Timeout)

	if element.Timeout+30 < time.Now().Unix() {
		depend.Delete(chain, first.GetKey())
	}

	if !element.IsBlock {
		return
	}

	key := element.Key
	index := core.GetLastBlockIndex(chain)
	procKey := core.GetTheBlockKey(chain, index)
	if bytes.Compare(key, procKey) == 0 {
		depend.Delete(chain, first.GetKey())
		go processEvent(chain)
		return
	}

	oldRel := core.GetBlockReliability(chain, key)
	if len(oldRel.Reliability) == 1 && oldRel.Reliability[0] > 3 && oldRel.Index <= index {
		log.Printf("the reliability of block > 3.chain:%d,key:%x,index:%d,last index:%d\n",
			chain, key, oldRel.Index, index)
		depend.Delete(chain, first.GetKey())
		go processEvent(chain)
		return
	}

	data := core.ReadBlockData(chain, key)
	if data == nil {
		log.Printf("not exist block data.chain:%d,key:%x\n", chain, key)
		download(chain, first.GetKey(), nil, &messages.ReqBlock{Chain: chain, Key: key})
		return
	}
	block := core.DecodeBlock(data)
	if block == nil {
		depend.Pop(chain)
		return
	}

	if block.Time > uint64(time.Now().Unix()+5)*1000 {
		return
	}

	if block.Index > 1 {
		rel := core.GetBlockReliability(chain, block.Previous[:])
		// 前一个还没处理，优先处理前一个
		if len(rel.Reliability) == 0 {
			depend.Insert(chain, &depend.DfElement{Key: block.Key[:], IsBlock: true},
				&depend.DfElement{Key: block.Previous[:], IsBlock: true})
			go processEvent(chain)
			return
		} else if len(rel.Reliability) == 0 && rel.Reliability[0] < 100 {
			log.Printf("previous is error,reliability:%x,self index:%d,id of err block:%d,key of block:%x\n",
				rel.Reliability, block.Index, rel.Index, rel.Key)
			core.SaveBlockReliability(chain, block.Key[:], rel)
			depend.Delete(chain, first.GetKey())
			return
		}
		preKey := core.GetTheBlockKey(chain, block.Index-1)
		if len(preKey) == 0 {
			depend.Insert(chain, &depend.DfElement{Key: block.Key[:], IsBlock: true},
				&depend.DfElement{Key: block.Previous[:], IsBlock: true})
			go processEvent(chain)
			return
		}
		// 前一个block不一样，忽略
		if bytes.Compare(block.Previous[:], preKey) != 0 {
			rel.Reliability = []byte{1}
			core.SaveBlockReliability(chain, key, rel)
			depend.Delete(chain, first.GetKey())
			return
		}
	} else if block.Index == 1 {
		if chain > 1 {
			if !core.ChainIsCreated(chain) {
				return
			}
		}
		c := conf.GetConf()
		if !core.IsExistTransaction(chain, c.FirstTransName) {
			depend.Insert(chain, &depend.DfElement{Key: block.Key[:], IsBlock: true}, &depend.DfElement{Key: c.FirstTransName})
			return
		}

		core.CreateBiosTrans(chain)
	}

	defer func() {
		depend.Delete(chain, first.GetKey())

		autoRegisterMiner(chain)

		t := core.GetBlockTime(chain)
		t += uint64(5 * 60 * 1000)
		if t/1000 < uint64(time.Now().Unix()) {
			return
		}

		c := conf.GetConf()
		lastIndex := core.GetLastBlockIndex(chain)
		if c.ForceMine || core.IsMiner(chain, lastIndex+1, c.WalletAddr) {
			go doMine(chain)
		}
	}()

	rel := block.GetReliability()
	core.SaveBlockReliability(chain, key, rel)

	lKey := core.GetTheBlockKey(chain, block.Index)
	if lKey != nil {
		lRel := core.GetBlockReliability(chain, lKey)
		if rel.Cmp(lRel) <= 0 {
			return
		}
		if block.Index+6 < index {
			log.Printf("block too old,chain:%d,index:%d,key:%x, exist index:%d\n", chain, block.Index, block.Key[:], index)
			return
		}
		//rollback
		err := dbRollBack(chain, block.Index, block.Previous[:])
		if err != nil {
			log.Printf("fail to rollback:%d,%x,%s\n", chain, key, err)
			return
		}
	}

	err := blockRun(chain, key)
	if err != nil {
		if len(oldRel.Reliability) == 1 {
			oldRel.Reliability[0]++
		} else {
			oldRel.Reliability = []byte{1}
		}

		core.SaveBlockReliability(chain, key, oldRel)
		log.Printf("error block,chain:%d,index:%d,key:%x,error:%s\n", chain, block.Index, block.Key[:], err)
		// log.Println("block info:", block)
		return
	}

	cInfo := core.GetChainInfo(chain)
	if cInfo.LeftChildID == 1 {
		childChain := 2 * chain
		id := core.GetLastBlockIndex(childChain)
		if id == 0 {
			data, err := ioutil.ReadFile("first_trans.dat")
			if err == nil {
				core.WriteTransaction(childChain, data)
			}
			data, err = ioutil.ReadFile("first_block.dat")
			if err == nil {
				core.WriteBlock(childChain, data)
			}
			depend.Insert(childChain, &depend.DfElement{Key: data[:core.HashLen], IsBlock: true}, nil)
			go processEvent(childChain)
		}
	}
	if cInfo.RightChildID == 1 {
		childChain := 2*chain + 1
		id := core.GetLastBlockIndex(childChain)
		if id == 0 {
			data, err := ioutil.ReadFile("first_trans.dat")
			if err == nil {
				core.WriteTransaction(childChain, data)
			}
			data, err = ioutil.ReadFile("first_block.dat")
			if err == nil {
				core.WriteBlock(childChain, data)
			}
			depend.Insert(childChain, &depend.DfElement{Key: data[:core.HashLen], IsBlock: true}, nil)
			go processEvent(childChain)
		}
	}

	info := messages.BlockInfo{}
	info.Chain = chain
	info.Index = block.Index
	info.Key = block.Key[:]

	mgr.net.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
	// log.Println("SendInternalMsg BlockInfo:", info)

	cfg := conf.GetConf()
	if chain == cfg.ChainOfMine || cfg.ChainOfMine == 0 {
		for _, key := range block.TransList {
			delTrans(chain, key[:])
		}
	}
	// log.Printf("success process block,chain:%d,index:%d,key:%x\n", chain, block.Index, block.Key[:])

	go processEvent(chain)
}

func getHashPower(in []byte) uint8 {
	var out uint8
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

	transList := getTransListForMine(chain)
	size, lst := filterTrans(chain, transList)

	block.SetTransList(lst)
	block.Size = size
	var key core.Hash
	var oldHP uint8

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
		if hp <= block.PreHashpower {
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

	depend.Insert(chain, &depend.DfElement{Key: key[:], IsBlock: true},
		&depend.DfElement{Key: block.Previous[:], IsBlock: true})

	log.Printf("mine one blok,chain:%d,index:%d,hashpower:%d,hp limit:%d,key:%x\n",
		chain, block.Index, oldHP, block.HashPower, block.Key[:])
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
