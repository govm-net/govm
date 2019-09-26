package handler

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/lengzhao/govm/conf"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/libp2p"
	"log"
	"runtime/debug"
	"time"
)

// MsgPlugin process p2p message
type MsgPlugin struct {
	*libp2p.Plugin
	net libp2p.Network
}

const (
	connTime   = "connect_time"
	closeStat  = "is_closed"
	reqBlock   = "req_block"
	reqTrans   = "req_trans"
	transOwner = "trans_owner"
)

var network libp2p.Network

// Startup is called only once when the plugin is loaded
func (p *MsgPlugin) Startup(n libp2p.Network) {
	p.net = n
	network = n
}

// PeerConnect is called every time a PeerSession is initialized and connected
func (p *MsgPlugin) PeerConnect(s libp2p.Session) {
	s.SetEnv(connTime, fmt.Sprintf("%d", time.Now().Unix()))
}

// PeerDisconnect plugin.PeerDisconnect
func (p *MsgPlugin) PeerDisconnect(s libp2p.Session) {
	s.SetEnv(closeStat, "true")
}

var first = true

// Receive receive message
func (p *MsgPlugin) Receive(ctx libp2p.Event) error {
	switch msg := ctx.GetMessage().(type) {
	case *messages.ReqBlockInfo:
		log.Printf("<%x> ReqBlockInfo %d %d\n", ctx.GetPeerID(), msg.Chain, msg.Index)
		key := core.GetTheBlockKey(msg.Chain, msg.Index)
		if len(key) == 0 {
			log.Println("fail to get the key,index:", msg.Index, ",chain:", msg.Chain)
			return nil
		}
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
		log.Printf("<%x> BlockKey %d %d,key:%x\n", ctx.GetPeerID(), msg.Chain, msg.Index, msg.Key)
		hp := getHashPower(msg.Key)
		if hp < 5 || hp > 250 {
			return nil
		}

		index := core.GetLastBlockIndex(msg.Chain)
		if index > msg.Index {
			if index > msg.Index+50 {
				index = msg.Index + 50
			}
			key := core.GetTheBlockKey(msg.Chain, index)
			if len(key) > 0 {
				rel := core.ReadBlockReliability(msg.Chain, key)
				ctx.Reply(&messages.BlockInfo{Chain: msg.Chain,
					Index: rel.Index, Key: key, PreKey: rel.Previous[:], HashPower: rel.HashPower})
			}
			return nil
		}
		if msg.Index > index+1 {
			ctx.Reply(&messages.ReqBlockInfo{Chain: msg.Chain, Index: index + 1})
			return nil
		}

		preKey := core.GetTheBlockKey(msg.Chain, msg.Index-1)
		if bytes.Compare(preKey, msg.PreKey) != 0 {
			return nil
		}

		if core.IsExistBlock(msg.Chain, msg.Key) {
			log.Println("block exist:", msg.Index, ",chain:", msg.Chain, ",self:", index)

			rel := core.ReadBlockReliability(msg.Chain, msg.Key)
			if rel.HashPower == 0 {
				core.DeleteBlock(msg.Chain, msg.Key)
				log.Printf("error hashpower of block,delete it.chain:%d,key:%x\n", msg.Chain, msg.Key)
				return nil
			}
			if !rel.Ready {
				downloadBlockDepend(ctx, msg.Chain, msg.Key)
			}
			return nil
		}
		ctx.GetSession().SetEnv(reqBlock, hex.EncodeToString(msg.Key))
		ctx.Reply(&messages.ReqBlock{Chain: msg.Chain, Index: msg.Index, Key: msg.Key})
		key := core.GetTheBlockKey(msg.Chain, 0)
		rel := core.ReadBlockReliability(msg.Chain, key)
		if msg.Index == rel.Index && msg.HashPower < rel.HashPower {
			ctx.Reply(&messages.BlockInfo{Chain: msg.Chain, Index: rel.Index,
				Key: key, HashPower: rel.HashPower, PreKey: rel.Previous[:]})
		}
	case *messages.TransactionInfo:
		log.Printf("<%x> TransactionInfo %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		if len(msg.Key) != core.HashLen {
			return nil
		}
		t := core.GetBlockTime(msg.Chain)
		if msg.Time > t+processStartTime || msg.Time+transAcceptTime < t {
			return nil
		}
		if ctx.GetSession().GetEnv(reqTrans) != "" {
			return nil
		}
		if core.IsExistTransaction(msg.Chain, msg.Key) {
			return nil
		}
		ctx.GetSession().SetEnv(reqTrans, hex.EncodeToString(msg.Key))
		ctx.Reply(&messages.ReqTransaction{Chain: msg.Chain, Key: msg.Key})
	case *messages.ReqBlock:
		log.Printf("<%x> ReqBlock %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		if len(msg.Key) == 0 {
			return nil
		}

		data := core.ReadBlockData(msg.Chain, msg.Key)
		if len(data) == 0 {
			log.Printf("not found.ReqBlock chain:%d index:%d key:%x\n", msg.Chain, msg.Index, msg.Key)
			return nil
		}
		ctx.Reply(&messages.BlockData{Chain: msg.Chain, Key: msg.Key, Data: data})
	case *messages.ReqTransaction:
		log.Printf("<%x> ReqTransaction %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		data := core.ReadTransactionData(msg.Chain, msg.Key)
		if data == nil {
			log.Printf("not found the transaction,chain:%d,key:%x\n", msg.Chain, msg.Key)
			return nil
		}
		ctx.Reply(&messages.TransactionData{Chain: msg.Chain, Key: msg.Key, Data: data})
	case *messages.BlockData:
		log.Printf("<%x> BlockData %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		if len(msg.Data) > 102400 {
			return nil
		}
		e := ctx.GetSession().GetEnv(reqBlock)
		k := hex.EncodeToString(msg.Key)
		if e != k {
			return nil
		}
		ctx.GetSession().SetEnv(reqBlock, "")
		err := processBlock(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			log.Printf("fail to processBlock,chain:%d,key:%x,err:%s\n", msg.Chain, msg.Key, err)
			return err
		}
		downloadBlockDepend(ctx, msg.Chain, msg.Key)

	case *messages.TransactionData:
		log.Printf("<%x> TransactionData %d %x\n", ctx.GetPeerID(), msg.Chain, msg.Key)
		if len(msg.Key) != core.HashLen {
			return nil
		}
		e := ctx.GetSession().GetEnv(reqTrans)
		k := hex.EncodeToString(msg.Key)
		if e != k {
			return nil
		}

		ctx.GetSession().SetEnv(reqTrans, "")
		err := processTransaction(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return err
		}
		e = ctx.GetSession().GetEnv(transOwner)
		if e == "" {
			return nil
		}

		bk, _ := hex.DecodeString(e)
		downloadBlockDepend(ctx, msg.Chain, bk)

	default:
		//log.Println("msg", ctx.GetPeerID(), msg)
		if first {
			first = false
			index := core.GetLastBlockIndex(1)
			index++
			if index == 1 {
				ctx.Reply(&messages.ReqTransaction{Chain: 1, Key: conf.GetConf().FirstTransName})
			} else {
				ctx.Reply(&messages.ReqBlockInfo{Chain: 1, Index: index})
			}
		}
	}

	return nil
}

func blockRun(chain uint64, key []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("blockRun error,chain %d, key %x, err:%s", chain, key, e)
			log.Println(string(debug.Stack()))
		}
	}()

	c := conf.GetConf()
	err = database.OpenFlag(chain, key)
	if err != nil {
		log.Println("fail to open Flag,", err)
		f := database.GetLastFlag(chain)
		database.Cancel(chain, f)
		return
	}
	defer database.Cancel(chain, key)

	param := runtime.Encode(chain)
	param = append(param, key...)
	runtime.RunApp(key, chain, c.CorePackName, nil, param, 2<<50, 0)
	database.Commit(chain, key)

	return
}

