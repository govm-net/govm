package depend

import (
	"log"
	"testing"
)

type Elem struct {
	Key string
}

func (e Elem) GetKey() string {
	return e.Key
}

func TestManager_Insert(t *testing.T) {
	m := New()
	m.Insert(Elem{"a"}, nil)
	m.Insert(Elem{"b"}, Elem{"a"})
	m.print()
	m.Insert(Elem{"c"}, Elem{"a"})
	m.print()
	m.Insert(Elem{"e"}, Elem{"f"})
	m.print()
	m.Insert(Elem{"s"}, Elem{"a"})
	m.print()
	m.Insert(Elem{"a"}, Elem{"y"})
	m.print()
	v := m.Pop()
	log.Println("pop:", v)
	m.print()
	v = m.Pop()
	log.Println("pop:", v)
	m.print()
	v = m.Pop()
	log.Println("pop:", v)
	m.print()
	v = m.Pop()
	log.Println("pop:", v)
	m.print()
	v = m.Pop()
	log.Println("pop:", v)
	v = m.Pop()
	log.Println("pop:", v)
	m.print()
	m.Pop()
	m.print()
	m.Pop()
	m.Pop()
	m.print()
	t.Error("aaaaaa")
}
