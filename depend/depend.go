package depend

import (
	"container/list"
	"encoding/hex"
	"log"
	"sync"
)

// IElement the element of depend
type IElement interface {
	GetKey() string
}

// Manager depend manager
type Manager struct {
	mu    sync.Mutex
	nodes map[string]*list.Element
	queue *list.List
}

// New new manager
func New() *Manager {
	out := new(Manager)
	out.nodes = make(map[string]*list.Element)
	out.queue = list.New()
	return out
}

// Insert 将element添加到mark之后，如果mark不存在，则将mark添加到最后。如果value已经存在，不做校验，将会有2个value存在
func (m *Manager) Insert(element, mark IElement) {
	if element == nil {
		log.Println("try to insert nil element")
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := element.GetKey()
	if mark == nil {
		m.nodes[key] = m.queue.PushBack(element)
		log.Println("depend,insert:", key, ",mark:nil")
		return
	}
	markKey := mark.GetKey()
	e, ok := m.nodes[markKey]
	if !ok {
		e, ok = m.nodes[key]
		if ok {
			m.nodes[markKey] = m.queue.InsertBefore(mark, e)
			return
		}
		e = m.queue.PushBack(mark)
		m.nodes[markKey] = e
	}
	m.nodes[key] = m.queue.InsertAfter(element, e)
	log.Println("depend,insert:", key, ",mark:", markKey)
}

// Pop pop first element
func (m *Manager) Pop() IElement {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.queue.Front()
	if e == nil {
		return nil
	}
	v := m.queue.Remove(e)
	out := v.(IElement)
	delete(m.nodes, out.GetKey())

	return out
}

// First return the key of first element
func (m *Manager) First() IElement {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.queue.Front()
	if e == nil {
		return nil
	}
	return e.Value.(IElement)
}

// Delete delete element by key
func (m *Manager) Delete(key string) IElement {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.nodes[key]
	if !ok {
		return nil
	}
	v := m.queue.Remove(e)
	out := v.(IElement)
	delete(m.nodes, key)

	return out
}

// Exist return true when the value exist
func (m *Manager) Exist(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.nodes[key]
	return ok
}

// Len return len of queue
func (m *Manager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queue.Len()
}

func (m *Manager) print() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for e := m.queue.Front(); e != nil; e = e.Next() {
		log.Print(e.Value, " ")
	}
	log.Print("\n")
}

type chainMgr struct {
	mu   sync.Mutex
	mgrs map[uint64]*Manager
}

var mgrs chainMgr

func init() {
	mgrs.mgrs = make(map[uint64]*Manager)
}

// getMgrOfChain get manager of chain
func getMgrOfChain(chain uint64) *Manager {
	mgrs.mu.Lock()
	defer mgrs.mu.Unlock()
	mgr, ok := mgrs.mgrs[chain]
	if !ok {
		mgr = New()
		mgrs.mgrs[chain] = mgr
	}
	return mgr
}

// Insert insert depend
func Insert(chain uint64, element, mark IElement) {
	mgr := getMgrOfChain(chain)
	mgr.Insert(element, mark)
}

// Pop pop the first element
func Pop(chain uint64) IElement {
	mgr := getMgrOfChain(chain)
	return mgr.Pop()
}

// First return the key of first element
func First(chain uint64) IElement {
	mgr := getMgrOfChain(chain)
	return mgr.First()
}

// Delete delete the element
func Delete(chain uint64, key string) IElement {
	mgr := getMgrOfChain(chain)
	return mgr.Delete(key)
}

// Exist if exist the key,return true
func Exist(chain uint64, key string) bool {
	mgr := getMgrOfChain(chain)
	return mgr.Exist(key)
}

// Len return the length of elements
func Len(chain uint64) int {
	mgr := getMgrOfChain(chain)
	return mgr.Len()
}

// DfElement default element struct
type DfElement struct {
	Key     []byte
	IsBlock bool
	hexKey  string
}

// GetKey get key
func (e *DfElement) GetKey() string {
	if e.hexKey == "" {
		e.hexKey = hex.EncodeToString(e.Key)
	}

	return e.hexKey

}