func processBlock(chain uint64, key, data []byte) (err error) {
	if getHashPower(key) < 5 {
		return
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
			return
		}

		if chain > 1 {
			nIndex := core.GetLastBlockIndex(chain / 2)
			if nIndex < 1 {
				return
			}
		}
	}

	now := uint64(time.Now().Unix()) * 1000
	if block.Time > now+processStartTime {
		return errors.New("too new")
	}

	// block已经处理过了，忽略
	lKey := core.GetTheBlockKey(chain, block.Index)
	if lKey != nil && bytes.Compare(key, lKey) == 0 {
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
	core.WriteBlock(chain, data)
	rel := block.GetReliability()
	core.SaveBlockReliability(chain, block.Key[:], rel)
	SaveTransList(chain, block.Key[:], block.TransList)
	log.Printf("new block,chain:%d,index:%d,key:%x,hashpower:%d\n", chain, block.Index, block.Key, rel.HashPower)

	return
}

func downloadBlockDepend(ctx libp2p.Event, chain uint64, key []byte) (err error) {
	log.Printf("downloadBlockDepend,chain:%d,key:%x\n", chain, key)
	rel := core.ReadBlockReliability(chain, key)
	if !rel.Previous.Empty() && !core.IsExistBlock(chain, rel.Previous[:]) {
		// e := hex.EncodeToString(rel.Previous[:])
		// ctx.GetSession().SetEnv(reqBlock, e)
		// ctx.Reply(&messages.ReqBlock{Chain: chain, Index: rel.Index - 1, Key: rel.Previous[:]})
		log.Printf("previous is not exist")
		return
	}
	if !rel.Parent.Empty() && !core.IsExistBlock(chain/2, rel.Parent[:]) {
		// e := hex.EncodeToString(rel.Parent[:])
		// ctx.GetSession().SetEnv(reqBlock, e)
		// ctx.Reply(&messages.ReqBlock{Chain: chain / 2, Index: 0, Key: rel.Parent[:]})
		// event.Send(&messages.SyncChain{Chain: chain, Key: key, Peer: ctx})
		log.Printf("parent is not exist")
		return
	}
	if !rel.LeftChild.Empty() && !core.IsExistBlock(chain*2, rel.LeftChild[:]) {
		// e := hex.EncodeToString(rel.LeftChild[:])
		// ctx.GetSession().SetEnv(reqBlock, e)
		// ctx.Reply(&messages.ReqBlock{Chain: chain * 2, Index: 0, Key: rel.LeftChild[:]})
		// event.Send(&messages.SyncChain{Chain: chain, Key: key, Peer: ctx})
		log.Printf("leftChild is not exist")
		return
	}
	if !rel.RightChild.Empty() && !core.IsExistBlock(chain*2+1, rel.RightChild[:]) {
		// e := hex.EncodeToString(rel.RightChild[:])
		// ctx.GetSession().SetEnv(reqBlock, e)
		// ctx.Reply(&messages.ReqBlock{Chain: chain*2 + 1, Index: 0, Key: rel.RightChild[:]})
		// event.Send(&messages.SyncChain{Chain: chain, Key: key, Peer: ctx})
		log.Printf("rightChild is not exist")
		return
	}
	transList := GetTransList(chain, key)
	for _, it := range transList {
		if core.IsExistTransaction(chain, it[:]) {
			continue
		}
		e := hex.EncodeToString(it[:])
		ctx.GetSession().SetEnv(reqTrans, e)
		e = hex.EncodeToString(key)
		ctx.GetSession().SetEnv(transOwner, e)
		ctx.Reply(&messages.ReqTransaction{Chain: chain, Key: it[:]})
		log.Printf("trans is not exist,chain:%d,key:%x\n", chain, it)
		return
	}
	//todo 确保前一个block已经ready，否则启动sync

	log.Printf("setBlockToIDBlocks,chain:%d,index:%d,key:%x,hp:%d\n", chain, rel.Index, rel.Key, rel.HashPower)
	ctx.GetSession().SetEnv(transOwner, "")
	setBlockToIDBlocks(chain, rel.Index, rel.Key, rel.HashPower)
	return nil
}

