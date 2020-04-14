package a1000000000000000000000000000000000000000000000000000000000000000

import core "github.com/lengzhao/govm/tools/debugger/ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c"

type tApp struct {
}

func run(user, in []byte, cost uint64) {
	core.Event(tApp{}, "start_app", user, in)
	u := core.Address{}
	core.Decode(0, user, &u)
	core.TransferAccounts(tApp{}, u, cost)
}
