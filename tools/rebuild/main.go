package main

import (
	"fmt"
	"path"

	"github.com/govm-net/govm/runtime"
)

func rebuild(chain uint64) {
	ch := fmt.Sprintf("chain%d", chain)
	dir := path.Join("app", ch)
	err := runtime.RebuildApp(chain, dir)
	fmt.Println("result,chain:", chain, err)
	if err != nil {
		return
	}
	rebuild(chain * 2)
	rebuild(chain*2 + 1)
}

func main() {
	rebuild(1)
}
