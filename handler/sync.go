package handler

import (
	"bytes"
	"encoding/hex"
	"fmt"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/libp2p"
	"log"
	"runtime/debug"
	"time"
)

// SyncPlugin sync plugin
type SyncPlugin struct {
	*libp2p.Plugin
	net libp2p.Network
}

const (
	eSyncing        = "is_syncing"
	eSyncBlock      = "sync_block"
	eSyncTrans      = "sync_trans"
	eSyncTransOwner = "sync_owner"
	eSyncTimeout    = "sync_timeout"
	sTimeout        = 120
)

// Startup is called only once when the plugin is loaded
func (p *SyncPlugin) Startup(n libp2p.Network) {
	p.net = n
}

func timeToString(t int64) string {
	return fmt.Sprintf("%020d", t)
}

// Receive receive message
func (p *SyncPlugin) Receive(ctx libp2p.Event) error {
	switch msg := ctx.GetMessage().(type) {
	case *messages.BlockInfo:
		// log.Printf("<%x> BlockInfo %d %d\n", ctx.GetPeerID(), msg.Chain, msg.Index)
		index := core.GetLastBlockIndex(msg.Chain)
		if index >= msg.Index {
			return nil
		}
		preKey := core.GetTheBlockKey(msg.Chain, msg.Index-1)
		if bytes.Compare(preKey, msg.PreKey) == 0 {
			return nil
		}
		if ctx.GetSession().GetEnv(eSyncing) != "" {
			return nil
		}
		if msg.Index > index+50 {
			ctx.Reply(&messages.ReqBlockInfo{Chain: msg.Chain, Index: index + 50})
			return nil
		}
		if core.IsExistBlock(msg.Chain, msg.Key) {
			rel := core.ReadBlockReliability(msg.Chain, msg.Key)
			if !rel.Ready {
				log.Printf("start sync,chain:%d,block:%x\n", msg.Chain, msg.Key)
				//start sync
				ctx.GetSession().SetEnv(eSyncing, "true")
				go p.syncDepend(ctx, msg.Chain, msg.Key)
			}
			return nil
		}
		SetSyncBlock(msg.Chain, msg.Index, msg.Key)
		to := timeToString(time.Now().Unix() + sTimeout)
		ctx.GetSession().SetEnv(eSyncTimeout, to)
		ctx.GetSession().SetEnv(eSyncBlock, hex.EncodeToString(msg.Key))
		ctx.GetSession().SetEnv(eSyncing, "true")
		ctx.Reply(&messages.ReqBlock{Chain: msg.Chain, Index: msg.Index, Key: msg.Key})
	case *messages.BlockData:
		if len(msg.Data) > 102400 {
			return nil
		}
		e := ctx.GetSession().GetEnv(eSyncBlock)
		k := hex.EncodeToString(msg.Key)
		if e != k {
			return nil
		}
		defer ctx.GetSession().SetEnv(eSyncBlock, "")
		err := processBlock(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return nil
		}
		go p.syncDepend(ctx, msg.Chain, msg.Key)
	case *messages.TransactionData:
		e := ctx.GetSession().GetEnv(eSyncTrans)
		k := hex.EncodeToString(msg.Key)
		if e != k {
			return nil
		}
		ctx.GetSession().SetEnv(eSyncTrans, "")
		err := processTransaction(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return err
		}
		e = ctx.GetSession().GetEnv(transOwner)
		if e == "" {
			return nil
		}
		key, _ := hex.DecodeString(e)
		if len(key) == 0 {
			return nil
		}
		go p.syncDepend(ctx, msg.Chain, key)
	default:
		if ctx.GetSession().GetEnv(eSyncing) == "" {
			return nil
		}
		to := ctx.GetSession().GetEnv(eSyncTimeout)
		now := timeToString(time.Now().Unix())
		if now > to {
			ctx.GetSession().SetEnv(eSyncing, "")
		}

	}
	return nil
}

