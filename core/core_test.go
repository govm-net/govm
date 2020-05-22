package zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"testing"
	"time"

	"github.com/govm-net/govm/conf"
	"github.com/govm-net/govm/runtime"
	"github.com/govm-net/govm/wallet"
	"github.com/lengzhao/database/disk"
	"github.com/lengzhao/database/server"
	"github.com/govm-net/govm/database"
)

const dbDir = "db_dir"
const enableDBServer = true

var miner wallet.TWallet
var admin wallet.TWallet

type rtForTest struct {
	runtime.TRuntime
}

func (r *rtForTest) Recover(address, sign, msg []byte) bool {
	return true
}

func TestMain(m *testing.M) {
	runtime.AppPath = "../"
	runtime.RunDir = "."
	runtime.BuildDir = "../"
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.Println("begin")
	os.RemoveAll(dbDir)
	conf.LoadWallet("./wallet.key", "123456")
	c := conf.GetConf()
	runtime.Decode(c.WalletAddr, &team)
	if enableDBServer {
		startDBServer()
		for {
			time.Sleep(100 * time.Millisecond)
			err := database.GetClient().Set(1, []byte("test"), []byte("test"), []byte("test"))
			if err == nil {
				break
			}
		}
	}

	miner.Key = wallet.NewPrivateKey()
	pubK := wallet.GetPublicKey(miner.Key)
	miner.Address = wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

	admin.Key = wallet.NewPrivateKey()
	pubK1 := wallet.GetPublicKey(admin.Key)
	admin.Address = wallet.PublicKeyToAddress(pubK1, wallet.EAddrTypeDefault)

	m.Run()
	log.Println("end")
}

func startDBServer() {
	db := server.NewRPCObj(dbDir)
	server.RegisterAPI(db, func(dir string, id uint64) server.DBApi {
		m, err := disk.Open(dir)
		if err != nil {
			log.Println("fail to open db manager,dir:", dir, err)
			return nil
		}
		return m
	})
	rpc.Register(db)
	rpc.HandleHTTP()
	go func() {
		err := http.ListenAndServe(conf.GetConf().DbServerAddr, nil)
		if err != nil {
			log.Println("fail to start db_server:", err)
			os.Exit(5)
		}
	}()
}

func filterTrans(chain uint64, transList [][]byte) (uint32, []Hash) {
	out := make([]Hash, 0)
	var size uint32
	for _, key := range transList {
		stream, _ := runtime.DbGet(dbTransactionData{}, chain, key)
		if stream == nil {
			log.Printf("fail to get transaction data,key:%x\n", key)
			continue
		}
		size += uint32(len(stream))
		h := Hash{}
		runtime.Decode(key, &h)
		out = append(out, h)
	}

	return size, out
}

func doMine(chain uint64, transList [][]byte) error {
	c := conf.GetConf()
	addr := Address{}
	runtime.Decode(c.WalletAddr, &addr)
	block := NewBlock(chain, addr)

	_, lst := filterTrans(chain, transList)

	WriteTransList(chain, lst)
	// block.SetTransList(lst)
	block.TransListHash = GetHashOfTransList(lst)

	for i := 0; i < 1000; i++ {
		signData := block.GetSignData()
		sign := wallet.Sign(c.PrivateKey, signData)
		block.SetSign(sign)
		data := block.Output()

		hp := getHashPower(block.Key)
		if hp < block.HashpowerLimit {
			block.Nonce++
			// log.Printf("drop hash:%x,Nonce:%d\n", block.Key, block.Nonce)
			continue
		}

		WriteBlock(chain, data)

		return ProcessBlockOfChain(chain, block.Key[:])
		// return block.Key[:]
	}
	return nil
}

func newChain(src, dst uint64) []byte {
	c := conf.GetConf()
	cAddr := Address{}
	runtime.Decode(c.WalletAddr, &cAddr)
	trans := NewTransaction(src, cAddr)
	trans.CreateNewChain(dst, 0)
	trans.Time = GetBlockTime(src)

	signData := trans.GetSignData()
	sign := wallet.Sign(c.PrivateKey, signData)
	trans.SetSign(sign)
	td := trans.Output()
	// WriteTransaction(src, td)
	runtime.AdminDbSet(dbTransactionData{}, src, runtime.GetHash(td), td, maxDbLife)
	return trans.Key[:]
}

