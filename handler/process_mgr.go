package handler

import (
	"bytes"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/messages"
)

// chain->index->blockKey->reliability
type tProcessMgr struct {
	mu       sync.Mutex
	wait     map[uint64]chan int
	Chains   map[uint64]chan int
	mineLock chan int
}

const (
	tSecond         = 1000
	tMinute         = 60 * 1000
	tHour           = 60 * tMinute
	tDat            = 24 * tHour
	blockAcceptTime = tMinute
	transAcceptTime = 9 * tDat
	blockSyncTime   = 5 * tMinute
)

var procMgr tProcessMgr

func init() {
	procMgr.wait = make(map[uint64]chan int)
	procMgr.Chains = make(map[uint64]chan int)
	procMgr.mineLock = make(chan int, 5)

	time.AfterFunc(time.Second*5, timeoutFunc)
}

func timeoutFunc() {
	time.AfterFunc(time.Second*20, timeoutFunc)
	processChains(1)
}

func processChains(chain uint64) {
	id := core.GetLastBlockIndex(chain)
	if id == 0 {
		return
	}
	go processEvent(chain)
	processChains(2 * chain)
	processChains(2*chain + 1)
}

func getBestBlock(chain, index uint64) core.TReliability {
	var relia, rel core.TReliability
	var num int
	ib := ReadIDBlocks(chain, index)
	now := time.Now().Unix() * 1000
	if len(ib.Items) == 1 {
		it := ib.Items[0]
		log.Printf("getBestBlock rst,num:1,chain:%d,index:%d,key:%x,hp:%d\n", chain, index, it.Key, it.HashPower)
		return core.ReadBlockReliability(chain, it.Key[:])
	}
	for i, it := range ib.Items {
		key := it.Key[:]
		rel = core.ReadBlockReliability(chain, key)

		// time.Second
		if rel.Time > uint64(now) {
			continue
		}

		stat := ReadBlockRunStat(chain, key)
		hp := rel.HashPower
		hp -= stat.SelectedCount / 5
		hp -= uint64(stat.RunTimes) / 10
		if hp == 0 {
			setBlockToIDBlocks(chain, index, it.Key, 0)
			core.DeleteBlock(chain, it.Key[:])
			stat = BlockRunStat{}
			SaveBlockRunStat(chain, it.Key[:], stat)
			rel.HashPower = 0
			core.SaveBlockReliability(chain, rel.Key[:], rel)
			continue
		}

		if rel.HashPower > it.HashPower {
			setBlockToIDBlocks(chain, index, it.Key, rel.HashPower)
		}

		log.Printf("getBestBlock,chain:%d,index:%d,key:%x,i:%d,hp:%d,"+
			"rollback:%d,runTimes:%d,success:%d,selected:%d\n",
			chain, index, key, i, rel.HashPower, stat.RollbackCount,
			stat.RunTimes, stat.RunSuccessCount, stat.SelectedCount)

		rel.HashPower = hp

		if rel.Cmp(relia) > 0 {
			relia = rel
		}
	}
	log.Printf("getBestBlock rst,num:%d,chain:%d,index:%d,hp:%d,key:%x\n", num, chain, index, relia.HashPower, relia.Key)
	return relia
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
		if !core.IsExistTransaction(chain, c.FirstTransName) {
			return
		}
		core.CreateBiosTrans(chain)
	}

	// check child chain
	{
		cInfo := core.GetChainInfo(chain)
		if chain > 1 && index < 100 {
			// parent chain rollback,the block(new chain) is not exist
			pk := core.GetParentBlockOfChain(chain)
			k := core.GetTheBlockKey(chain/2, cInfo.ParentID)
			if bytes.Compare(k, pk[:]) != 0 {
				log.Printf("rollback chain:%d,index:%d,hope:%x,get:%x\n", chain, index, pk, k)
				lk := core.GetTheBlockKey(chain, 1)
				dbRollBack(chain, 1, lk)
				return
			}
		}
		if cInfo.LeftChildID == 1 {
			go writeFirstBlockToChain(chain * 2)
		} else if cInfo.RightChildID == 1 {
			go writeFirstBlockToChain(chain*2 + 1)
		}

		// chain1.Time-chain2.Time > blockSyncTime, stop
		t1 := core.GetBlockTime(chain)
		if cInfo.ParentID > 1 {
			t0 := core.GetBlockTime(chain / 2)
			if t1 > t0+blockSyncTime {
				log.Printf("wait chain:%d,index:%d. parent.Time:%d,self.Time:%d\n", chain, index, t0, t1)
				return
			}
		}
		if cInfo.LeftChildID > 1 {
			t2 := core.GetBlockTime(chain * 2)
			if t1 > t2+blockSyncTime {
				log.Printf("wait chain:%d,index:%d. leftChild.Time:%d,self.Time:%d\n", chain, index, t2, t1)
				return
			}
		}
		if cInfo.RightChildID > 1 {
			t3 := core.GetBlockTime(chain*2 + 1)
			if t1 > t3+blockSyncTime {
				log.Printf("wait chain:%d,index:%d. rightChild.Time:%d,self.Time:%d\n", chain, index, t3, t1)
				return
			}
		}
	}

	var relia core.TReliability
	now := uint64(time.Now().Unix() * 1000)

	relia = getBestBlock(chain, index+1)
	if relia.Key.Empty() {
		log.Printf("no next block key,chain:%d,index:%d\n", chain, index+1)
		relia = getBestBlock(chain, index)
		nowKey := core.GetTheBlockKey(chain, index)
		if !relia.Key.Empty() && bytes.Compare(relia.Key[:], nowKey) != 0 {
			log.Printf("dbRollBack block. index:%d,key:%x,next block:%x\n", index, nowKey, relia.Key)
			dbRollBack(chain, index, nowKey)
			go processEvent(chain)
			return
		}
		relia = getBestBlock(chain, index+2)
		if !relia.Previous.Empty() {
			setBlockToIDBlocks(chain, relia.Index-1, relia.Previous, core.BaseRelia)
		}
		t := core.GetBlockTime(chain)
		go doMine(chain, false)

		if t+core.GetBlockInterval(chain) >= now {
			return
		}

		info := messages.ReqBlockInfo{Chain: chain, Index: index + 1}
		network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
		return
	}
	stat := ReadBlockRunStat(chain, relia.Key[:])
	stat.SelectedCount++
	preKey := core.GetTheBlockKey(chain, index)
	if bytes.Compare(relia.Previous[:], preKey) != 0 {
		log.Printf("dbRollBack block. index:%d,key:%x,next block:%x\n", index, preKey, relia.Key)
		ib := IDBlocks{}
		it := ItemBlock{relia.Previous, 1}
		ib.Items = append(ib.Items, it)
		SaveIDBlocks(chain, index, ib)
		SaveBlockRunStat(chain, relia.Key[:], stat)
		dbRollBack(chain, index, preKey)
		go processEvent(chain)
		return
	}
	stat.RunTimes++
	log.Printf("start to process block,chain:%d,index:%d,key:%x\n", chain, relia.Index, relia.Key)
	err := blockRun(chain, relia.Key[:])
	if err != nil {
		log.Printf("fail to process block,chain:%d,index:%d,key:%x,error:%s\n", chain, index+1, relia.Key, err)
		SaveBlockRunStat(chain, relia.Key[:], stat)
		setBlockToIDBlocks(chain, relia.Index, relia.Key, 0)
		return
	}
	stat.RunSuccessCount++

	SaveBlockRunStat(chain, relia.Key[:], stat)

	info := messages.BlockInfo{}
	info.Chain = chain
	info.Index = index + 1
	info.Key = relia.Key[:]
	info.HashPower = relia.HashPower
	network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})

	go processEvent(chain)
}

