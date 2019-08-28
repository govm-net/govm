package event

import (
	"log"
)

// Message event messages.
type Message interface {
	String() string
}

// BaseMsg base message
type BaseMsg struct {
	Info string
}

func (m *BaseMsg) String() string {
	return m.Info
}

// Consumer message Consumer
type Consumer func(Message) error

var cbFuncs []Consumer

func init() {
	cbFuncs = make([]Consumer, 0)
}

// RegisterConsumer register consumer
func RegisterConsumer(cb Consumer) {
	if cb == nil {
		return
	}
	cbFuncs = append(cbFuncs, cb)
}

// Send send message
func Send(msg Message) error {
	defer func() {
		recover()
	}()
	for _, cb := range cbFuncs {
		err := cb(msg)
		if err != nil {
			log.Println("fail to process message:", msg, cb, err)
			return err
		}
	}
	return nil
}
