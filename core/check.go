package ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c

import (
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/lengzhao/govm/runtime"
)

// CheckTransaction check trans for mine
func CheckTransaction(chain uint64, tKey []byte) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			log.Println("something error,", e)
			log.Println(string(debug.Stack()))
			err = fmt.Errorf("recover:%s", e)
		}
	}()
	if chain == 0 {
		return errors.New("not support,chain == 0")
	}

	if runtime.DbExist(dbTransInfo{}, chain, tKey) {
		return errors.New("transaction is exist")
	}
	var proc processer
	proc.initEnv(chain, []byte("testmode"))
	runt := proc.iRuntime.(*runtime.TRuntime)
	runt.SetTestMode()
	key := proc.pLogBlockInfo.read(chain, proc.Encode(0, proc.ID-1))
	stream := proc.pLogBlockInfo.read(chain, key[:])
	if len(stream) == 0 {
		return fmt.Errorf("fail to read block info:%x", key)
	}
	block := BlockInfo{}
	proc.Decode(0, stream, &block)

	h := Hash{}
	runtime.Decode(tKey, &h)
	start := time.Now().Unix()
	proc.processTransaction(block, h)
	inv := GetBlockInterval(chain) + 5000
	sub := time.Now().Unix() - start
	if uint64(sub) > inv/5000 {
		return fmt.Errorf("timeout:%d", sub)
	}
	return nil
}

// CheckTransList check trans list for mine
func CheckTransList(chain uint64, factory func(uint64) Hash) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			log.Println("CheckTransList error:", e)
			err = fmt.Errorf("recover:%s", e)
		}
		// log.Printf("CheckTransList input:%d,out:%d", len(keys), len(out))
	}()
	var proc processer
	proc.initEnv(chain, []byte("testmode"))
	runt := proc.iRuntime.(*runtime.TRuntime)
	runt.SetTestMode()
	key := proc.pLogBlockInfo.read(chain, proc.Encode(0, proc.ID-1))
	stream := proc.pLogBlockInfo.read(chain, key[:])
	if len(stream) == 0 {
		return nil
	}
	block := BlockInfo{}
	proc.Decode(0, stream, &block)

	for {
		k := factory(chain)
		if k.Empty() {
			return nil
		}
		proc.processTransaction(block, k)
	}
}
