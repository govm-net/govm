package handler

import (
	"encoding/json"
	"fmt"
	"github.com/govm-net/govm/event"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/govm-net/govm/conf"
	core "github.com/govm-net/govm/core"
	"github.com/govm-net/govm/database"
	"github.com/govm-net/govm/messages"
	"github.com/lengzhao/libp2p"
	"github.com/lengzhao/libp2p/plugins"
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
	Nodes     map[string]libp2p.Session
}

type addressInfo struct {
	timeout int64
	count   int
}

const keyMyAddr = "report_my_address"
const keyReportInfo = "report_my_info"
const keyNodeInfo = "node_info"

var (
	// SelfAddress node address
	SelfAddress string
	// NodesCount nodes count
	NodesCount   int
	timeOfReport int64
	timeString   string
	startTime    int64
	minerNum     int
)

// GetNodes get p2p nodes
var GetNodes func() map[string]string

// Startup is called only once when the plugin is loaded
func (p *NATTPlugin) Startup(n libp2p.Network) {
	p.network = n
	p.peers = make(map[string]int)
	p.addrs = database.NewLRUCache(1000)
	u, _ := url.Parse(n.GetAddress())
	p.sid = u.User.Username()
	SelfAddress = n.GetAddress()
	p.Nodes = make(map[string]libp2p.Session)
	startTime = time.Now().Unix()

	event.RegisterConsumer(func(m event.Message) error {
		switch msg := m.(type) {
		case *messages.MinerInfo:
			minerNum = msg.Total
		}
		return nil
	})
	GetNodes = p.GetNodes
	go p.connectNodes()
}

// PeerConnect peer connect
func (p *NATTPlugin) PeerConnect(s libp2p.Session) {
	peer := s.GetPeerAddr()
	user := peer.User()
	p.mu.Lock()
	id := s.GetEnv(libp2p.EnvConnectID)
	p.Nodes[id] = s
	p.peers[user] = p.peers[user] + 1
	if p.peers[user] == 1 {
		NodesCount++
	}
	p.mu.Unlock()
}

// PeerDisconnect peer disconnect
func (p *NATTPlugin) PeerDisconnect(s libp2p.Session) {
	peer := s.GetPeerAddr()
	user := peer.User()
	p.mu.Lock()
	ov := p.peers[user]
	if ov > 1 {
		p.peers[user] = ov - 1
	} else {
		delete(p.peers, user)
		NodesCount--
		if NodesCount == 0 {
			go p.connectNodes()
		}
	}
	id := s.GetEnv(libp2p.EnvConnectID)
	delete(p.Nodes, id)
	p.mu.Unlock()
}

// GetNodes get nodes
func (p *NATTPlugin) GetNodes() map[string]string {
	out := make(map[string]string)
	nid := make(map[string]bool)
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, session := range p.Nodes {
		uid := session.GetPeerAddr().User()
		if nid[uid] {
			continue
		}
		nid[uid] = true
		addr := session.GetPeerAddr().String()
		out[addr] = session.GetEnv(keyNodeInfo)
		if len(out) >= 20 {
			break
		}
	}
	return out
}

func (p *NATTPlugin) connectNodes() {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/nodes", conf.GetConf().TrustedServer))
	if err != nil {
		log.Println("NodeNumber=0,fail to get node list,", err)
		return
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Println("NodeNumber=0,fail to get node list,", string(data))
		return
	}
	if len(data) == 0 {
		return
	}
	var peers map[string]interface{}
	json.Unmarshal(data, &peers)
	for k := range peers {
		s, err := p.network.NewSession(k)
		if err != nil {
			continue
		}
		s.Send(plugins.Ping{IsServer: s.GetSelfAddr().IsServer()})
	}
}

// Receive receive message
func (p *NATTPlugin) Receive(ctx libp2p.Event) error {
	now := time.Now().Unix()
	if timeOfReport+120 < now {
		timeOfReport = now
		timeString = fmt.Sprintf("%d", timeOfReport)
	}
	session := ctx.GetSession()
	if session.GetEnv(keyReportInfo) != timeString {
		session.SetEnv(keyReportInfo, timeString)
		info := messages.NodeInfo{}
		info.Version = conf.Version
		info.NodesConnected = NodesCount
		info.RunTime = startTime
		info.Height = core.GetLastBlockIndex(1)
		info.MinersConnected = minerNum
		ctx.Reply(&info)
	}

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
	case *messages.NodeInfo:
		data, _ := json.Marshal(msg)
		ctx.GetSession().SetEnv(keyNodeInfo, string(data))
	}
	return nil
}
