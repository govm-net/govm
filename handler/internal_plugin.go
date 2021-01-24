package handler

import (
	"errors"
	"expvar"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	core "github.com/govm-net/govm/core"
	"github.com/govm-net/govm/event"
	"github.com/govm-net/govm/messages"
	"github.com/lengzhao/libp2p"
	"github.com/lengzhao/libp2p/plugins"
)

// InternalPlugin process p2p message
type InternalPlugin struct {
	libp2p.Plugin
	network libp2p.Network
	mu      sync.Mutex
	reconn  map[string]string
}

const (
	reconnNum   = 15
	connectTime = "conn_start_time"
)

var reconnStat = expvar.NewMap("reconnect")

// Startup is called only once when the plugin is loaded
func (p *InternalPlugin) Startup(n libp2p.Network) {
	Init()
	p.network = n
	p.reconn = make(map[string]string)
	event.RegisterConsumer(p.event)
	time.AfterFunc(time.Minute*2, p.timeout)
}

// Cleanup plugin uninstall
func (p *InternalPlugin) Cleanup(n libp2p.Network) {
	ldb.Close()
}

func (p *InternalPlugin) timeout() {
	time.AfterFunc(time.Minute*5, p.timeout)
	p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.RandsendMsg, Msg: plugins.Ping{}})
	nodes := make(map[string]string)
	var i int
	p.mu.Lock()
	for k, v := range p.reconn {
		if i == 0 {
			delete(p.reconn, k)
		}
		i++
		nodes[k] = v
	}
	p.mu.Unlock()
	for k, node := range nodes {
		s, err := p.network.NewSession(node)
		if err != nil {
			continue
		}
		if s.GetEnv(keyOldConn) != "" {
			p.mu.Lock()
			delete(p.reconn, k)
			p.mu.Unlock()
			continue
		}
		reconnStat.Add("sendPing", 1)
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
	now := time.Now().Unix()
	s.SetEnv(connectTime, fmt.Sprintf("%d", now))
	p.mu.Lock()
	defer p.mu.Unlock()
	for k := range p.reconn {
		if k == id || len(p.reconn) > reconnNum {
			delete(p.reconn, k)
		}
	}
}

// PeerDisconnect peer disconnect
func (p *InternalPlugin) PeerDisconnect(s libp2p.Session) {
	peer := s.GetPeerAddr()
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer.IsServer() {
		if len(p.reconn) > reconnNum {
			return
		}
		var t int64
		tStr := s.GetEnv(connectTime)
		if tStr != "" {
			t, _ = strconv.ParseInt(tStr, 10, 64)
		}
		if t+60 > time.Now().Unix() {
			return
		}
		if s.GetEnv("inDHT") == "true" {
			reconnStat.Add("add_node", 1)
			p.reconn[peer.User()] = peer.String()
		}
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
		err := core.WriteTransaction(msg.Chain, msg.Data)
		if err != nil {
			log.Println("fail to write transaction:", err)
			return err
		}
		err = core.CheckTransaction(msg.Chain, msg.Key)
		if err != nil {
			log.Printf("fail to new transaction.chain:%d,key:%x,error:%s\n", msg.Chain, msg.Key, err)
			core.DeleteTransaction(msg.Chain, msg.Key)
			return err
		}

		go func() {
			defer recover()

			stream := ldb.LGet(msg.Chain, ldbBroadcastTrans, msg.Key)
			if len(stream) > 0 {
				return
			}
			ldb.LSet(msg.Chain, ldbBroadcastTrans, msg.Key, []byte{1})

			trans := core.DecodeTrans(msg.Data)
			m := &messages.TransactionInfo{}
			m.Chain = msg.Chain
			m.Key = msg.Key
			m.Time = trans.Time
			m.User = trans.User[:]
			p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: m})
			err := processTransaction(msg.Chain, msg.Key, msg.Data)
			if err != nil {
				log.Printf("result of trans:%x,err :%s\n", msg.Key, err)
			}
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
		go doMining(msg.Chain)
		return nil
	case *messages.NewNode:
		session, err := p.network.NewSession(msg.Peer)
		if err != nil {
			return err
		}
		return session.Send(plugins.Ping{IsServer: session.GetSelfAddr().IsServer()})
	case *messages.RawData:
		if !msg.IsTrans {
			return fmt.Errorf("not support")
		}

		if core.IsExistTransaction(msg.Chain, msg.Key) {
			return nil
		}

		err := processTransaction(msg.Chain, msg.Key, msg.Data)
		if err != nil {
			return err
		}

		err = core.CheckTransaction(msg.Chain, msg.Key)
		if err != nil {
			return err
		}

		stream := ldb.LGet(msg.Chain, ldbBroadcastTrans, msg.Key)
		if len(stream) > 0 {
			return nil
		}
		ldb.LSet(msg.Chain, ldbBroadcastTrans, msg.Key, []byte{1})

		trans := core.DecodeTrans(msg.Data)
		m := &messages.TransactionInfo{}
		m.Chain = msg.Chain
		m.Key = msg.Key
		m.Time = trans.Time
		m.User = trans.User[:]
		p.network.SendInternalMsg(&messages.BaseMsg{Type: messages.BroadcastMsg, Msg: m})
		return nil
	}
	return nil
}
