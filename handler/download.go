package handler

import (
	"github.com/lengzhao/govm/messages"
	"github.com/lengzhao/libp2p"
	"log"
	"sync"
	"time"
)

type task struct {
	msg     interface{}
	chain   uint64
	timeout int64
	from    libp2p.Session
	retry   int
}

type dlMgr struct {
	mu    sync.Mutex
	net   libp2p.Network
	tasks map[string]*task
	array []string
}

var mgr dlMgr

func initDLMgr(n libp2p.Network) {
	mgr.tasks = make(map[string]*task)
	mgr.net = n
}

func existTask(chain uint64, key string) bool {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	_, ok := mgr.tasks[key]
	return ok
}

func download(chain uint64, key string, from libp2p.Session, msg interface{}) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	t, ok := mgr.tasks[key]
	if ok {
		return
	}
	t = new(task)
	t.msg = msg
	t.from = from

	t.chain = chain
	mgr.tasks[key] = t
	mgr.array = append(mgr.array, key)
	log.Printf("new download. chain:%d,key:%s, task number:%d\n", chain, key, len(mgr.tasks))
	go doDL()
	return
}

func finishDL(key string) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	delete(mgr.tasks, key)
	go doDL()
}

func doDL() {
	var t *task
	var k string
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if len(mgr.array) == 0 {
		return
	}

	now := time.Now().Unix()
	k = mgr.array[0]
	t = mgr.tasks[k]
	if t == nil {
		mgr.array = mgr.array[1:]
		go doDL()
		return
	}
	if t.timeout >= now {
		return
	}
	if t.retry > 10 {
		mgr.array = mgr.array[1:]
		delete(mgr.tasks, k)
		log.Println("download task retry>10,", k)
		go doDL()
		return
	}
	// todo
	if t.from != nil {
		err := t.from.Send(t.msg)
		if err != nil {
			t.from = nil
		}
	} else {
		m := messages.BaseMsg{Type: messages.RandsendMsg, Msg: t.msg}
		mgr.net.SendInternalMsg(&m)
		// log.Println("SendInternalMsg:", m)
	}

	// log.Println("download task:", t.chain, k)

	t.timeout = now + 30
	t.retry++
	mgr.array = append(mgr.array[1:], k)
	go doDL()
}