func transfer(src, cost uint64, peer []byte) []byte {
	c := conf.GetConf()
	cAddr := Address{}
	pAddr := Address{}
	runtime.Decode(c.WalletAddr, &cAddr)
	runtime.Decode(peer, &pAddr)
	trans := NewTransaction(src, cAddr)
	trans.CreateTransfer(pAddr, cost)
	trans.Time = GetBlockTime(src)
	signData := trans.GetSignData()
	sign := wallet.Sign(c.PrivateKey, signData)
	trans.SetSign(sign)
	td := trans.Output()
	// WriteTransaction(src, td)
	runtime.AdminDbSet(dbTransactionData{}, src, runtime.GetHash(td), td, maxDbLife)
	return trans.Key[:]
}

func move(src, dst, cost uint64, privK []byte) []byte {
	pubK := wallet.GetPublicKey(privK)
	addr := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)
	cAddr := Address{}
	runtime.Decode(addr, &cAddr)
	trans := NewTransaction(src, cAddr)
	trans.CreateMove(dst, cost)
	trans.Time = GetBlockTime(src)

	td := trans.GetSignData()
	sign := wallet.Sign(privK, td)
	trans.SetSign(sign)
	td = trans.Output()
	// WriteTransaction(src, td)
	runtime.AdminDbSet(dbTransactionData{}, src, runtime.GetHash(td), td, maxDbLife)
	return trans.Key[:]
}

func firstBlock(chain uint64) error {
	t := time.Date(2019, 1, 1, 22, 30, 1, 1, time.UTC)

	privKey := wallet.NewPrivateKey()
	pubK := wallet.GetPublicKey(privKey)
	address := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

	cAddr := Address{}
	runtime.Decode(address, &cAddr)

	var key []byte
	// first block
	{
		block := new(StBlock)
		block.HashpowerLimit = 5
		block.Producer = cAddr

		block.Time = uint64(t.Unix() * 1000)
		block.Index = 1
		block.Chain = 0
		for {
			signData := block.GetSignData()
			sign := wallet.Sign(privKey, signData)
			block.SetSign(sign)
			data := block.Output()

			hp := getHashPower(block.Key)
			if hp <= block.HashpowerLimit {
				block.Nonce++
				continue
			}
			key = block.Key[:]
			WriteBlock(chain, data)
			break
		}
	}

	ProcessBlockOfChain(chain, key)

	// pDbStat.SetValue([]byte{StatHashPower}, uint64(defaultHashPower), maxDbLife)
	runtime.AdminDbSet(dbStat{}, 1, []byte{StatHashPower}, runtime.Encode(uint64(1000)), maxDbLife)

	return nil
}

func TestFirstBlock(t *testing.T) {
	log.Println("testing start", t.Name())
	err := firstBlock(1)
	if err != nil {
		t.Error("fail to create first transaction:", err)
	}
	id := GetLastBlockIndex(1)
	if id != 1 {
		t.Error("hope block id = 1,get ", id)
	}
	doMine(1, nil)
	id = GetLastBlockIndex(1)
	if id != 2 {
		t.Error("hope block id = 2,get ", id)
	}
}

func TestTransfer(t *testing.T) {
	log.Println("testing start", t.Name())
	privK := wallet.NewPrivateKey()
	pubK := wallet.GetPublicKey(privK)
	addr := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

	var cost uint64
	cost = 3000000
	trans := transfer(1, cost, addr)
	list := make([][]byte, 0)
	list = append(list, trans)
	doMine(1, list)
	cost2 := GetUserCoin(1, addr)
	if cost2 != cost {
		t.Error("error transfer:", cost, cost2)
	}
}

func TestNewChain(t *testing.T) {
	log.Println("testing start", t.Name())
	id := GetLastBlockIndex(1)
	for i := id; i < 20; i++ {
		doMine(1, nil)
	}

	trans := newChain(1, 2)
	list := make([][]byte, 0)
	list = append(list, trans)
	err := doMine(1, list)
	if err != nil {
		t.Fatal("fail to create new chain:", err)
	}
	key := GetTheBlockKey(1, 1)
	data := ReadBlockData(1, key)
	WriteBlock(2, data)
	err = ProcessBlockOfChain(2, key)
	if err != nil {
		t.Fatal("fail to process first block of chain2:", err)
	}
	runtime.AdminDbSet(dbStat{}, 2, []byte{StatHashPower}, runtime.Encode(uint64(1000)), maxDbLife)

	doMine(1, nil)
	doMine(1, nil)
	doMine(2, nil)
	doMine(2, nil)
}

