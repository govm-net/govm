package handler

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	core "github.com/lengzhao/govm/core"
	"github.com/lengzhao/govm/event"
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/libp2p"
	"github.com/lengzhao/libp2p/plugins"
)

// InternalPlugin process p2p message
type InternalPlugin struct {
	libp2p.Plugin
	network libp2p.Network
	mu      sync.Mutex
	reconn  map[string]string
	miners  map[string]libp2p.Session
}

const (
	reconnNum = 15
)

// Nodes p2p nodes
var Nodes map[string]bool

// Startup is called only once when the plugin is loaded
func (p *InternalPlugin) Startup(n libp2p.Network) {
	Init()

	p.network = n
	p.reconn = make(map[string]string)
	p.miners = make(map[string]libp2p.Session)
	Nodes = make(map[string]bool)
	event.RegisterConsumer(p.event)
	time.AfterFunc(time.Minute*2, p.timeout)
}

// Cleanup plugin uninstall
func (p *InternalPlugin) Cleanup(n libp2p.Network) {
	ldb.Close()
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
	cid := s.GetEnv(libp2p.EnvConnectID)
	addr := peer.String()
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(Nodes, addr)
	delete(p.miners, cid)
	if peer.IsServer() {
		if len(p.reconn) > reconnNum {
			return
		}
		p.reconn[peer.User()] = peer.String()
	}
}

// Receive receive message
func (p *InternalPlugin) Receive(ctx libp2p.Event) error {
	switch ctx.GetMessage().(type) {
	case *messages.Miner:
		s := ctx.GetSession()
		ip, _, _ := net.SplitHostPort(s.GetPeerAddr().Host())
		if ip != "127.0.0.1" && ip != "::1" {
			return nil
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		// not close this session
		s.SetEnv("inDHT", "true")
		cid := s.GetEnv(libp2p.EnvConnectID)
		p.miners[cid] = s
	}
	return nil
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
		info := messages.ReceiveTrans{}
		info.Chain = msg.Chain
		info.Key = msg.Key
		event.Send(&info)
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
	case *messages.RawData:
		if msg.IsTrans {
			err := processTransaction(msg.Chain, msg.Key, msg.Data)
			if err != nil {
				return err
			}
			if !msg.Broadcast {
				return nil
			}
			trans := core.DecodeTrans(msg.Data)
			m := &messages.TransactionInfo{}
			m.Chain = msg.Chain
			m.Key = msg.Key
			m.Time = trans.Time
			m.User = trans.User[:]
			p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: m})
			return nil
		}
		err := processBlock(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			log.Printf("error block,chain:%d,key:%x,err:%s\n",
				msg.Chain, msg.Key, err)
			return err
		}
		if msg.LockNum > 0 {
			ldb.LSet(msg.Chain, ldbBlockLocked, msg.Key, runtime.Encode(msg.LockNum))
		}
		rel := ReadBlockReliability(msg.Chain, msg.Key)
		if rel.Index == 0 || rel.Key.Empty() {
			return nil
		}
		if msg.LockNum > 0 {
			setBlockToIDBlocks(msg.Chain, rel.Index, rel.Key, rel.HashPower)
		}

		if !msg.Broadcast {
			return nil
		}
		info := messages.BlockInfo{}
		info.Chain = msg.Chain
		info.Index = rel.Index
		info.Key = rel.Key[:]
		info.HashPower = rel.HashPower
		info.PreKey = rel.Previous[:]
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: &info})

	case *messages.ReceiveTrans:
		var nodes []libp2p.Session
		p.mu.Lock()
		for _, s := range p.miners {
			nodes = append(nodes, s)
		}
		p.mu.Unlock()
		for _, s := range nodes {
			m := &messages.TransactionInfo{}
			m.Chain = msg.Chain
			m.Key = msg.Key
			s.Send(m)
		}
	}
	return nil
}
