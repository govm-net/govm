package api

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/govm-net/govm/conf"
	"github.com/govm-net/govm/event"
	"github.com/govm-net/govm/messages"
	"github.com/govm-net/govm/runtime"
	"github.com/govm-net/govm/wallet"
	"golang.org/x/net/websocket"
)

var minerNum int

type wsConn struct {
	ws    *websocket.Conn
	send  chan []byte
	chain uint64
	peer  string
}

type hub struct {
	clients    map[string]*wsConn
	broadcast  chan *messages.BlockForMining
	register   chan *wsConn
	unregister chan *wsConn
	content    *messages.BlockForMining
	lastTime   int64
}

func (h *hub) run() {
	connLimit := conf.GetConf().OneConnPerMiner
	var index int64
	for {
		select {
		case c := <-h.register:
			if c.peer == "" {
				break
			}
			index++
			if !connLimit {
				c.peer = fmt.Sprintf("id%d", index)
			} else {
				_, ok := h.clients[c.peer]
				if ok {
					// log.Println("exist peer:", c.peer)
					c.peer = ""
					close(c.send)
					break
				}
			}
			stat.Add("ws_connect", 1)
			// log.Println("ws connect number:", len(h.clients), c.peer)
			h.clients[c.peer] = c
			minerNum = len(h.clients)
			m := messages.MinerInfo{}
			m.Total = minerNum
			m.ID = c.peer
			m.Online = true
			event.Send(&m)

			if h.content == nil {
				break
			}

			if h.content.Chain != c.chain && c.chain != 0 {
				break
			}
			c.send <- h.content.Data

		case c := <-h.unregister:
			_, ok := h.clients[c.peer]
			if ok {
				delete(h.clients, c.peer)
				c.peer = ""
				close(c.send)
			}
			minerNum = len(h.clients)
			m := messages.MinerInfo{}
			m.Total = minerNum
			m.ID = c.peer
			m.Online = false
			event.Send(&m)

		case m := <-h.broadcast:
			if m == nil || len(m.Data) == 0 {
				break
			}
			if h.content != nil && bytes.Compare(h.content.Data, m.Data) == 0 {
				// log.Println("same block for mining")
				break
			}
			stat.Add("ws_broadcast_package", 1)
			h.content = m
			h.broadcastMessage()
		}
	}
}

func (h *hub) broadcastMessage() {
	if h.content == nil {
		return
	}

	for _, c := range h.clients {
		if c.chain != h.content.Chain && c.chain != 0 {
			continue
		}
		select {
		case c.send <- h.content.Data:
			break

		// We can't reach the client
		default:
			delete(h.clients, c.peer)
			c.peer = ""
			close(c.send)
		}
	}
}

var h = hub{
	broadcast:  make(chan *messages.BlockForMining, 10),
	register:   make(chan *wsConn, 10),
	unregister: make(chan *wsConn),
	clients:    make(map[string]*wsConn),
	content:    nil,
}

func init() {
	event.RegisterConsumer(func(m event.Message) error {
		switch msg := m.(type) {
		case *messages.BlockForMining:
			h.broadcast <- msg
		}
		return nil
	})
	go h.run()
}

type wsHead struct {
	Address [24]byte
	Time    int64
}

// WSBlockForMining ws
func WSBlockForMining(ws *websocket.Conn) {
	vars := mux.Vars(ws.Request())
	chain, err := strconv.ParseUint(vars["chain"], 10, 64)
	if err != nil || chain > 100 {
		return
	}
	limit := conf.GetConf().MinerConnLimit
	if limit > 0 && minerNum > limit {
		return
	}
	// fmt.Println("remote:", ws.Request().RemoteAddr)
	c := &wsConn{
		send:  make(chan []byte, 1),
		ws:    ws,
		chain: chain,
	}

	msg := make([]byte, 1000)
	n, err := ws.Read(msg)
	if err != nil {
		return
	}
	info := wsHead{}
	msg = msg[:n]
	n = runtime.Decode(msg, &info)
	sign := msg[n:]
	rst := wallet.Recover(info.Address[:], sign, msg[:n])
	if !rst {
		// log.Println("error sign")
		ws.Write([]byte("error sign"))
		return
	}
	c.peer = fmt.Sprintf("c%d_k%x", chain, info.Address)

	go func() {
		msg := make([]byte, 10)
		for {
			_, err := ws.Read(msg)
			if err != nil {
				break
			}
		}
		h.unregister <- c
	}()

	select {
	case h.register <- c:
	default:
		return
	}

	for {
		data, ok := <-c.send
		if !ok {
			break
		}
		if len(data) == 0 {
			continue
		}
		stat.Add("ws_send", 1)
		_, err := ws.Write(data)
		if err != nil {
			break
		}
	}

	ws.Close()
}
