package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/govm-net/govm/conf"
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
}

type addressInfo struct {
	timeout int64
	count   int
}

const keyMyAddr = "report_my_address"
const keyNodeInfo = "report_my_info"

var (
	// SelfAddress node address
	SelfAddress string
	// NodesCount nodes number
	NodesCount int
	// Nodes p2p nodes
	Nodes        map[string]string
	timeOfReport int64
	timeString   string
	startTime    int64
)

// Startup is called only once when the plugin is loaded
func (p *NATTPlugin) Startup(n libp2p.Network) {
	p.network = n
	p.peers = make(map[string]int)
	p.addrs = database.NewLRUCache(1000)
	u, _ := url.Parse(n.GetAddress())
	p.sid = u.User.Username()
	SelfAddress = n.GetAddress()
	Nodes = make(map[string]string)
	startTime = time.Now().Unix()
	go p.connectNodes()
}

// PeerConnect peer connect
func (p *NATTPlugin) PeerConnect(s libp2p.Session) {
	peer := s.GetPeerAddr()
	user := peer.User()
	p.mu.Lock()
	p.peers[user] = p.peers[user] + 1
	if p.peers[user] == 1 {
		NodesCount++
		if len(Nodes) < 20 {
			Nodes[peer.String()] = "nil"
		}
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
	delete(Nodes, s.GetPeerAddr().String())
	p.mu.Unlock()
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
	if timeOfReport+120 > now {
		timeOfReport = now
		timeString = fmt.Sprintf("%d", timeOfReport)
	}
	session := ctx.GetSession()
	if session.GetEnv(keyNodeInfo) != timeString {
		session.SetEnv(keyNodeInfo, timeString)
		info := messages.NodeInfo{}
		info.Version = conf.Version
		info.NodesConnected = NodesCount
		info.RunTime = startTime
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
		peer := ctx.GetSession().GetPeerAddr().String()
		p.mu.Lock()
		_, ok := Nodes[peer]
		if ok {
			data, _ := json.Marshal(msg)
			Nodes[peer] = string(data)
		}
		p.mu.Unlock()
	}
	return nil
}
