package handler

import (
	"bytes"
	"encoding/hex"
	"errors"
	"expvar"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/govm-net/govm/conf"
	core "github.com/govm-net/govm/core"
	"github.com/govm-net/govm/database"
	"github.com/govm-net/govm/messages"
	"github.com/govm-net/govm/runtime"
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
var msgStat = expvar.NewMap("blocks")

const blockHPNumber = 120

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
		// log.Printf("<%x> ReqBlockInfo %d %d\n", ctx.GetPeerID(), msg.Chain, msg.Index)
		rel := ReadBlockReliability(msg.Chain, key)
		resp := new(messages.BlockInfo)
		resp.Chain = msg.Chain
		resp.Index = msg.Index
		resp.Key = key
		resp.PreKey = rel.Previous[:]
		resp.HashPower = rel.HashPower
		ctx.Reply(resp)
		msgStat.Add("ReqBlockInfo", 1)
		return nil
	case *messages.TransactionInfo:
		if len(msg.Key) != core.HashLen {
			return nil
		}
		if !needDownload(msg.Chain, msg.Key) {
			return nil
		}

		t := core.GetBlockTime(msg.Chain)
		if msg.Time > t+blockAcceptTime || msg.Time+transAcceptTime < t {
			return nil
		}
		if core.IsExistTransaction(msg.Chain, msg.Key) {
			return nil
		}
		// log.Printf("<%x> TransactionInfo %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		ctx.GetSession().SetEnv(getEnvKey(msg.Chain, reqTrans), hex.EncodeToString(msg.Key))
		ctx.Reply(&messages.ReqTransaction{Chain: msg.Chain, Key: msg.Key})
		msgStat.Add("TransactionInfo", 1)
	case *messages.ReqBlock:
		if len(msg.Key) == 0 {
			return nil
		}

		data := core.ReadBlockData(msg.Chain, msg.Key)
		if len(data) == 0 {
			log.Printf("not found.ReqBlock chain:%d index:%d key:%x\n", msg.Chain, msg.Index, msg.Key)
			return nil
		}
		msgStat.Add("ReqBlock", 1)
		// log.Printf("<%x> ReqBlock %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		ctx.Reply(&messages.BlockData{Chain: msg.Chain, Key: msg.Key, Data: data})
	case *messages.ReqTransList:
		if len(msg.Key) == 0 {
			return nil
		}
		var data []byte
		transList := GetTransList(msg.Chain, msg.Key)
		if len(transList) > 0 {
			data = core.TransListToBytes(transList)
		} else {
			data = core.ReadTransList(msg.Chain, msg.Key)
		}
		if len(data) == 0 {
			return nil
		}
		msgStat.Add("ReqTransList", 1)
		ctx.Reply(&messages.TransactionList{Chain: msg.Chain, Key: msg.Key, Data: data})
	case *messages.ReqTransaction:
		data := core.ReadTransactionData(msg.Chain, msg.Key)
		if data == nil {
			log.Printf("not found the transaction,chain:%d,key:%x\n", msg.Chain, msg.Key)
			return nil
		}
		msgStat.Add("ReqTransaction", 1)
		// log.Printf("<%x> ReqTransaction %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		ctx.Reply(&messages.TransactionData{Chain: msg.Chain, Key: msg.Key, Data: data})
	default:
		//log.Println("msg", ctx.GetPeerID(), msg)
		if first {
			first = false
			index := core.GetLastBlockIndex(1)
			index++
			ctx.Reply(&messages.ReqBlockInfo{Chain: 1, Index: index})
			createSystemAPP(1)
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

	now := getCoreTimeNow()
	if block.Index > 2 && block.Time > now+blockAcceptTime {
		log.Printf("block too new,chain:%d,key:%x,Producer:%x\n", block.Chain, block.Key, block.Producer)
		return errors.New("too new")
	}

	// block已经处理过了，忽略
	lKey := core.GetTheBlockKey(chain, block.Index)
	if lKey != nil && bytes.Compare(key, lKey) == 0 {
		rel := getReliability(block)
		rel.Ready = true
		rel.HashPower = getHashPower(key)
		SaveBlockReliability(chain, block.Key[:], rel)
		return
	}

	if chain > 1 && block.Index > 1 {
		if block.Parent.Empty() {
			return errors.New("empty parent")
		}
	}

	if block.Index > 2 {
		preRel := ReadBlockReliability(block.Chain, block.Previous[:])
		if block.Time < preRel.Time+core.GetBlockInterval(block.Chain)*9/10 {
			return errors.New("error block.Time")
		}
	}
	msgStat.Add("processBlock", 1)
	rel := getReliability(block)
	SaveBlockReliability(chain, block.Key[:], rel)
	if needSave {
		core.WriteBlock(chain, data)
		if hp > 20 {
			hp = core.GetHashPowerLimit(chain)
		}
		val := uint64(1) << hp
		hpi := int64(block.Time / 1000 / 60)
		old, ok := blockHP.Get(keyOfBlockHP{chain, hpi})
		if ok {
			val += old.(uint64)
		}
		blockHP.Set(keyOfBlockHP{chain, hpi}, val)
	}

	if !rel.TransListHash.Empty() {
		return
	}

	if rel.Time+tMinute > getCoreTimeNow() && needBroadcastBlock(chain, rel) {
		log.Printf("BroadcastBlock,chain:%d,index:%d,key:%x\n", chain, rel.Index, rel.Key)
		info := messages.BlockInfo{}
		info.Chain = chain
		info.Index = rel.Index
		info.Key = rel.Key[:]
		info.HashPower = rel.HashPower
		info.PreKey = rel.Previous[:]
		network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})
	}
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

	if trans.Chain != chain {
		return errors.New("different chain,the Chain of trans must be 0")
	}

	now := getCoreTimeNow()
	// future trans
	if trans.Time > now+tHour {
		return errors.New("error time")
	}

	err := core.WriteTransaction(chain, data)
	if err != nil {
		return err
	}

	log.Printf("new transaction.chain%d, key:%x ,osp:%d\n", chain, key, trans.Ops)

	blockTime := core.GetBlockTime(chain)
	if blockTime+processTransTime < now {
		return nil
	}

	tInfo := transInfo{}
	tInfo.TransactionHead = trans.TransactionHead
	runtime.Decode(trans.Key, &tInfo.Key)
	tInfo.Size = uint32(len(data))
	rst := core.CheckTransaction(chain, trans.Key)
	if rst == nil {
		saveTransInfo(chain, trans.Key, tInfo)
		pushTransInfo(chain, &tInfo)
	}
	msgStat.Add("processTransaction", 1)

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
	client := database.GetClient()
	for nIndex >= index {
		msgStat.Add("dbRollBack", 1)
		lKey = core.GetTheBlockKey(chain, nIndex)
		err = client.Rollback(chain, lKey)
		log.Printf("dbRollBack,chain:%d,index:%d,key:%x\n", chain, nIndex, lKey)
		if err != nil {
			log.Println("fail to Rollback.", nIndex, err)
			f := client.GetLastFlag(chain)
			client.Cancel(chain, f)
			return err
		}
		stat := ReadBlockRunStat(chain, lKey)
		stat.RollbackCount++
		SaveBlockRunStat(chain, lKey, stat)
		// core.DeleteBlockReliability(chain, lKey)
		var lk core.Hash
		runtime.Decode(lKey, &lk)
		// setBlockToIDBlocks(chain, nIndex, lk, 0)
		rel := ReadBlockReliability(chain, lKey)
		if !rel.TransListHash.Empty() {
			lst := GetTransList(chain, rel.TransListHash[:])
			for _, it := range lst {
				trans := readTransInfo(chain, it[:])
				if !trans.Key.Empty() {
					trans.Flag = time.Now().UnixNano()
					pushTransInfo(chain, &trans)
				}
			}
		}

		nIndex--
	}

	return nil
}

