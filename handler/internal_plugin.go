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

// Nodes p2p nodes
var Nodes map[string]bool

// Startup is called only once when the plugin is loaded
func (p *InternalPlugin) Startup(n libp2p.Network) {
	p.network = n
	p.reconn = make(map[string]string)
	Nodes = make(map[string]bool)
	event.RegisterConsumer(p.event)
	time.AfterFunc(time.Minute*2, p.timeout)
}

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
	id := peer.User()
	if id == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(Nodes) < 20 {
		Nodes[peer.String()] = peer.IsServer()
	}
	for k := range p.reconn {
		if k == id || len(p.reconn) > reconnNum-2 {
			delete(p.reconn, k)
		}
	}
}

// PeerDisconnect peer disconnect
func (p *InternalPlugin) PeerDisconnect(s libp2p.Session) {
	peer := s.GetPeerAddr()
	addr := peer.String()
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(Nodes, addr)
	if peer.IsServer() {
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
		log.Printf("create new trans:%x,\n", msg.Key[:])
		core.WriteTransaction(msg.Chain, msg.Data)
		err := core.CheckTransaction(msg.Chain, msg.Key)
		if err != nil {
			log.Printf("fail to new transaction.chain:%d,key:%x,error:%s\n", msg.Chain, msg.Key, err)
			core.DeleteTransaction(msg.Chain, msg.Key)
			return err
		}
		go func() {
			defer recover()
			trans := core.DecodeTrans(msg.Data)
			m := &messages.TransactionInfo{}
			m.Chain = msg.Chain
			m.Key = msg.Key
			m.Time = trans.Time
			m.User = trans.User[:]
			p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: m})
			processTransaction(msg.Chain, msg.Key, msg.Data)
		}()
		return nil
	case *messages.Mine:
		id := core.GetLastBlockIndex(msg.Chain)
		if id == 0 {
			return errors.New("not exist the chain")
		}
		log.Println("do mine:", msg.Chain)
		m := &messages.ReqBlockInfo{Chain: msg.Chain, Index: id}
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: m})
		go doMine(msg.Chain, true)
		return nil
	case *messages.Rollback:
		log.Println("rollback block")
		ib := IDBlocks{}
		for i := 0; i < 100; i++ {
			SaveIDBlocks(msg.Chain, msg.Index+uint64(i), ib)
		}
		m := &messages.ReqBlockInfo{Chain: msg.Chain, Index: msg.Index}
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: m})

		procMgr.mu.Lock()
		cl, ok := procMgr.Chains[msg.Chain]
		procMgr.mu.Unlock()
		if !ok {
			return errors.New("not exist chain lock")
		}

		cl <- 1
		defer func() { <-cl }()
		return dbRollBack(msg.Chain, msg.Index, msg.Key)
	case *messages.NewNode:
		session, err := p.network.NewSession(msg.Peer)
		if err != nil {
			return err
		}
		return session.Send(plugins.Ping{IsServer: session.GetSelfAddr().IsServer()})
	case *messages.OSExit:
		log.Println("exit by user:", msg.Msg)
		p.network.Close()
	}
	return nil
}
