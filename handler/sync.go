package handler

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/libp2p"
)

// SyncPlugin sync plugin
type SyncPlugin struct {
	*libp2p.Plugin
	net     libp2p.Network
	syncCID string
	timeout int64
}

const (
	eSyncing = byte(iota)
	eSyncBlock
	eSyncTrans
	eSyncTransOwner
	eSyncLastBlock
	sTimeout      = 40
	acceptBlockID = 20
	maxSyncNum    = 300
)

func getSyncEnvKey(chain uint64, typ byte) string {
	return fmt.Sprintf("sync_%x_%x", chain, typ)
}

// Startup is called only once when the plugin is loaded
func (p *SyncPlugin) Startup(n libp2p.Network) {
	p.net = n
}

func timeToString(t int64) string {
	return fmt.Sprintf("%020d", t)
}

// PeerDisconnect peer disconnect
func (p *SyncPlugin) PeerDisconnect(s libp2p.Session) {
	cid := s.GetEnv(libp2p.EnvConnectID)
	if p.syncCID == cid {
		p.syncCID = ""
		p.timeout = 0
	}
}

// Receive receive message
func (p *SyncPlugin) Receive(ctx libp2p.Event) error {
	switch msg := ctx.GetMessage().(type) {
	case *messages.BlockInfo:
		cid := ctx.GetSession().GetEnv(libp2p.EnvConnectID)
		now := time.Now().Unix()
		if now > p.timeout {
			ctx.GetSession().SetEnv(getSyncEnvKey(msg.Chain, eSyncing), "")
			if p.syncCID == cid {
				p.syncCID = ""
			}
		}

		// log.Printf("<%x> BlockInfo %d %d\n", ctx.GetPeerID(), msg.Chain, msg.Index)
		index := core.GetLastBlockIndex(msg.Chain)
		if index >= msg.Index {
			return nil
		}
		if index == 0 && msg.Chain > 1 {
			return nil
		}
		preKey := core.GetTheBlockKey(msg.Chain, msg.Index-1)
		if bytes.Compare(preKey, msg.PreKey) == 0 {
			return nil
		}
		if ctx.GetSession().GetEnv(getSyncEnvKey(msg.Chain, eSyncing)) != "" {
			return nil
		}
		if msg.Index > index+maxSyncNum {
			if p.syncCID == "" && needRequstID(msg.Chain, index+maxSyncNum) {
				ctx.Reply(&messages.ReqBlockInfo{Chain: msg.Chain, Index: index + maxSyncNum})
			}
			return nil
		}
		if msg.Index > index+acceptBlockID {
			if core.GetBlockTime(msg.Chain)+tHour > uint64(time.Now().Unix())*1000 &&
				needRequstID(msg.Chain, index+acceptBlockID) {
				ctx.Reply(&messages.ReqBlockInfo{Chain: msg.Chain, Index: index + acceptBlockID})
				return nil
			}
			if p.syncCID == "" || p.syncCID == cid {
				p.syncCID = cid
			} else if needRequstID(msg.Chain, index+acceptBlockID) {
				ctx.Reply(&messages.ReqBlockInfo{Chain: msg.Chain, Index: index + acceptBlockID})
				return nil
			}
		}
		if core.IsExistBlock(msg.Chain, msg.Key) {
			rel := ReadBlockReliability(msg.Chain, msg.Key)
			if !rel.Ready || rel.Index == 0 {
				log.Printf("start sync,chain:%d,index:%d,block:%x\n", msg.Chain, msg.Index, msg.Key)
				//start sync
				ctx.GetSession().SetEnv(getSyncEnvKey(msg.Chain, eSyncing), "true")
				go p.syncDepend(ctx, msg.Chain, msg.Key)
			} else {
				setBlockToIDBlocks(msg.Chain, rel.Index, rel.Key, 1)
			}
			return nil
		}

		if !needDownload(msg.Chain, msg.Key) {
			return nil
		}

		SetSyncBlock(msg.Chain, msg.Index, msg.Key)
		if p.syncCID == cid {
			p.timeout = now + sTimeout
		}

		ctx.GetSession().SetEnv(getSyncEnvKey(msg.Chain, eSyncBlock), hex.EncodeToString(msg.Key))
		ctx.GetSession().SetEnv(getSyncEnvKey(msg.Chain, eSyncing), "true")
		ctx.GetSession().SetEnv(getSyncEnvKey(msg.Chain, eSyncLastBlock), hex.EncodeToString(msg.Key))
		ctx.Reply(&messages.ReqBlock{Chain: msg.Chain, Index: msg.Index, Key: msg.Key})
	case *messages.BlockData:
		if len(msg.Data) > 102400 {
			return nil
		}
		e := ctx.GetSession().GetEnv(getSyncEnvKey(msg.Chain, eSyncBlock))
		k := hex.EncodeToString(msg.Key)
		if e != k {
			return nil
		}
		defer ctx.GetSession().SetEnv(getSyncEnvKey(msg.Chain, eSyncBlock), "")
		err := processBlock(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return nil
		}
		go p.syncDepend(ctx, msg.Chain, msg.Key)
	case *messages.TransactionData:
		e := ctx.GetSession().GetEnv(getSyncEnvKey(msg.Chain, eSyncTrans))
		k := hex.EncodeToString(msg.Key)
		if e != k {
			return nil
		}
		ctx.GetSession().SetEnv(getSyncEnvKey(msg.Chain, eSyncTrans), "")
		err := processTransaction(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return err
		}
		e = ctx.GetSession().GetEnv(getSyncEnvKey(msg.Chain, eSyncTransOwner))
		if e == "" {
			return nil
		}
		key, _ := hex.DecodeString(e)
		if len(key) == 0 {
			return nil
		}
		go p.syncDepend(ctx, msg.Chain, key)

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
	// log.Printf("syncDepend,chain:%d,key:%x,from:%s\n", chain, key, ctx.GetSession().GetPeerAddr().Host())
	cid := ctx.GetSession().GetEnv(libp2p.EnvConnectID)
	if p.syncCID == cid {
		p.timeout = time.Now().Unix() + sTimeout
	}

	if !core.IsExistBlock(chain, key) {
		ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncBlock), hex.EncodeToString(key))
		ctx.Reply(&messages.ReqBlock{Chain: chain, Key: key})
		// log.Printf("syncDepend, ReqBlock,chain:%d,key:%x\n", chain, key)
		return
	}
	rel := ReadBlockReliability(chain, key)
	if rel.Index == 0 {
		err := processBlock(chain, key, nil)
		if err != nil {
			core.DeleteBlock(chain, key)
			SaveBlockReliability(chain, key, TReliability{})
			ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncBlock), hex.EncodeToString(key))
			ctx.Reply(&messages.ReqBlock{Chain: chain, Key: key})
			return
		}
		rel = ReadBlockReliability(chain, key)
	}
	id := core.GetLastBlockIndex(chain)
	if id > rel.Index+1000 {
		rel.Ready = true
		SaveBlockReliability(chain, rel.Key[:], rel)
		newKey := GetSyncBlock(chain, rel.Index+1)
		if len(newKey) > 0 {
			// log.Printf("start next SyncBlock,chain:%d,key:%x,next:%x\n", chain, key, newKey)
			go p.syncDepend(ctx, chain, newKey)
		} else {
			// log.Printf("stop sync,not next SyncBlock,chain:%d,key:%x,next:%d\n", chain, key, rel.Index+1)
			ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncBlock), "")
			ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncing), "")
			updateBLN(chain, key)
			go processEvent(chain)
		}
		return
	}
	SetSyncBlock(chain, rel.Index, key)
	if !rel.Previous.Empty() {
		pRel := ReadBlockReliability(chain, rel.Previous[:])
		if !pRel.Ready {
			// log.Printf("syncDepend previous,chain:%d,index:%d,preKey:%x\n", chain, rel.Index, rel.Previous)
			go p.syncDepend(ctx, chain, rel.Previous[:])
			return
		}
	}
	if !rel.Parent.Empty() {
		pRel := ReadBlockReliability(chain/2, rel.Parent[:])
		if !pRel.Ready {
			// log.Printf("syncDepend Parent,chain:%d,index:%d,Parent:%x\n", chain, rel.Index, rel.Parent)
			go p.syncDepend(ctx, chain/2, rel.Parent[:])
			return
		}
	}
	if !rel.LeftChild.Empty() {
		lRel := ReadBlockReliability(chain*2, rel.LeftChild[:])
		if !lRel.Ready {
			// log.Printf("syncDepend LeftChild,chain:%d,index:%d,LeftChild:%x\n", chain, rel.Index, rel.LeftChild)
			go p.syncDepend(ctx, chain*2, rel.LeftChild[:])
			return
		}
	}
	if !rel.RightChild.Empty() {
		rRel := ReadBlockReliability(chain*2+1, rel.RightChild[:])
		if !rRel.Ready {
			// log.Printf("syncDepend RightChild,chain:%d,index:%d,RightChild:%x\n", chain, rel.Index, rel.RightChild)
			go p.syncDepend(ctx, chain*2, rel.RightChild[:])
			return
		}
	}
	transList := GetTransList(chain, key)
	for _, it := range transList {
		if core.IsExistTransaction(chain, it[:]) {
			continue
		}
		// log.Printf("syncDepend trans,chain:%d,index:%d,trans:%x\n", chain, rel.Index, it)
		e := hex.EncodeToString(it[:])
		ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncTrans), e)
		e = hex.EncodeToString(key)
		ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncTransOwner), e)
		ctx.Reply(&messages.ReqTransaction{Chain: chain, Key: it[:]})
		return
	}

	data := core.ReadBlockData(chain, key)
	err := processBlock(chain, key, data)
	if err != nil {
		ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncBlock), "")
		ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncing), "")
		return
	}

	rel.Recalculation(chain)
	rel.Ready = true
	SaveBlockReliability(chain, rel.Key[:], rel)

	SetSyncBlock(chain, rel.Index, nil)
	t := core.GetBlockTime(chain)
	now := uint64(time.Now().Unix() * 1000)
	if t+blockSyncTime < now {
		setBlockToIDBlocks(chain, rel.Index, rel.Key, rel.HashPower)
	}

	newKey := GetSyncBlock(chain, rel.Index+1)
	if len(newKey) > 0 {
		// log.Printf("start next SyncBlock,chain:%d,key:%x,next:%x\n", chain, key, newKey)
		go p.syncDepend(ctx, chain, newKey)
	} else {
		// log.Printf("stop sync,not next SyncBlock,chain:%d,key:%x,next:%d\n", chain, key, rel.Index+1)
		ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncBlock), "")
		ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncing), "")
		if p.syncCID == cid {
			p.timeout = 0
		}
		updateBLN(chain, key)
		go processEvent(chain)
	}
}

func updateBLN(chain uint64, key []byte) {
	if len(key) == 0 {
		return
	}
	rel := ReadBlockReliability(chain, key)
	bln := getBlockLockNum(chain, key)
	index := core.GetLastBlockIndex(chain)
	for rel.Index > index {
		old := getBlockLockNum(chain, rel.Previous[:])
		if old > bln {
			break
		}
		bln++
		setBlockLockNum(chain, rel.Previous[:], bln)
		if !rel.LeftChild.Empty() {
			setBlockLockNum(chain*2, rel.LeftChild[:], 2*bln)
		}
		if !rel.RightChild.Empty() {
			setBlockLockNum(chain*2, rel.RightChild[:], 2*bln)
		}
		rel = ReadBlockReliability(chain, rel.Previous[:])
	}
}
