package app

import (
	core "github.com/govm-net/govm/core/test_data/zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
)

type tApp struct{}

func run(user, in []byte, cost uint64) {
	core.Event(tApp{}, "run1", user, in)
	core.Event(tApp{}, "run", user, in)
	core.Event(tApp{}, "run", user, in)
}