func TestMove(t *testing.T) {
	log.Println("testing start", t.Name())
	privK := wallet.NewPrivateKey()
	pubK := wallet.GetPublicKey(privK)
	addr := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

	var cost uint64
	cost = 1<<30 + 3000000
	trans := transfer(1, cost, addr)
	list := make([][]byte, 0)
	list = append(list, trans)
	doMine(1, list)
	cost2 := GetUserCoin(1, addr)
	if cost2 != cost {
		t.Error("error transfer:", cost, cost2)
	}
	log.Println("transfer result:", cost, cost2)

	{
		cost -= 3000000
		cAddr := Address{}
		runtime.Decode(addr, &cAddr)
		tran := NewTransaction(1, cAddr)
		tran.CreateMove(2, cost)
		tran.Time = GetBlockTime(1)
		tran.Energy = 3000000

		td := tran.GetSignData()
		sign := wallet.Sign(privK, td)
		tran.SetSign(sign)
		td = tran.Output()
		// WriteTransaction(1, td)
		runtime.AdminDbSet(dbTransactionData{}, 1, runtime.GetHash(td), td, maxDbLife)
		list = make([][]byte, 0)
		list = append(list, runtime.GetHash(td))
		doMine(1, list)
	}

	cost3 := GetUserCoin(1, addr)
	if cost3 != 0 {
		t.Error("fail to move,", cost3)
		log.Println("fail to move:", cost3)
	}
	for i := 0; i < 5; i++ {
		doMine(1, nil)
		doMine(2, nil)
	}
	log.Println("finish to sync")

	cost4 := GetUserCoin(2, addr)
	if cost != cost4 {
		t.Error("error 2:", cost, cost4)
		log.Println("fail to move result:", cost, cost4)
	}
}

func TestNewApp1(t *testing.T) {
	log.Println("testing start", t.Name())
	c := conf.GetConf()
	code, ln := CreateAppFromSourceCode("./test_data/app2/app2.go", AppFlagImport|AppFlagPlublc|AppFlagRun|AppFlagGzipCompress)
	cAddr := Address{}
	runtime.Decode(c.WalletAddr, &cAddr)
	var appName Hash
	//new app
	{
		trans := NewTransaction(1, cAddr)
		trans.CreateNewApp(code, ln)
		trans.Time = GetBlockTime(1)

		td := trans.GetSignData()
		sign := wallet.Sign(c.PrivateKey, td)
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}
		trans.SetSign(sign)
		td = trans.Output()
		// WriteTransaction(1, td)
		runtime.AdminDbSet(dbTransactionData{}, 1, runtime.GetHash(td), td, maxDbLife)
		list := make([][]byte, 0)
		list = append(list, runtime.GetHash(td))

		doMine(1, list)
		hs := runtime.GetHash(code)
		runtime.Decode(hs, &appName)
	}

	// run app
	{
		privK := wallet.NewPrivateKey()
		pubK := wallet.GetPublicKey(privK)
		addr := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

		list := make([][]byte, 0)
		appInfo := GetAppInfoOfChain(1, appName[:])
		key := transfer(1, 1<<20, appInfo.Account[:])
		list = append(list, key)
		key = transfer(1, 1<<20, addr[:])
		list = append(list, key)

		trans := NewTransaction(1, cAddr)
		trans.CreateRunApp(appName, 10000, addr)
		trans.Time = GetBlockTime(1)
		trans.Energy = 1 << 20

		td := trans.GetSignData()
		sign := wallet.Sign(c.PrivateKey, td)
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}
		trans.SetSign(sign)
		td = trans.Output()
		// WriteTransaction(1, td)
		runtime.AdminDbSet(dbTransactionData{}, 1, runtime.GetHash(td), td, maxDbLife)
		list = append(list, runtime.GetHash(td))
		doMine(1, list)
		if GetUserCoin(1, addr[:]) != 1<<20 {
			t.Errorf("error cost:%d\n", GetUserCoin(1, addr[:]))
		}
	}

}

