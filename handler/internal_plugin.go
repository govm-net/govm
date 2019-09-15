package handler

import (
	"errors"
	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/event"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/libp2p"
	"github.com/lengzhao/libp2p/plugins"
	"log"
	"sync"
	"time"
)

// InternalPlugin process p2p message
type InternalPlugin struct {
	libp2p.Plugin
	network libp2p.Network
	mu      sync.Mutex
	reconn  map[string]string
}

const (
	reconnNum = 15
)

// Startup is called only once when the plugin is loaded
func (p *InternalPlugin) Startup(n libp2p.Network) {
	p.network = n
	p.reconn = make(map[string]string)
	event.RegisterConsumer(p.event)
	time.AfterFunc(time.Minute*2, p.timeout)
}

// Nodes p2p nodes
var Nodes [10]string

func (p *InternalPlugin) timeout() {
	time.AfterFunc(time.Minute*2, p.timeout)
	p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: plugins.Ping{}})
	nodes := make(map[string]string)
	p.mu.Lock()
	for k, v := range p.reconn {
		nodes[k] = v
	}
	p.mu.Unlock()
	for _, node := range nodes {
		s, err := p.network.NewSession(node)
		if err != nil {
			continue
		}
		s.Send(plugins.Ping{IsServer: s.GetSelfAddr().IsServer()})
	}
}

// PeerConnect peer connect
func (p *InternalPlugin) PeerConnect(s libp2p.Session) {
	peer := s.GetPeerAddr()

	for i, n := range Nodes {
		if n == "" {
			if peer.IsServer() {
				Nodes[i] = peer.String()
			} else {
				Nodes[i] = peer.String() + "(client)"
			}
			break
		}
	}
	id := peer.User()
	p.mu.Lock()
	defer p.mu.Unlock()
	for k := range p.reconn {
		if k == id || len(p.reconn) > reconnNum-2 {
			delete(p.reconn, k)
		}
	}
}

// PeerDisconnect peer connect
func (p *InternalPlugin) PeerDisconnect(s libp2p.Session) {
	peer := s.GetPeerAddr()
	addr := peer.String()
	if !peer.IsServer() {
		addr += "(client)"
	}
	for i, n := range Nodes {
		if n == addr {
			Nodes[i] = ""
		}
	}
	if peer.IsServer() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if len(p.reconn) > reconnNum {
			return
		}
		p.reconn[peer.User()] = peer.String()
	}
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
		m := &messages.ReqBlockInfo{Chain: msg.Chain, Index: msg.Index}
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: m})
		return dbRollBack(msg.Chain, msg.Index, msg.Key)
	case *messages.NewNode:
		session, err := p.network.NewSession(msg.Peer)
		if err != nil {
			return err
		}
		return session.Send(plugins.Ping{IsServer: session.GetSelfAddr().IsServer()})
	}
	return nil
}