func writeFirstBlockToChain(chain uint64) {
	if chain <= 1 {
		return
	}
	id := core.GetLastBlockIndex(chain)
	if id > 0 {
		return
	}
	c := conf.GetConf()
	data := core.ReadTransactionData(1, c.FirstTransName)
	core.WriteTransaction(chain, data)
	key := core.GetTheBlockKey(1, 1)
	data = core.ReadBlockData(1, key)
	processBlock(chain, key, data)
	rel := core.ReadBlockReliability(chain, key)
	setBlockToIDBlocks(chain, rel.Index, rel.Key, rel.HashPower)
	log.Println("new chain:", chain)
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

// hp=0,delete;hp>1,add and update; hp=1,add
func setBlockToIDBlocks(chain, index uint64, key core.Hash, hp uint64) {
	if key.Empty() {
		return
	}
	ib := ReadIDBlocks(chain, index)
	var newIB IDBlocks
	for _, it := range ib.Items {
		if key == it.Key {
			if hp > it.HashPower {
				it.HashPower = hp
			}
			if hp > 0 {
				newIB.Items = append(newIB.Items, it)
			}
			hp = 0
			continue
		}
		if hp > it.HashPower {
			// log.Printf("IDBlocks switch,index:%d,i:%d,old:%x,new:%x\n", block.Index, i, b, key)
			nit := ItemBlock{Key: key, HashPower: hp}
			newIB.Items = append(newIB.Items, nit)
			newIB.Items = append(newIB.Items, it)
			hp = 0
		} else {
			newIB.Items = append(newIB.Items, it)
		}
	}
	if hp > 0 {
		nit := ItemBlock{Key: key, HashPower: hp}
		newIB.Items = append(newIB.Items, nit)
	}
	if len(newIB.Items) > core.MinerNum {
		newIB.Items = newIB.Items[:core.MinerNum]
	}
	newIB.MaxHeight = ib.MaxHeight
	SaveIDBlocks(chain, index, newIB)
}
