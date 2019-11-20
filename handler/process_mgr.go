package handler

import (
	"bytes"
	"errors"
	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/govm/runtime"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// chain->index->blockKey->reliability
type tProcessMgr struct {
	mu       sync.Mutex
	wait     map[uint64]chan int
	Chains   map[uint64]chan int
	mineLock chan int
}

const (
	tSecond           = 1000
	tMinute           = 60 * 1000
	tHour             = 60 * tMinute
	tDay              = 24 * tHour
	blockAcceptTime   = 2 * tMinute
	transAcceptTime   = 9 * tDay
	blockSyncTime     = 5 * tMinute
	hpAcceptRange     = 20
	blockLockInterval = 6
	lockBySelfChain   = 1
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
	var relia core.TReliability
	var num int
	ib := ReadIDBlocks(chain, index)
	now := time.Now().Unix() * 1000
	for i, it := range ib.Items {
		key := it.Key[:]
		rel := core.ReadBlockReliability(chain, key)

		// time.Second
		if rel.Time > uint64(now) {
			continue
		}
		if it.HashPower > rel.HashPower {
			rel.Recalculation(chain)
			core.SaveBlockReliability(chain, rel.Key[:], rel)
		}

		stat := ReadBlockRunStat(chain, key)
		hp := rel.HashPower
		hp -= stat.SelectedCount / 5
		hp -= uint64(stat.RunTimes) / 10
		hp -= uint64(stat.RunTimes - stat.RunSuccessCount)
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

func checkOtherChain(chain uint64) error {
	index := core.GetLastBlockIndex(chain)
	cInfo := core.GetChainInfo(chain)
	if chain > 1 && index > 0 && index < 100 {
		// parent chain rollback,the block(new chain) is not exist
		pk := core.GetParentBlockOfChain(chain)
		if !pk.Empty() && !core.BlockOnTheChain(chain/2, pk[:]) {
			log.Printf("rollback chain:%d,index:%d,hope:%x on parent chain\n", chain, index, pk)
			var i uint64
			ib := IDBlocks{}
			for i = 2; i < index+1; i++ {
				SaveIDBlocks(chain, i, ib)
			}
			lk := core.GetTheBlockKey(chain, 1)
			dbRollBack(chain, 1, lk)
			return errors.New("different parent")
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
			go processEvent(chain / 2)
			return errors.New("wait parent chain")
		}
	}
	if cInfo.LeftChildID > 1 {
		t2 := core.GetBlockTime(chain * 2)
		if t1 > t2+blockSyncTime {
			log.Printf("wait chain:%d,index:%d. leftChild.Time:%d,self.Time:%d\n", chain, index, t2, t1)
			go processEvent(chain * 2)
			return errors.New("wait LeftChild chain")
		}
	}
	if cInfo.RightChildID > 1 {
		t3 := core.GetBlockTime(chain*2 + 1)
		if t1 > t3+blockSyncTime {
			log.Printf("wait chain:%d,index:%d. rightChild.Time:%d,self.Time:%d\n", chain, index, t3, t1)
			go processEvent(chain*2 + 1)
			return errors.New("wait RightChild chain")
		}
	}

	return nil
}

func beforeProcBlock(chain uint64, rel core.TReliability) error {
	preKey := core.GetTheBlockKey(chain, rel.Index-1)
	if bytes.Compare(rel.Previous[:], preKey) == 0 {
		return nil
	}
	t := core.GetBlockTime(chain)
	now := uint64(time.Now().Unix() * 1000)
	if t+tMinute*2 > now {
		// Need rollback,it could not be the newest block. Prevention of attacks
		return errors.New("need rollback,but block too new")
	}
	if rel.Time < t+core.GetBlockInterval(chain)/2 {
		setBlockToIDBlocks(chain, rel.Index, rel.Key, 0)
		return errors.New("error block time")
	}
	rst := ldb.LGet(chain, ldbBlockLock, preKey)
	if len(rst) > 0 {
		log.Printf("the pre_block is locked,unable rollback,chain:%d,pre key:%x,key:%x\n", chain, preKey, rel.Key)
		ib := IDBlocks{}
		for i := rel.Index; i < rel.Index+30; i++ {
			SaveIDBlocks(chain, i, ib)
		}
		// lock by other chain,rst is block id
		if len(rst) > 1 {
			rel := core.ReadBlockReliability(chain, rel.Previous[:])
			rel.HashPower = core.BaseRelia
			rel.Previous = core.Hash{}
			core.SaveBlockReliability(chain, rel.Key[:], rel)
			core.DeleteBlock(chain, rel.Previous[:])
			core.DeleteBlock(chain, rel.Key[:])
			return errors.New("the block is locked by other chain")
		}
		ldb.LSet(chain, ldbBlockLock, preKey, nil)
		info := messages.ReqBlockInfo{Chain: chain, Index: rel.Index}
		network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: &info})
		network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: &info})
		return errors.New("locked,ReqBlockInfo")
	}
	log.Printf("dbRollBack block. index:%d,key:%x,next block:%x\n", rel.Index, preKey, rel.Key)
	ib := IDBlocks{}
	it := ItemBlock{rel.Previous, core.BaseRelia}
	ib.Items = append(ib.Items, it)
	SaveIDBlocks(chain, rel.Index-1, ib)
	dbRollBack(chain, rel.Index-1, preKey)
	go processEvent(chain)
	// return

	return errors.New("rollback")
}

