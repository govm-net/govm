package handler

import (
	"fmt"
	"os"
	"time"

	"github.com/govm-net/govm/conf"
	core "github.com/govm-net/govm/core"
	"github.com/govm-net/govm/runtime"
	"github.com/govm-net/govm/wallet"
)

func createFirstBlock() {
	t := time.Date(2020, 5, 22, 0, 0, 0, 0, time.UTC)
	id := core.GetLastBlockIndex(1)
	if id > 0 {
		fmt.Println("exist first block")
		return
	}
	fmt.Println("start to create first block")

	c := conf.GetConf()
	conf.LoadWallet(c.WalletFile, c.Password)

	block := new(core.StBlock)
	block.Index = 1
	// block.Time = uint64(time.Now().Unix()-10) * 1000
	block.Time = uint64(t.Unix() * 1000)
	runtime.Decode(c.WalletAddr, &block.Producer)
	for {
		signData := block.GetSignData()
		sign := wallet.Sign(c.PrivateKey, signData)
		if len(sign) == 0 {
			fmt.Println("error sign,length=0")
			os.Exit(2)
		}
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}
		block.SetSign(sign)
		data := block.Output()
		hp := getHashPower(block.Key[:])
		if hp < 10 {
			block.Nonce++
			continue
		}
		rel := getReliability(block)
		core.WriteBlock(1, data)
		SaveBlockReliability(1, block.Key[:], rel)
		setIDBlocks(1, 1, block.Key, 100)

		fmt.Printf("create first block.key:%x\n", block.Key)
		go processEvent(1)
		break
	}
}
