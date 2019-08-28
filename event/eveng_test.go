package event

import (
	"fmt"
	"testing"
)

type Msg1 struct {
	BaseMsg
}
type Msg2 struct {
	BaseMsg
}
type Msg3 struct {
	BaseMsg
}

func TestSend(t *testing.T) {
	var cb = func(msg Message) error {
		switch v := msg.(type) {
		default:
			return fmt.Errorf("%T", v)
		}
		//return nil
	}
	RegisterConsumer(cb)
	err := Send(new(Msg1))
	if err.Error() != "*event.Msg1" {
		t.Error(err)
	}
	err = Send(new(Msg2))
	if err.Error() != "*event.Msg2" {
		t.Error(err)
	}
	err = Send(new(Msg3))
	if err.Error() != "*event.Msg3" {
		t.Error(err)
	}
}