// GetHashPowerOfBlocks get average hashpower of blocks
func GetHashPowerOfBlocks(chain uint64) uint64 {
	procMgr.mu.Lock()
	defer procMgr.mu.Unlock()
	var sum uint64
	var count uint64
	hpi := time.Now().Unix() / 60
	for i := hpi - blockHPNumber; i < hpi; i++ {
		v, ok := blockHP.Get(keyOfBlockHP{chain, i})
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

func startCheckBlock() {
	var rid uint64
	id := core.GetLastBlockIndex(1) - 30
	rid = id
	for {
		key := getKeyFromServer(1, rid)
		if len(key) == 0 {
			break
		}
		ok := core.BlockOnTheChain(1, key)
		if ok {
			break
		}
		rid -= 100
		if id > rid+5000 {
			fmt.Println("The local block is different from the server.")
			os.Exit(5)
		}
	}
	if id > rid {
		if !conf.GetConf().AutoRollback {
			fmt.Println("The local block is different from the server.")
			os.Exit(5)
		}
		autoRollback(1, rid, nil)
	}
}

func getKeyFromServer(chain, id uint64) []byte {
	c := conf.GetConf()
	if c.TrustedServer == "" {
		log.Println("check block,server is null")
		return nil
	}
	if !c.CheckBlock {
		return nil
	}
	if id < 1 {
		return nil
	}
	urlStr := fmt.Sprintf("%s/api/v1/1/block/trusted?index=%d", c.TrustedServer, id)
	resp, err := http.Get(urlStr)
	if err != nil {
		log.Println("fail to check block,", err)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		log.Println("error response of check block,", resp.Status)
		return nil
	}
	key, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("check block,fail to read body:", err)
		return nil
	}
	return key
}

func autoRollback(chain, index uint64, key []byte) error {
	nIndex := core.GetLastBlockIndex(chain)
	client := database.GetClient()
	lKey := core.GetTheBlockKey(chain, 0)
	client.Cancel(chain, lKey)
	var err error
	var count int
	for nIndex >= index {
		lKey = core.GetTheBlockKey(chain, 0)
		err = client.Rollback(chain, lKey)
		if err != nil {
			log.Println("fail to rollback:", err)
			return err
		}
		nIndex = core.GetLastBlockIndex(chain)
		count++
		if count > 10000 {
			log.Println("rollback too many.", count)
			return fmt.Errorf("rollback too many")
		}
	}
	info := core.GetChainInfo(chain)
	if info.LeftChildID > 0 {
		err = autoRollback(2*chain, info.LeftChildID, nil)
		if err != nil {
			return err
		}
	}
	if info.RightChildID > 0 {
		err = autoRollback(2*chain+1, info.RightChildID, nil)
	}
	return err
}