func TestNewApp2(t *testing.T) {
	log.Println("testing start", t.Name())
	/*
		1.doMine，以使两条链的时间接近
		2.分别在两条链上创建app
		3.再次doMine，以达到跨链读取log的时间
		4.分别读取另一条链里的数据
	*/
	var t1, t2 uint64
	c := conf.GetConf()
	code, ln := CreateAppFromSourceCode("./test_data/app3/app3.go", AppFlagPlublc|AppFlagRun)
	cAddr := Address{}
	runtime.Decode(c.WalletAddr, &cAddr)
	var appName Hash
	hs := runtime.GetHash(code)
	runtime.Decode(hs, &appName)
	//new app
	for {
		t1 = GetBlockTime(1)
		t2 = GetBlockTime(2)
		if t1+2*maxBlockInterval >= t2 {
			break
		}
		err := doMine(1, nil)
		if err != nil {
			t.Fatal("fail to do mine2:", err)
		}
	}
	for {
		t1 = GetBlockTime(1)
		t2 = GetBlockTime(2)
		if t2+2*maxBlockInterval >= t1 {
			break
		}
		err := doMine(2, nil)
		if err != nil {
			t.Fatal("fail to do mine2:", err)
		}
	}

	var i uint64
	var cost uint64 = 2 << 50
	for i = 1; i <= 2; i++ {
		runtime.AdminDbSet(dbCoin{}, i, cAddr[:], runtime.Encode(cost), maxDbLife)

		trans := NewTransaction(i, cAddr)
		trans.CreateNewApp(code, ln)
		trans.Time = GetBlockTime(i)

		td := trans.GetSignData()
		sign := wallet.Sign(c.PrivateKey, td)
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}
		trans.SetSign(sign)
		td = trans.Output()
		// WriteTransaction(1, td)
		runtime.AdminDbSet(dbTransactionData{}, i, runtime.GetHash(td), td, maxDbLife)
		list := make([][]byte, 0)
		list = append(list, runtime.GetHash(td))

		doMine(i, list)
	}

	type tApp struct {
		Ops   uint8
		Key   [4]byte
		Value uint64
		Other uint64
	}

	const (
		OpsWriteData = uint8(iota)
		OpsReadData
		OpsWriteLog
		OpsReadLog
	)

	dInfo := tApp{OpsWriteLog, [4]byte{1, 2, 3, 4}, 100, 0}
	// run app: write
	for i = 1; i <= 2; i++ {
		privK := wallet.NewPrivateKey()
		pubK := wallet.GetPublicKey(privK)
		addr := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

		runtime.AdminDbSet(dbCoin{}, i, cAddr[:], runtime.Encode(cost), maxDbLife)

		list := make([][]byte, 0)
		appInfo := GetAppInfoOfChain(i, appName[:])
		runtime.AdminDbSet(dbCoin{}, i, appInfo.Account[:], runtime.Encode(cost), maxDbLife)
		runtime.AdminDbSet(dbCoin{}, i, addr[:], runtime.Encode(cost), maxDbLife)

		trans := NewTransaction(i, cAddr)
		trans.CreateRunApp(appName, 1<<50, runtime.Encode(dInfo))
		trans.Time = GetBlockTime(i)
		trans.Energy = 1 << 40

		td := trans.GetSignData()
		sign := wallet.Sign(c.PrivateKey, td)
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}
		trans.SetSign(sign)
		td = trans.Output()

		runtime.AdminDbSet(dbTransactionData{}, i, runtime.GetHash(td), td, maxDbLife)
		list = append(list, runtime.GetHash(td))
		doMine(i, list)
	}

	// run app: read log,由于时间还没到，应该都是读到空数据
	for i = 1; i <= 2; i++ {
		dInfo := tApp{OpsReadLog, [4]byte{1, 2, 3, 4}, 0, 1}
		if i == 1 {
			dInfo.Other = 2
		}
		privK := wallet.NewPrivateKey()
		pubK := wallet.GetPublicKey(privK)
		addr := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

		runtime.AdminDbSet(dbCoin{}, i, cAddr[:], runtime.Encode(cost), maxDbLife)

		list := make([][]byte, 0)
		appInfo := GetAppInfoOfChain(i, appName[:])
		runtime.AdminDbSet(dbCoin{}, i, appInfo.Account[:], runtime.Encode(cost), maxDbLife)
		runtime.AdminDbSet(dbCoin{}, i, addr[:], runtime.Encode(cost), maxDbLife)

		trans := NewTransaction(i, cAddr)
		trans.CreateRunApp(appName, 1<<50, runtime.Encode(dInfo))
		trans.Time = GetBlockTime(i)
		trans.Energy = 1 << 40

		td := trans.GetSignData()
		sign := wallet.Sign(c.PrivateKey, td)
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}
		trans.SetSign(sign)
		td = trans.Output()

		runtime.AdminDbSet(dbTransactionData{}, i, runtime.GetHash(td), td, maxDbLife)
		list = append(list, runtime.GetHash(td))
		doMine(i, list)
	}

	t1 = GetBlockTime(1)
	t2 = GetBlockTime(2)

	for i := 0; i < 40; i++ {
		if GetBlockTime(1) > GetBlockTime(2) {
			doMine(2, nil)
		} else {
			doMine(1, nil)
		}
	}
	for {
		ti := GetBlockTime(1)
		if ti > t1+blockSyncMax && ti > t2+blockSyncMax {
			break
		}
		doMine(1, nil)
	}
	for {
		ti := GetBlockTime(2)
		if ti > t1+blockSyncMax && ti > t2+blockSyncMax {
			break
		}
		doMine(2, nil)
	}

	// run app: read log
	for i = 1; i <= 2; i++ {
		dInfo := tApp{OpsReadLog, [4]byte{1, 2, 3, 4}, 100, 1}
		if i == 1 {
			dInfo.Other = 2
		}
		privK := wallet.NewPrivateKey()
		pubK := wallet.GetPublicKey(privK)
		addr := wallet.PublicKeyToAddress(pubK, wallet.EAddrTypeDefault)

		runtime.AdminDbSet(dbCoin{}, i, cAddr[:], runtime.Encode(cost), maxDbLife)

		list := make([][]byte, 0)
		appInfo := GetAppInfoOfChain(i, appName[:])
		runtime.AdminDbSet(dbCoin{}, i, appInfo.Account[:], runtime.Encode(cost), maxDbLife)
		runtime.AdminDbSet(dbCoin{}, i, addr[:], runtime.Encode(cost), maxDbLife)

		trans := NewTransaction(i, cAddr)
		trans.CreateRunApp(appName, 1<<50, runtime.Encode(dInfo))
		trans.Time = GetBlockTime(i)
		trans.Energy = 1 << 40

		td := trans.GetSignData()
		sign := wallet.Sign(c.PrivateKey, td)
		if len(c.SignPrefix) > 0 {
			s := make([]byte, len(c.SignPrefix))
			copy(s, c.SignPrefix)
			sign = append(s, sign...)
		}
		trans.SetSign(sign)
		td = trans.Output()

		runtime.AdminDbSet(dbTransactionData{}, i, runtime.GetHash(td), td, maxDbLife)
		list = append(list, runtime.GetHash(td))
		doMine(i, list)
	}
	doMine(1, nil)
	doMine(2, nil)
	doMine(1, nil)
	doMine(2, nil)
}

