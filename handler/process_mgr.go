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
	processEvent(1)
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
		if stat.RunTimes-stat.RunSuccessCount > 3 ||
			stat.RunTimes > 8 {
			log.Printf("delete idBlocks.chain:%d,index:%d,key:%x,rollback:%d,runTimes:%d,success:%d\n",
				chain, index, key, stat.RollbackCount,
				stat.RunTimes, stat.RunSuccessCount)
			setBlockToIDBlocks(chain, index, it.Key, 0)
			continue
		}
		hp := rel.HashPower
		hp -= stat.SelectedCount / 5
		hp -= uint64(stat.RunTimes) / 10

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

	t1 := core.GetBlockTime(chain)
	t0 := core.GetBlockTime(chain)
	if t0 > 0 {
		if t1 > t0+blockSyncTime {
			log.Printf("blockSyncTime,process parent.chain:%d\n", chain)
			go processEvent(chain / 2)
			return
		} else if t1 > t0 {
			go processEvent(chain / 2)
		}
	}
	t2 := core.GetBlockTime(2 * chain)
	if t2 > 0 {
		if t1 > t2+blockSyncTime {
			log.Printf("blockSyncTime,process leftchild.chain:%d\n", chain)
			go processEvent(2 * chain)
			return
		} else if t1 > t2 {
			go processEvent(2 * chain)
		}
	}
	t3 := core.GetBlockTime(2*chain + 1)
	if t3 > 0 {
		if t1 > t3+blockSyncTime {
			log.Printf("blockSyncTime,process rightchild.chain:%d\n", chain)
			go processEvent(2*chain + 1)
			return
		} else if t1 > t3 {
			go processEvent(2*chain + 1)
		}
	}

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
				ib := ReadIDBlocks(chain/2, 1)
				it := ib.Items[0]
				data = core.ReadBlockData(chain/2, it.Key[:])
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
	relia.Ready = true

	core.SaveBlockReliability(chain, relia.Key[:], relia)
	SaveBlockRunStat(chain, relia.Key[:], stat)

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
	info.HashPower = relia.HashPower
	network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})

	cInfo := core.GetChainInfo(chain)
	if cInfo.LeftChildID == 1 {
		go writeFirstBlockToChain(chain * 2)
	} else if cInfo.RightChildID == 1 {
		go writeFirstBlockToChain(chain*2 + 1)
	}

	go processEvent(chain)
}

func writeFirstBlockToChain(chain uint64) {
	if chain <= 1 {
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
