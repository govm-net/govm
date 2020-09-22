package zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f

import (
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/govm-net/govm/runtime"
)

// MaxGuerdon MaxGuerdon
const MaxGuerdon = maxGuerdon

// CheckTransaction check trans for mine
func CheckTransaction(chain uint64, tKey []byte) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			log.Println("something error,", e)
			log.Println(string(debug.Stack()))
			err = fmt.Errorf("%s", e)
		}
	}()
	if chain == 0 {
		return errors.New("not support,chain == 0")
	}

	if runtime.DbExist(dbTransInfo{}, chain, tKey) {
		return errors.New("transaction is exist")
	}
	c := dfMgr.alloc()
	if c == nil {
		return errors.New("fail to get new client for check")
	}
	defer dfMgr.free(c)
	flag := runtime.Encode(time.Now().UnixNano())
	flag = append(flag, tKey...)
	at, ad := c.GetAddress()
	err = c.OpenFlag(chain, flag)
	if err != nil {
		log.Println("fail to open Flag,", err, at, ad)
		f := c.GetLastFlag(chain)
		c.Cancel(chain, f)
		return err
	}
	defer c.Cancel(chain, flag)

	var proc processer
	proc.initEnv(chain, flag, at, ad)
	runt := proc.iRuntime.(*runtime.TRuntime)
	runt.SetMode("check")
	defer runt.Close()
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
			err = fmt.Errorf("%s", e)
		}
		// log.Printf("CheckTransList input:%d,out:%d", len(keys), len(out))
	}()
	c := dfMgr.alloc()
	if c == nil {
		return errors.New("fail to get new client for check")
	}
	defer dfMgr.free(c)
	at, ad := c.GetAddress()
	flag := runtime.Encode(time.Now().UnixNano())
	err = c.OpenFlag(chain, flag)
	if err != nil {
		log.Println("fail to open Flag,", err, at, ad)
		f := c.GetLastFlag(chain)
		c.Cancel(chain, f)
		return err
	}
	defer c.Cancel(chain, flag)

	var proc processer
	proc.initEnv(chain, flag, at, ad)
	runt := proc.iRuntime.(*runtime.TRuntime)
	runt.SetMode("check")
	defer runt.Close()
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