func TestRegisterMiner(t *testing.T) {
	log.Println("testing start", t.Name())
	mineFunc := func() error {
		var a Address
		runtime.Decode(miner.Address, &a)
		block := NewBlock(1, a)

		for i := 0; i < 1000; i++ {
			signData := block.GetSignData()
			sign := wallet.Sign(miner.Key, signData)
			block.SetSign(sign)
			data := block.Output()

			hp := getHashPower(block.Key)
			if hp < block.HashpowerLimit {
				block.Nonce++
				// log.Printf("drop hash:%x,Nonce:%d\n", block.Key, block.Nonce)
				continue
			}
			WriteBlock(1, data)
			return ProcessBlockOfChain(1, block.Key[:])
		}
		return fmt.Errorf("error1")
	}

	err := mineFunc()
	if err == nil {
		t.Error("not right to mine")
	}

	c := conf.GetConf()
	trans := NewTransaction(1, team)
	trans.CreateRegisterMiner(1, maxGuerdon, miner.Address)
	trans.Time = GetBlockTime(1)

	td := trans.GetSignData()
	sign := wallet.Sign(c.PrivateKey, td)
	if len(c.SignPrefix) > 0 {
		s := make([]byte, len(c.SignPrefix))
		copy(s, c.SignPrefix)
		sign = append(s, sign...)
	}
	trans.SetSign(sign)
	td = trans.Output()
	// WriteTransaction(1, td)
	runtime.AdminDbSet(dbTransactionData{}, 1, runtime.GetHash(td), td, maxDbLife)
	list := make([][]byte, 0)
	list = append(list, runtime.GetHash(td))

	doMine(1, list)

	err = mineFunc()
	if err == nil {
		t.Error("not right to mine")
	}

	id := GetLastBlockIndex(1)
	i := id
	for i < id+activeMiner+1 {
		i = GetLastBlockIndex(1)
		t1 := GetBlockTime(1)
		t2 := GetBlockTime(2)
		if t1 > t2 {
			err := doMine(2, nil)
			if err != nil {
				t.Fatal("fail to do mine2:", err)
			}
		} else {
			err := doMine(1, nil)
			if err != nil {
				t.Fatal("fail to do mine2:", err)
			}
		}
	}

	err = mineFunc()
	if err != nil {
		t.Error("fail to doMine,", err)
	}
}

