package a1000000000000000000000000000000000000000000000000000000000000000

import core "github.com/govm-net/govm/tools/debugger/zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

type tApp struct {
}

func run(user, in []byte, cost uint64) {
	core.Event(tApp{}, "start_app", user, in)
	u := core.Address{}
	core.Decode(0, user, &u)
	core.TransferAccounts(tApp{}, u, cost)
}
