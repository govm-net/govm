package handler

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"runtime/debug"

	core "github.com/govm-net/govm/core"
	"github.com/govm-net/govm/messages"
	"github.com/lengzhao/libp2p"
)

// SyncPlugin sync plugin
type SyncPlugin struct {
	*libp2p.Plugin
	net libp2p.Network
}

const (
	eSyncTrans = byte(iota)
	eSyncTransOwner
	acceptBlockID = 20
	maxSyncNum    = 100
	acceptOldID   = 10
)

func getSyncEnvKey(chain uint64, typ byte) string {
	return fmt.Sprintf("sync_%x_%x", chain, typ)
}

// Startup is called only once when the plugin is loaded
func (p *SyncPlugin) Startup(n libp2p.Network) {
	p.net = n
}

// Receive receive message
func (p *SyncPlugin) Receive(ctx libp2p.Event) error {
	switch msg := ctx.GetMessage().(type) {
	case *messages.BlockInfo:
		// log.Printf("<%x> BlockInfo %d %d\n", ctx.GetPeerID()[:6], msg.Chain, msg.Index)
		if !needDownload(msg.Chain, msg.Key) {
			return nil
		}

		index := core.GetLastBlockIndex(msg.Chain)
		if index > msg.Index+acceptOldID {
			return nil
		}
		if index == 0 && msg.Chain > 1 {
			return nil
		}
		if msg.Index > index+maxSyncNum+10 {
			if needRequstID(msg.Chain, index+maxSyncNum) {
				ctx.Reply(&messages.ReqBlockInfo{Chain: msg.Chain, Index: index + maxSyncNum})
			}
			return nil
		}

		if core.IsExistBlock(msg.Chain, msg.Key) {
			rel := ReadBlockReliability(msg.Chain, msg.Key)
			if !rel.Ready || rel.Index == 0 {
				// log.Printf("start sync,chain:%d,index:%d,block:%x\n", msg.Chain, msg.Index, msg.Key)
				go p.syncDepend(ctx, msg.Chain, msg.Key)
			} else {
				setIDBlocks(msg.Chain, rel.Index, rel.Key, rel.HashPower)
				updateBLN(msg.Chain, rel.Key[:])
			}
			return nil
		}

		SetSyncBlock(msg.Chain, msg.Index, msg.Key)
		ctx.Reply(&messages.ReqBlock{Chain: msg.Chain, Index: msg.Index, Key: msg.Key})
	case *messages.BlockData:
		if len(msg.Data) > 102400 {
			return nil
		}
		// log.Printf("<%x> BlockData %d %x\n", ctx.GetPeerID()[:6], msg.Chain, msg.Key)
		err := processBlock(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return nil
		}
		go p.syncDepend(ctx, msg.Chain, msg.Key)
	case *messages.TransactionList:
		if len(msg.Data)%core.HashLen != 0 {
			return nil
		}
		transList := core.ParseTransList(msg.Data)
		if len(transList) == 0 {
			return nil
		}
		hk := core.GetHashOfTransList(transList)
		if bytes.Compare(hk[:], msg.Key) != 0 {
			return nil
		}
		SaveTransList(msg.Chain, msg.Key, transList)
		core.WriteTransList(msg.Chain, transList)
		s := ctx.GetSession()
		ek := getSyncEnvKey(msg.Chain, eSyncTransOwner)
		e := s.GetEnv(ek)
		if e == "" {
			return nil
		}
		s.SetEnv(ek, "")
		key, _ := hex.DecodeString(e)
		if len(key) == 0 {
			return nil
		}
		go p.syncDepend(ctx, msg.Chain, key)
	case *messages.TransactionData:
		err := processTransaction(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return err
		}
		s := ctx.GetSession()
		ek := getSyncEnvKey(msg.Chain, eSyncTransOwner)
		e := s.GetEnv(ek)
		if e == "" {
			return nil
		}
		s.SetEnv(ek, "")
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

	if !core.IsExistBlock(chain, key) {
		if !needDownload(chain, key) {
			return
		}
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
			ctx.Reply(&messages.ReqBlock{Chain: chain, Key: key})
			return
		}
		rel = ReadBlockReliability(chain, key)
	}
	id := core.GetLastBlockIndex(chain)
	if id > rel.Index+acceptOldID {
		rel.Ready = true
		SaveBlockReliability(chain, rel.Key[:], rel)
		newKey := GetSyncBlock(chain, rel.Index+1)
		if len(newKey) > 0 {
			// log.Printf("start next SyncBlock,chain:%d,key:%x,next:%x\n", chain, key, newKey)
			go p.syncDepend(ctx, chain, newKey)
		} else {
			// log.Printf("stop sync,not next SyncBlock,chain:%d,key:%x,next:%d\n", chain, key, rel.Index+1)
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
	if !rel.TransListHash.Empty() {
		transList := GetTransList(chain, key)
		if len(transList) == 0 {
			data := core.ReadTransList(chain, rel.TransListHash[:])
			transList = core.ParseTransList(data)
			if len(transList) == 0 {
				e := hex.EncodeToString(key)
				ctx.GetSession().SetEnv(getSyncEnvKey(chain, eSyncTransOwner), e)
				// log.Printf("syncDepend transList,chain:%d,index:%d,list:%x\n", chain, rel.Index, rel.TransListHash[:])
				ctx.Reply(&messages.ReqTransList{Chain: chain, Key: rel.TransListHash[:]})
				return
			}
			SaveTransList(chain, key, transList)
		}
		for _, it := range transList {
			if !needDownload(chain, it[:]) {
				continue
			}
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
	}

	rel.Recalculation(chain)
	rel.Ready = true
	SaveBlockReliability(chain, rel.Key[:], rel)

	newKey := GetSyncBlock(chain, rel.Index+1)
	if len(newKey) > 0 {
		// log.Printf("start next SyncBlock,chain:%d,key:%x,next:%x\n", chain, key, newKey)
		go p.syncDepend(ctx, chain, newKey)
	} else {
		// log.Printf("stop sync,not next SyncBlock,chain:%d,key:%x,next:%d\n", chain, key, rel.Index+1)
		if rel.Time+10*tMinute < getCoreTimeNow() {
			ctx.Reply(&messages.ReqBlockInfo{Chain: chain, Index: rel.Index + 10})
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
	index := core.GetLastBlockIndex(chain)
	hp := getBlockLock(chain, rel.Key[:])
	if hp == 0 {
		hp = rel.HashPower
	}
	setIDBlocks(chain, rel.Index, rel.Key, rel.HashPower)
	for rel.Index >= index && rel.Index > 0 {
		// log.Printf("updateBLN,chain:%d,index:%d,next:%x\n", chain, rel.Index, rel.Previous)
		if rel.Previous.Empty() {
			break
		}
		rst := setBlockLock(chain, rel.Previous[:], hp)
		setIDBlocks(chain, rel.Index-1, rel.Previous, 1)
		if !rst {
			// log.Println("fail to setBlockLock,", rst)
			break
		}
		rel = ReadBlockReliability(chain, rel.Previous[:])
	}
}