func TestRegisterAdmin(t *testing.T) {
	log.Println("testing start", t.Name())
	var cost uint64 = 2000 * maxGuerdon
	runtime.AdminDbSet(dbCoin{}, 1, admin.Address, runtime.Encode(cost), maxDbLife)

	var oldAdmins [AdminNum]Address
	getDataFormDB(1, dbStat{}, []byte{StatAdmin}, &oldAdmins)
	for _, it := range oldAdmins {
		if bytes.Compare(admin.Address, it[:]) == 0 {
			t.Error("hope not admin")
		}
	}

	var a Address
	runtime.Decode(admin.Address, &a)
	trans1 := NewTransaction(1, a)
	trans1.CreateRegisterAdmin(1000 * maxGuerdon)
	trans1.Time = GetBlockTime(1)

	td1 := trans1.GetSignData()
	sign1 := wallet.Sign(admin.Key, td1)
	trans1.SetSign(sign1)
	td1 = trans1.Output()
	WriteTransaction(1, td1)

	list := make([][]byte, 0)
	list = append(list, trans1.Key)

	trans2 := NewTransaction(1, team)
	trans2.VoteAdmin(maxGuerdon, admin.Address)
	trans2.Time = GetBlockTime(1)

	c := conf.GetConf()
	td2 := trans2.GetSignData()
	sign2 := wallet.Sign(c.PrivateKey, td2)
	trans2.SetSign(sign2)
	td2 = trans2.Output()
	WriteTransaction(1, td2)
	list = append(list, trans2.Key)

	doMine(1, list)

	var admins [AdminNum]Address
	getDataFormDB(1, dbStat{}, []byte{StatAdmin}, &admins)
	for _, it := range admins {
		if bytes.Compare(admin.Address, it[:]) == 0 {
			return
		}
	}
	t.Error("hope is admin")
}

func TestReward(t *testing.T) {

	t0 := GetBlockTime(1)
	if (t0+100*maxBlockInterval)/TimeDay == t0/TimeDay {
		id := GetLastBlockIndex(1)
		fmt.Println("block id:", id)
		t.Fatal("error,need too long time")
	}

	t1 := GetBlockTime(1)
	t2 := GetBlockTime(2)
	for {
		if t1 > t2 {
			doMine(2, nil)
			t2 = GetBlockTime(2)
		} else {
			doMine(1, nil)
			t1 = GetBlockTime(1)
			if t1/TimeDay != t0/TimeDay {
				break
			}
		}
	}
}

func TestGetHashOfTransList(t *testing.T) {
	var hexList = []string{
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e10",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e11",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e12",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e13",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e14",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e15",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e17",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e18",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e19",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1a",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1b",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1c",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1d",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1e",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		"ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e20",
	}
	var trans = make([]Hash, len(hexList))
	for i, it := range hexList {
		data, _ := hex.DecodeString(it)
		runtime.Decode(data, &trans[i])
	}
	h1 := GetHashOfTransList(nil)
	if !h1.Empty() {
		t.Error("hope empty")
	}
	h2 := GetHashOfTransList(trans[:1])
	if h2 != trans[0] {
		t.Error("only one, return self")
	}
	h3 := GetHashOfTransList(trans[:2])
	h4 := GetHashOfTransList(trans[:3])
	h5 := GetHashOfTransList(trans[:4])
	if h3 == h4 || h4 == h5 {
		t.Error("get same hash")
	}
	h6 := GetHashOfTransList(trans)
	if h6.Empty() {
		t.Error("get empty hash")
	}
}
