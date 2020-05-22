package handler

import (
	"github.com/govm-net/govm/database"
	"github.com/lengzhao/libp2p"
	"github.com/lengzhao/libp2p/plugins"
	"net/url"
	"sync"
	"time"
)

// NATTPlugin nat traversal
type NATTPlugin struct {
	libp2p.Plugin
	network   libp2p.Network
	mu        sync.Mutex
	peers     map[string]int
	addrs     *database.LRUCache
	myAddress string
	info      addressInfo
	sid       string
}

type addressInfo struct {
	timeout int64
	count   int
}

const keyMyAddr = "report_my_address"

// SelfAddress node address
var SelfAddress string

// Startup is called only once when the plugin is loaded
func (p *NATTPlugin) Startup(n libp2p.Network) {
	p.network = n
	p.peers = make(map[string]int)
	p.addrs = database.NewLRUCache(1000)
	u, _ := url.Parse(n.GetAddress())
	p.sid = u.User.Username()
}

// PeerConnect peer connect
func (p *NATTPlugin) PeerConnect(s libp2p.Session) {
	peer := s.GetPeerAddr().User()
	p.mu.Lock()
	p.peers[peer] = p.peers[peer] + 1
	p.mu.Unlock()
}

// PeerDisconnect peer disconnect
func (p *NATTPlugin) PeerDisconnect(s libp2p.Session) {
	peer := s.GetPeerAddr().User()
	p.mu.Lock()
	ov := p.peers[peer]
	if ov > 1 {
		p.peers[peer] = ov - 1
	} else {
		delete(p.peers, peer)
	}
	p.mu.Unlock()
}

// Receive receive message
func (p *NATTPlugin) Receive(ctx libp2p.Event) error {
	switch msg := ctx.GetMessage().(type) {
	case plugins.Pong:
		if msg.ToAddr == "" {
			return nil
		}
		if !ctx.GetSession().GetSelfAddr().IsServer() {
			return nil
		}
		u, err := url.Parse(msg.ToAddr)
		if err != nil {
			return err
		}
		if u.User.String() != p.sid {
			return nil
		}
		if ctx.GetSession().GetEnv(keyMyAddr) == "true" {
			return nil
		}
		ctx.GetSession().SetEnv(keyMyAddr, "true")
		p.mu.Lock()
		defer p.mu.Unlock()
		var info *addressInfo
		val, ok := p.addrs.Get(msg.ToAddr)
		if ok {
			info = val.(*addressInfo)
		} else {
			info = new(addressInfo)
			p.addrs.Set(msg.ToAddr, info)
		}
		now := time.Now().Unix()
		if p.info.timeout+3600 < now {
			p.info.timeout = now
			p.info.count /= 2
		}
		if info.timeout+3600 < now {
			info.count /= 2
			info.timeout = now
		}
		info.count++
		if info.count > p.info.count {
			p.info = *info
			p.info.count++
			p.myAddress = msg.ToAddr
			SelfAddress = msg.ToAddr
			// log.Println("myAddress:", p.myAddress, info.count)
		}
	case plugins.NatTraversal:
		if p.myAddress == "" {
			return nil
		}
		tu, err := url.Parse(msg.ToAddr)
		if err != nil {
			return nil
		}
		if tu.User.String() != p.sid {
			return nil
		}

		peer := ctx.GetSession().GetPeerAddr().User()
		p.mu.Lock()
		ov := p.peers[peer]
		p.mu.Unlock()
		if ov > 0 {
			return nil
		}
		if p.myAddress != msg.ToAddr {
			trav := plugins.NatTraversal{}
			trav.FromAddr = p.myAddress
			trav.ToAddr = msg.FromAddr
			ctx.Reply(trav)
		}
	}
	return nil
}
