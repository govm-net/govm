package handler

import (
	"errors"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/event"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/libp2p"
	"github.com/lengzhao/libp2p/plugins"
	"log"
)

// InternalPlugin process p2p message
type InternalPlugin struct {
	libp2p.Plugin
	network libp2p.Network
}

// Startup is called only once when the plugin is loaded
func (p *InternalPlugin) Startup(n libp2p.Network) {
	p.network = n
	event.RegisterConsumer(p.event)
}

// Event internal event
func (p *InternalPlugin) event(m event.Message) error {
	switch msg := m.(type) {
	case *messages.NewTransaction:
		if core.IsExistTransaction(msg.Chain, msg.Key) {
			log.Printf("[event]trans is exist,chain:%d,key:%x\n", msg.Chain, msg.Key)
			return nil
		}
		trans := core.DecodeTrans(msg.Data)
		if trans == nil {
			return errors.New("error transaction")
		}
		core.WriteTransaction(trans.Chain, msg.Data)
		addTrans(msg.Key[:], trans.User[:], trans.Chain, trans.Time, trans.Energy, uint64(len(msg.Data)))
		m := &messages.TransactionInfo{Chain: trans.Chain, Key: msg.Key[:]}
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: m})
		log.Printf("new trans:%x,chain:%d,ops:%d\n", msg.Key[:], trans.Chain, trans.Ops)
		return nil
	case *messages.Mine:
		id := core.GetLastBlockIndex(msg.Chain)
		if id == 0 {
			return errors.New("not exist the chain")
		}
		log.Println("do mine:", msg.Chain)
		m := &messages.ReqBlockInfo{Chain: msg.Chain, Index: id}
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: m})
		doMine(msg.Chain)
		return nil
	case *messages.Rollback:
		log.Println("rollback block")
		ib := core.IDBlocks{}
		for i := 0; i < 100; i++ {
			core.SaveIDBlocks(msg.Chain, msg.Index+uint64(i), ib)
		}
		ek := core.Hash{}
		er := core.TReliability{}
		core.SaveBlockReliability(msg.Chain, ek[:], er)
		m := &messages.ReqBlockInfo{Chain: msg.Chain, Index: msg.Index}
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: m})
		return dbRollBack(msg.Chain, msg.Index, msg.Key)
	case *messages.NewNode:
		session, err := p.network.NewSession(msg.Peer)
		if err != nil {
			return err
		}
		return session.Send(plugins.Ping{})
	}
	return nil
}