func (p *SyncPlugin) syncDepend(ctx libp2p.Event, chain uint64, key []byte) {
	defer func() {
		err := recover()
		if err != nil {
			log.Printf("syncDepend error,chain:%d,key:%x,err:%s\n", chain, key, err)
			log.Println(string(debug.Stack()))
		}
	}()
	log.Printf("syncDepend,chain:%d,key:%x\n", chain, key)
	to := timeToString(time.Now().Unix() + sTimeout)
	ctx.GetSession().SetEnv(eSyncTimeout, to)

	if !core.IsExistBlock(chain, key) {
		ctx.GetSession().SetEnv(eSyncBlock, hex.EncodeToString(key))
		ctx.Reply(&messages.ReqBlock{Chain: chain, Key: key})
		log.Printf("syncDepend, ReqBlock,chain:%d,key:%x\n", chain, key)
		return
	}
	rel := core.ReadBlockReliability(chain, key)
	SetSyncBlock(chain, rel.Index, key)
	if !rel.Previous.Empty() {
		pRel := core.ReadBlockReliability(chain, rel.Previous[:])
		if !pRel.Ready {
			log.Printf("syncDepend previous,chain:%d,index:%d,preKey:%x\n", chain, rel.Index, rel.Previous)
			go p.syncDepend(ctx, chain, rel.Previous[:])
			return
		}
	}
	if !rel.Parent.Empty() {
		pRel := core.ReadBlockReliability(chain/2, rel.Parent[:])
		if !pRel.Ready {
			log.Printf("syncDepend Parent,chain:%d,index:%d,Parent:%x\n", chain, rel.Index, rel.Parent)
			go p.syncDepend(ctx, chain/2, rel.Parent[:])
			return
		}
	}
	if !rel.LeftChild.Empty() {
		lRel := core.ReadBlockReliability(chain*2, rel.LeftChild[:])
		if !lRel.Ready {
			log.Printf("syncDepend LeftChild,chain:%d,index:%d,LeftChild:%x\n", chain, rel.Index, rel.LeftChild)
			go p.syncDepend(ctx, chain*2, rel.LeftChild[:])
			return
		}
	}
	if !rel.RightChild.Empty() {
		rRel := core.ReadBlockReliability(chain*2+1, rel.RightChild[:])
		if !rRel.Ready {
			log.Printf("syncDepend RightChild,chain:%d,index:%d,RightChild:%x\n", chain, rel.Index, rel.RightChild)
			go p.syncDepend(ctx, chain*2, rel.RightChild[:])
			return
		}
	}
	transList := GetTransList(chain, key)
	for _, it := range transList {
		if core.IsExistTransaction(chain, it[:]) {
			continue
		}
		log.Printf("syncDepend trans,chain:%d,index:%d,trans:%x\n", chain, rel.Index, it)
		e := hex.EncodeToString(it[:])
		ctx.GetSession().SetEnv(reqTrans, e)
		e = hex.EncodeToString(key)
		ctx.GetSession().SetEnv(transOwner, e)
		ctx.Reply(&messages.ReqTransaction{Chain: chain, Key: it[:]})
		return
	}
	rel.Ready = true
	core.SaveBlockReliability(chain, rel.Key[:], rel)
	SetSyncBlock(chain, rel.Index, nil)
	lastID := core.GetLastBlockIndex(chain)
	if rel.Index < lastID {
		ib := IDBlocks{}
		it := ItemBlock{}
		it.Key = rel.Key
		it.HashPower = rel.HashPower
		ib.Items = append(ib.Items, it)
		SaveIDBlocks(chain, rel.Index, ib)
	} else {
		setBlockToIDBlocks(chain, rel.Index, rel.Key, rel.HashPower)
	}

	newKey := GetSyncBlock(chain, rel.Index+1)
	if len(newKey) > 0 {
		log.Printf("start next SyncBlock,chain:%d,key:%x,next:%x\n", chain, key, newKey)
		go p.syncDepend(ctx, chain, newKey)
	} else {
		log.Printf("stop sync,not next SyncBlock,chain:%d,key:%x,next:%d\n", chain, key, rel.Index+1)
		ctx.GetSession().SetEnv(eSyncBlock, "")
		ctx.GetSession().SetEnv(eSyncing, "")

		go processEvent(chain)
	}
}