func successToProcBlock(chain uint64, rel core.TReliability) error {
	ib := ReadIDBlocks(chain, rel.Index-blockLockInterval)
	for _, it := range ib.Items {
		if !core.BlockOnTheChain(chain, it.Key[:]) {
			// log.Printf("delete block,chain:%d,key:%x\n", chain, it.Key)
			core.DeleteBlock(chain, it.Key[:])
			core.DeleteBlockReliability(chain, it.Key[:])
		} else {
			rst := ldb.LGet(chain, ldbBlockLock, it.Key[:])
			if len(rst) == 0 {
				ldb.LSet(chain, ldbBlockLock, it.Key[:], []byte{lockBySelfChain})
			}
		}
	}
	if rel.Index > 2 && !rel.Parent.Empty() {
		ldb.LSet(chain/2, ldbBlockLock, rel.Parent[:], rel.Key[:])
	}
	if !rel.LeftChild.Empty() {
		ldb.LSet(chain*2, ldbBlockLock, rel.LeftChild[:], rel.Key[:])
	}
	if !rel.RightChild.Empty() {
		ldb.LSet(chain*2+1, ldbBlockLock, rel.RightChild[:], rel.Key[:])
	}

	old := rel.HashPower
	rel.Recalculation(chain)
	ldb.LSet(chain, ldbHPLimit, runtime.Encode(rel.Index), runtime.Encode(rel.HashPower))
	if old != rel.HashPower {
		core.SaveBlockReliability(chain, rel.Key[:], rel)
	}
	return nil
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
	log.Println("start processEvent:", chain)
	defer func() {
		log.Println("finish processEvent:", chain)
		<-cl
	}()

	index := core.GetLastBlockIndex(chain)
	if index == 0 {
		if chain > 1 {
			return
		}
		// first block
		c := conf.GetConf()
		if !core.IsExistTransaction(chain, c.FirstTransName) {
			return
		}
		core.CreateBiosTrans(chain)
	}

	// check child chain
	err := checkOtherChain(chain)
	if err != nil {
		log.Printf("checkOtherChain,chain:%d,err:%s\n", chain, err)
		return
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
		t := core.GetBlockTime(chain)
		go doMine(chain, false)

		if t+core.GetBlockInterval(chain) >= now {
			return
		}

		info := messages.ReqBlockInfo{Chain: chain, Index: index + 1}
		network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: &info})
		return
	}
	stat := ReadBlockRunStat(chain, relia.Key[:])
	stat.SelectedCount++
	//
	err = beforeProcBlock(chain, relia)
	if err != nil {
		log.Println("beforeProcBlock,chain:", chain, err)
		SaveBlockRunStat(chain, relia.Key[:], stat)
		return
	}
	stat.RunTimes++
	log.Printf("start to process block,chain:%d,index:%d,key:%x\n", chain, relia.Index, relia.Key)
	err = core.ProcessBlockOfChain(chain, relia.Key[:])
	if err != nil {
		log.Printf("fail to process block,chain:%d,index:%d,key:%x,error:%s\n", chain, index+1, relia.Key, err)
		SaveBlockRunStat(chain, relia.Key[:], stat)
		setBlockToIDBlocks(chain, relia.Index, relia.Key, 0)
		//delete transaction
		transList := GetTransList(chain, relia.Key[:])
		for _, k := range transList {
			deleteTransInfo(chain, k[:])
			tInfo := readTransInfo(chain, k[:])
			saveBlackItem(chain, tInfo.User[:])
		}
		return
	}
	stat.RunSuccessCount++
	SaveBlockRunStat(chain, relia.Key[:], stat)
	// delete older block,relia.Index-6
	successToProcBlock(chain, relia)

	if relia.Time+blockSyncTime < now {
		go processEvent(chain)
		return
	}
	autoRegisterMiner(chain)

	info := messages.BlockInfo{}
	info.Chain = chain
	info.Index = relia.Index
	info.Key = relia.Key[:]
	info.HashPower = relia.HashPower
	info.PreKey = relia.Previous[:]
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

func reliaRecalculation(chain uint64) {
	var index uint64
	var preKey []byte
	defer recover()
	for index = 1; ; index++ {
		key := core.GetTheBlockKey(chain, index)
		if len(key) == 0 {
			break
		}
		rel := core.ReadBlockReliability(chain, key)
		if rel.Index != index {
			rel.Index = index
		}
		if index > 1 && bytes.Compare(rel.Previous[:], preKey) != 0 {
			log.Printf("error Previous of Recalculation,chain:%d,index:%d,hope:%x,get:%x\n",
				chain, index, preKey, rel.Previous)
			data := core.ReadBlockData(chain, key)
			processBlock(chain, key, data)
			continue
		}
		preKey = key
		old := rel.HashPower
		rel.Recalculation(chain)
		if rel.HashPower != old {
			log.Printf("rel.Recalculation chain:%d,index:%d,new hp:%d,old hp:%d\n", chain, index, rel.HashPower, old)
			core.SaveBlockReliability(chain, key, rel)
		}
	}
	if index == 1 {
		log.Println("finish reliaRecalculation(not block):", chain)
		return
	}
	ib := ReadIDBlocks(chain, index)
	for _, it := range ib.Items {
		rel := core.ReadBlockReliability(chain, it.Key[:])
		old := rel.HashPower
		rel.Recalculation(chain)
		if rel.HashPower != old {
			log.Printf("IDBlocks.Recalculation chain:%d,index:%d,new hp:%d,old hp:%d\n", chain, index, rel.HashPower, old)
			core.SaveBlockReliability(chain, it.Key[:], rel)
		}
	}

	reliaRecalculation(2 * chain)
	reliaRecalculation(2*chain + 1)
	log.Println("finish reliaRecalculation:", chain, index)
}
