package handler

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/libp2p"
)

// MsgPlugin process p2p message
type MsgPlugin struct {
	*libp2p.Plugin
	net libp2p.Network
}

const (
	reqBlock = byte(iota)
	reqTrans
	transOwner
	transOfBlock
)

type keyOfBlockHP struct {
	Chain uint64
	Index int64
}

var network libp2p.Network
var activeNode libp2p.Session
var blockHP *database.LRUCache

const blockHPNumber = 30

func init() {
	blockHP = database.NewLRUCache(100 * blockHPNumber)
}

// Startup is called only once when the plugin is loaded
func (p *MsgPlugin) Startup(n libp2p.Network) {
	p.net = n
	network = n
}

// Cleanup is called only once when the plugin is unload
func (p *MsgPlugin) Cleanup(n libp2p.Network) {
	procMgr.stop = true
}

var first = true

func getEnvKey(chain uint64, typ byte) string {
	return fmt.Sprintf("req_%x_%x", chain, typ)
}

// Receive receive message
func (p *MsgPlugin) Receive(ctx libp2p.Event) error {
	switch msg := ctx.GetMessage().(type) {
	case *messages.ReqBlockInfo:
		key := core.GetTheBlockKey(msg.Chain, msg.Index)
		if len(key) == 0 {
			// log.Println("fail to get the key,index:", msg.Index, ",chain:", msg.Chain)
			return nil
		}
		log.Printf("<%x> ReqBlockInfo %d %d\n", ctx.GetPeerID(), msg.Chain, msg.Index)
		rel := core.ReadBlockReliability(msg.Chain, key)
		resp := new(messages.BlockInfo)
		resp.Chain = msg.Chain
		resp.Index = msg.Index
		resp.Key = key
		resp.PreKey = rel.Previous[:]
		resp.HashPower = rel.HashPower
		ctx.Reply(resp)
		return nil
	case *messages.BlockInfo:
		hp := getHashPower(msg.Key)
		if hp < 5 || hp > 250 {
			return nil
		}

		index := core.GetLastBlockIndex(msg.Chain)
		if index == 0 && msg.Chain > 1 {
			return nil
		}
		if index > msg.Index {
			key := core.GetTheBlockKey(msg.Chain, index)
			if len(key) > 0 {
				rel := core.ReadBlockReliability(msg.Chain, key)
				ctx.Reply(&messages.BlockInfo{Chain: msg.Chain,
					Index: rel.Index, Key: key, PreKey: rel.Previous[:], HashPower: rel.HashPower})
			}
			return nil
		}

		preKey := core.GetTheBlockKey(msg.Chain, msg.Index-1)
		if bytes.Compare(preKey, msg.PreKey) != 0 {
			if msg.Index > index+1 && needRequstID(msg.Chain, index+1) {
				ctx.Reply(&messages.ReqBlockInfo{Chain: msg.Chain, Index: index + 1})
			}
			return nil
		}

		if core.IsExistBlock(msg.Chain, msg.Key) {
			log.Println("block exist:", msg.Index, ",chain:", msg.Chain, ",self:", index)
			rel := core.ReadBlockReliability(msg.Chain, msg.Key)
			if rel.HashPower == 0 {
				err := processBlock(msg.Chain, msg.Key, nil)
				if err != nil {
					core.DeleteBlock(msg.Chain, msg.Key)
					log.Printf("error hashpower of block,delete it.chain:%d,key:%x\n", msg.Chain, msg.Key)
					return nil
				}
			}
			if !rel.Ready {
				p.downloadBlockDepend(ctx, msg.Chain, msg.Key)
			} else {
				rel.Recalculation(msg.Chain)
				var hpLimit uint64
				getData(msg.Chain, ldbHPLimit, runtime.Encode(rel.Index-1), &hpLimit)
				if rel.HashPower+hpAcceptRange >= hpLimit {
					setBlockToIDBlocks(msg.Chain, rel.Index, rel.Key, rel.HashPower)
				}
				go processEvent(msg.Chain)
			}
			return nil
		}
		if needDownload(msg.Chain, msg.Key) {
			log.Printf("<%x> BlockKey %d %d,key:%x\n", ctx.GetPeerID(), msg.Chain, msg.Index, msg.Key)
			ctx.GetSession().SetEnv(getEnvKey(msg.Chain, reqBlock), hex.EncodeToString(msg.Key))
			ctx.Reply(&messages.ReqBlock{Chain: msg.Chain, Index: msg.Index, Key: msg.Key})
		}

		key := core.GetTheBlockKey(msg.Chain, 0)
		rel := core.ReadBlockReliability(msg.Chain, key)
		if msg.Index == rel.Index && msg.HashPower < rel.HashPower {
			ctx.Reply(&messages.BlockInfo{Chain: msg.Chain, Index: rel.Index,
				Key: key, HashPower: rel.HashPower, PreKey: rel.Previous[:]})
		}
	case *messages.TransactionInfo:
		if len(msg.Key) != core.HashLen {
			return nil
		}
		t := core.GetBlockTime(msg.Chain)
		if msg.Time > t+blockAcceptTime || msg.Time+transAcceptTime < t {
			return nil
		}
		if core.IsExistTransaction(msg.Chain, msg.Key) {
			return nil
		}
		if needDownload(msg.Chain, msg.Key) {
			log.Printf("<%x> TransactionInfo %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
			ctx.GetSession().SetEnv(getEnvKey(msg.Chain, reqTrans), hex.EncodeToString(msg.Key))
			ctx.Reply(&messages.ReqTransaction{Chain: msg.Chain, Key: msg.Key})
		}
	case *messages.ReqBlock:
		if len(msg.Key) == 0 {
			return nil
		}

		data := core.ReadBlockData(msg.Chain, msg.Key)
		if len(data) == 0 {
			log.Printf("not found.ReqBlock chain:%d index:%d key:%x\n", msg.Chain, msg.Index, msg.Key)
			return nil
		}
		log.Printf("<%x> ReqBlock %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		ctx.Reply(&messages.BlockData{Chain: msg.Chain, Key: msg.Key, Data: data})
	case *messages.ReqTransaction:
		data := core.ReadTransactionData(msg.Chain, msg.Key)
		if data == nil {
			log.Printf("not found the transaction,chain:%d,key:%x\n", msg.Chain, msg.Key)
			return nil
		}
		log.Printf("<%x> ReqTransaction %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		ctx.Reply(&messages.TransactionData{Chain: msg.Chain, Key: msg.Key, Data: data})
	case *messages.BlockData:
		if len(msg.Data) > 102400 {
			return nil
		}
		e := ctx.GetSession().GetEnv(getEnvKey(msg.Chain, reqBlock))
		k := hex.EncodeToString(msg.Key)
		if e != k {
			return nil
		}
		ctx.GetSession().SetEnv(getEnvKey(msg.Chain, reqBlock), "")
		err := processBlock(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			log.Printf("fail to processBlock,chain:%d,key:%x,err:%s\n", msg.Chain, msg.Key, err)
			return err
		}

		procMgr.mu.Lock()
		activeNode = ctx.GetSession()
		procMgr.mu.Unlock()

		log.Printf("<%x> BlockData %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		p.downloadBlockDepend(ctx, msg.Chain, msg.Key)

	case *messages.TransactionData:
		if len(msg.Key) != core.HashLen {
			return nil
		}
		e := ctx.GetSession().GetEnv(getEnvKey(msg.Chain, reqTrans))
		k := hex.EncodeToString(msg.Key)
		if e == k {
			ctx.GetSession().SetEnv(getEnvKey(msg.Chain, reqTrans), "")
			err := processTransaction(msg.Chain, msg.Key, msg.Data)
			if err != nil {
				return nil
			}
			head := readTransInfo(msg.Chain, msg.Key)
			if head.Size == 0 {
				return nil
			}
			info := messages.TransactionInfo{}
			info.Chain = msg.Chain
			info.Time = head.Time
			info.Key = msg.Key
			info.User = head.User[:]
			network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
			return nil
		}
		e = ctx.GetSession().GetEnv(getEnvKey(msg.Chain, transOfBlock))
		if e != k {
			return nil
		}

		ctx.GetSession().SetEnv(getEnvKey(msg.Chain, transOfBlock), "")
		err := processTransaction(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return err
		}
		e = ctx.GetSession().GetEnv(getEnvKey(msg.Chain, transOwner))
		if e == "" {
			return nil
		}

		bk, _ := hex.DecodeString(e)
		p.downloadBlockDepend(ctx, msg.Chain, bk)

	default:
		//log.Println("msg", ctx.GetPeerID(), msg)
		if first {
			first = false
			index := core.GetLastBlockIndex(1)
			index++
			if index == 1 {
				ctx.Reply(&messages.ReqTransaction{Chain: 1, Key: conf.GetConf().FirstTransName})
			} else if needRequstID(1, index) {
				ctx.Reply(&messages.ReqBlockInfo{Chain: 1, Index: index})
			}
			createSystemAPP(1)
			c := conf.GetConf()
			_, err := os.Stat("./db_dir/recalculation")
			if c.ReliaRecalculation || os.IsNotExist(err) {
				ioutil.WriteFile("./db_dir/recalculation", []byte("111"), 0666)
				go reliaRecalculation(1)
			}
		}
	}

	return nil
}

func createSystemAPP(chain uint64) {
	id := core.GetLastBlockIndex(chain)
	if id == 0 {
		return
	}
	core.CreateBiosTrans(chain)
	createSystemAPP(2 * chain)
	createSystemAPP(2*chain + 1)
}

func processBlock(chain uint64, key, data []byte) (err error) {
	needSave := true
	hp := getHashPower(key)
	if hp < 5 {
		return errors.New("error hashpower")
	}
	if len(data) == 0 {
		data = core.ReadBlockData(chain, key)
		needSave = false
	}

	// 解析block
	block := core.DecodeBlock(data)
	if block == nil {
		log.Printf("error block,chain:%d,key:%x\n", chain, key)
		return errors.New("fail to decode")
	}

	if bytes.Compare(key, block.Key[:]) != 0 {
		log.Printf("error block key,chain:%d,hope key:%x,key:%x\n", chain, key, block.Key[:])
		return errors.New("different key")
	}

	//first block
	if chain != block.Chain {
		if block.Chain != 0 {
			log.Printf("error chain of block,hope chain:%d,chain:%d,key:%x\n", chain, block.Chain, key)
			return errors.New("error chain")
		}

		if chain > 1 {
			nIndex := core.GetLastBlockIndex(chain / 2)
			if nIndex < 1 {
				return errors.New("not parent")
			}
		}
	}

	now := uint64(time.Now().Unix()) * 1000
	if block.Index > 2 && block.Time > now+blockAcceptTime {
		return errors.New("too new")
	}

	// block已经处理过了，忽略
	lKey := core.GetTheBlockKey(chain, block.Index)
	if lKey != nil && bytes.Compare(key, lKey) == 0 {
		rel := block.GetReliability()
		core.SaveBlockReliability(chain, block.Key[:], rel)
		return
	}

	if chain > 1 && block.Index > 1 {
		if block.Parent.Empty() {
			return errors.New("empty parent")
		}
	}

	if block.Index > 2 {
		preRel := core.ReadBlockReliability(block.Chain, block.Previous[:])
		if block.Time < preRel.Time+core.GetBlockInterval(block.Chain)*9/10 {
			return errors.New("error block.Time")
		}
	}

	// 将数据写入db
	rel := block.GetReliability()
	core.SaveBlockReliability(chain, block.Key[:], rel)
	SaveTransList(chain, block.Key[:], block.TransList)
	if needSave {
		core.WriteBlock(chain, data)
		val := uint64(1) << hp
		hpi := time.Now().Unix() / 60
		old, ok := blockHP.Get(keyOfBlockHP{chain, hpi})
		if ok {
			val += old.(uint64)
		}
		blockHP.Set(keyOfBlockHP{chain, hpi}, val)
	}

	return
}

func (p *MsgPlugin) downloadBlockDepend(ctx libp2p.Event, chain uint64, key []byte) {
	log.Printf("downloadBlockDepend,chain:%d,key:%x\n", chain, key)
	rel := core.ReadBlockReliability(chain, key)
	if rel.Ready {
		ctx.GetSession().SetEnv(getEnvKey(chain, transOwner), "")
		return
	}
	if !rel.Previous.Empty() && !core.IsExistBlock(chain, rel.Previous[:]) {
		log.Printf("previous is not exist")
		return
	}
	if !rel.Parent.Empty() && !core.IsExistBlock(chain/2, rel.Parent[:]) {
		log.Printf("parent is not exist")
		return
	}
	if !rel.LeftChild.Empty() && !core.IsExistBlock(chain*2, rel.LeftChild[:]) {
		log.Printf("leftChild is not exist")
		return
	}
	if !rel.RightChild.Empty() && !core.IsExistBlock(chain*2+1, rel.RightChild[:]) {
		log.Printf("rightChild is not exist")
		return
	}
	transList := GetTransList(chain, key)
	for _, it := range transList {
		if core.IsExistTransaction(chain, it[:]) {
			continue
		}
		e := hex.EncodeToString(it[:])
		ctx.GetSession().SetEnv(getEnvKey(chain, transOfBlock), e)
		e = hex.EncodeToString(key)
		ctx.GetSession().SetEnv(getEnvKey(chain, transOwner), e)
		ctx.Reply(&messages.ReqTransaction{Chain: chain, Key: it[:]})
		log.Printf("trans is not exist,chain:%d,key:%x\n", chain, it)
		return
	}
	rel.Recalculation(chain)
	rel.Ready = true
	core.SaveBlockReliability(chain, rel.Key[:], rel)
	ctx.GetSession().SetEnv(getEnvKey(chain, transOwner), "")

	var hpLimit uint64
	getData(chain, ldbHPLimit, runtime.Encode(rel.Index-1), &hpLimit)
	if rel.HashPower+hpAcceptRange >= hpLimit {
		// log.Printf("setBlockToIDBlocks,chain:%d,index:%d,key:%x,hp:%d\n", chain, rel.Index, rel.Key, rel.HashPower)
		setBlockToIDBlocks(chain, rel.Index, rel.Key, rel.HashPower)
	}

	go processEvent(chain)
	return
}

func processTransaction(chain uint64, key, data []byte) error {
	if chain == 0 {
		return errors.New("not support")
	}

	trans := core.DecodeTrans(data)
	if trans == nil {
		return errors.New("error transaction")
	}
	if bytes.Compare(trans.Key, key) != 0 {
		return errors.New("error transaction key")
	}

	nIndex := core.GetLastBlockIndex(chain)
	if trans.Chain > 1 {
		if nIndex <= 1 {
			return errors.New("chain no exist")
		}
	}

	c := conf.GetConf()
	if trans.Chain != chain {
		if trans.Chain != 0 {
			return errors.New("different chain,the Chain of trans must be 0")
		}
		if bytes.Compare(c.FirstTransName, key) != 0 {
			return errors.New("different first transaction")
		}
	}

	now := uint64(time.Now().Unix()) * 1000
	// future trans
	if trans.Time > now+blockAcceptTime {
		return errors.New("error time")
	}

	err := core.WriteTransaction(chain, data)
	if err != nil {
		return err
	}

	if bytes.Compare(trans.User[:], c.WalletAddr) == 0 {
		k := runtime.Encode(^uint64(0) - trans.Time)
		k = append(k, trans.Key[:16]...)
		ldb.LSet(chain, ldbOutputTrans, k, trans.Key[:])
	} else if trans.Ops == core.OpsTransfer && bytes.Compare(trans.Data[:core.AddressLen], c.WalletAddr) == 0 {
		k := runtime.Encode(^uint64(0) - trans.Time)
		k = append(k, trans.Key[:16]...)
		ldb.LSet(chain, ldbInputTrans, k, trans.Key[:])
	}

	if (believable(chain, trans.User[:]) || bytes.Compare(trans.User[:], c.WalletAddr) == 0) &&
		(chain == c.ChainOfMine || c.ChainOfMine == 0) &&
		trans.Energy >= c.EnergyOfTrans &&
		trans.Time <= now && trans.Time+transAcceptTime > now {

		info := transInfo{}
		info.TransactionHead = trans.TransactionHead
		runtime.Decode(trans.Key, &info.Key)
		info.Size = uint32(len(data))
		rst := core.CheckTransaction(chain, trans.Key)
		if rst == nil {
			saveTransInfo(chain, trans.Key, info)
		} else if rst.Error() == "recover:trans_newer" {
			// newer transaction
			t := uint64(time.Now().Unix()) + blockSyncTime
			k := runtime.Encode(t)
			k = append(k, trans.Key[:]...)
			ldb.LSet(chain, ldbNewerTrans, k, runtime.Encode(info))
		}
	}

	log.Printf("new transaction.chain%d, key:%x ,osp:%d\n", chain, key, trans.Ops)

	return nil
}

func dbRollBack(chain, index uint64, key []byte) error {
	var err error
	nIndex := core.GetLastBlockIndex(chain)
	if nIndex < index {
		return nil
	}
	if nIndex > index+100 {
		return fmt.Errorf("the index < (lastIndex -100),will rollback:%d,last index:%d", index, nIndex)
	}

	lKey := core.GetTheBlockKey(chain, index)
	if bytes.Compare(lKey, key) != 0 {
		log.Printf("dbRollBack error.different key of chain:%d,hope:%x,get:%x\n", chain, key, lKey)
		return errors.New("error block key of the index")
	}
	lKey = core.GetTheBlockKey(chain, nIndex)
	bln := getBlockLockNum(chain, lKey)
	client := database.GetClient()
	for nIndex >= index {
		lKey = core.GetTheBlockKey(chain, nIndex)
		err = client.Rollback(chain, lKey)
		log.Printf("dbRollBack,chain:%d,index:%d,key:%x\n", chain, nIndex, lKey)
		if err != nil {
			log.Println("fail to Rollback.", nIndex, err)
			f := client.GetLastFlag(chain)
			client.Cancel(chain, f)
			return err
		}
		setBlockLockNum(chain, lKey, bln)
		stat := ReadBlockRunStat(chain, lKey)
		stat.RollbackCount++
		SaveBlockRunStat(chain, lKey, stat)
		// core.DeleteBlockReliability(chain, lKey)
		var lk core.Hash
		runtime.Decode(lKey, &lk)
		setBlockToIDBlocks(chain, nIndex, lk, 0)
		transList := GetTransList(chain, lKey)
		for _, trans := range transList {
			v := ldb.LGet(chain, ldbAllTransInfo, trans[:])
			ldb.LSet(chain, ldbTransInfo, trans[:], v)
		}
		nIndex--
		bln++
	}

	return nil
}

// GetHashPowerOfBlocks get average hashpower of blocks
func GetHashPowerOfBlocks(chain uint64) uint64 {
	procMgr.mu.Lock()
	defer procMgr.mu.Unlock()
	var sum uint64
	hpi := time.Now().Unix() / 60
	for i := hpi - blockHPNumber; i < hpi; i++ {
		v, ok := blockHP.Get(keyOfBlockHP{chain, i})
		if ok {
			sum += v.(uint64)
		}
	}

	return sum / blockHPNumber
}