func processTransaction(chain uint64, key, data []byte) error {
	if chain == 0 {
		return errors.New("not support")
	}

	trans := core.DecodeTrans(data)
	if trans == nil {
		return errors.New("error transaction")
	}
	if bytes.Compare(trans.Key[:], key) != 0 {
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

	// future trans
	if trans.Time > uint64(time.Now().Unix())*1000+processStartTime {
		return errors.New("error time")
	}

	err := core.WriteTransaction(chain, data)
	if err != nil {
		return err
	}

	if (chain == c.ChainOfMine || c.ChainOfMine == 0) && trans.Energy > c.EnergyLimitOfMine {
		addTrans(key, trans.User[:], chain, trans.Time, trans.Energy, uint64(len(data)))
	}

	log.Printf("receive new transaction:%d, %x \n", chain, key)

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
		return errors.New("error block key of the index")
	}
	for nIndex >= index {
		lKey = core.GetTheBlockKey(chain, nIndex)
		err = database.Rollback(chain, lKey)
		log.Printf("dbRollBack,chain:%d,index:%d,key:%x\n", chain, nIndex, lKey)
		if err != nil {
			log.Println("fail to Rollback.", nIndex, err)
			return err
		}
		stat := ReadBlockRunStat(chain, lKey)
		stat.RollbackCount++
		SaveBlockRunStat(chain, lKey, stat)
		// core.DeleteBlockReliability(chain, lKey)
		nIndex--
	}

	return nil
}
