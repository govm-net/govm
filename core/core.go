package zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/govm-net/govm/runtime"
)

type dbBlockData struct{}
type dbTransList struct{}
type dbTransactionData struct{}
type dbTransInfo struct{}
type dbCoin struct{}
type dbStat struct{}
type dbApp struct{}
type dbDepend struct{}
type dbMiner struct{}
type dbAdmin struct{}
type dbVoteReward struct{}
type dbVote struct{}
type dbErrorBlock struct{}
type logBlockInfo struct{}
type logSync struct{}
type statMining struct{}
type statTransferIn struct{}
type statTransList struct{}
type statMove struct{}
type statAPPRun struct{}
type statVoteReward struct{}

// Hash The KEY of the block of transaction
type Hash [HashLen]byte

// Address the wallet address
type Address [AddressLen]byte

// DependItem App's dependency information
type DependItem struct {
	Alias   [4]byte
	AppName Hash
}

// iRuntime The interface that the executive needs to register
type iRuntime interface {
	//Get the hash of the data
	GetHash(in []byte) []byte
	//Encoding data into data streams.
	Encode(typ uint8, in interface{}) []byte
	//The data stream is filled into a variable of the specified type.
	Decode(typ uint8, in []byte, out interface{}) int
	//Signature verification
	Recover(address, sign, msg []byte) bool
	//The write interface of the database
	DbSet(owner interface{}, key, value []byte, life uint64)
	//The reading interface of the database
	DbGet(owner interface{}, key []byte) ([]byte, uint64)
	//get life of the db
	DbGetLife(owner interface{}, key []byte) uint64
	//The write interface of the log
	LogWrite(owner interface{}, key, value []byte, life uint64)
	//The reading interface of the log
	LogRead(owner interface{}, chain uint64, key []byte) ([]byte, uint64)
	//get life of the log
	LogReadLife(owner interface{}, key []byte) uint64
	//Get the app name with the private structure of app
	GetAppName(in interface{}) []byte
	//New app
	NewApp(name []byte, code []byte)
	//Run app,The content returned is allowed to read across the chain
	RunApp(name, user, data []byte, energy, cost uint64)
	//Event interface for notification to the outside
	Event(user interface{}, event string, param ...[]byte)
}

// DB Type definition of a database.
type DB struct {
	owner interface{}
	p     *processer
}

// Log Type definition of a log. Log data can be read on other chains. Unable to overwrite the existing data.
type Log struct {
	owner interface{}
	p     *processer
}

// AppInfo App info in database
type AppInfo struct {
	Account Address `json:"account,omitempty"`
	LineSum uint64  `json:"line_sum,omitempty"`
	Life    uint64  `json:"life,omitempty"`
	Flag    uint8   `json:"flag"`
}

// BaseInfo stat info of last block
type BaseInfo struct {
	Key           Hash
	Time          uint64
	Chain         uint64
	ID            uint64
	BaseOpsEnergy uint64
	Producer      Address
	ParentID      uint64
	LeftChildID   uint64
	RightChildID  uint64
}

type processer struct {
	BaseInfo
	iRuntime
	isAdmin            bool
	sInfo              tSyncInfo
	pDbBlockData       *DB
	pDbTransactionData *DB
	pDbTransInfo       *DB
	pDbCoin            *DB
	pDbAdmin           *DB
	pDbStat            *DB
	pDbApp             *DB
	pDbDepend          *DB
	pDbMiner           *DB
	pLogSync           *Log
	pLogBlockInfo      *Log
	TransKey           Hash
}

// StatSwitch define switch of govm status
type StatSwitch struct {
	SWMiningCount bool
	SWTransferOut bool
	SWTransferIn  bool
	SWAPPRun      bool
	SWMove        bool
}

// Switch switch of status
var Switch StatSwitch = StatSwitch{
	true, true, true, true, true,
}

// time
const (
	TimeMillisecond = 1
	TimeSecond      = 1000 * TimeMillisecond
	TimeMinute      = 60 * TimeSecond
	TimeHour        = 60 * TimeMinute
	TimeDay         = 24 * TimeHour
	TimeYear        = 31558150 * TimeSecond
	TimeMonth       = TimeYear / 12
)

const (
	// HashLen the byte length of Hash
	HashLen = 32
	// AddressLen the byte length of Address
	AddressLen = 24
	// AdminNum admin number, DPOS vote
	AdminNum = 23

	maxBlockInterval   = TimeMinute
	minBlockInterval   = 10 * TimeMillisecond
	blockSizeLimit     = 1 << 20
	blockSyncMin       = 4 * TimeMinute
	blockSyncMax       = 5 * TimeMinute
	defauldbLife       = 6 * TimeMonth
	adminLife          = 10 * TimeYear
	acceptTransTime    = 10 * TimeDay
	logLockTime        = 3 * TimeDay
	maxDbLife          = 1 << 50
	maxGuerdon         = 5000000000
	minGuerdon         = 50000
	prefixOfPlublcAddr = 255
	guerdonUpdateCycle = 500000
	depositCycle       = 50000
	voteCost           = 1000000000
	defaultHashPower   = 20000
	hpStep             = 1000
	redemptionTotal    = 3103870425939320
)

//Key of the running state
const (
	StatBaseInfo = uint8(iota)
	StatTransKey
	StatGuerdon
	StatBlockSizeLimit
	StatAvgBlockSize
	StatHashPower
	StatBlockInterval
	StatSyncInfo
	StatFirstBlockKey
	StatChangingConfig
	StatBroadcast
	StatParentKey
	StatUser
	StatAdmin
	StatTotalVotes
	StatLastRewarID
	StatTotalCoins
)

const (
	// OpsTransfer pTransfer
	OpsTransfer = uint8(iota)
	// OpsMove Move out of coin, move from this chain to adjacent chains
	OpsMove
	// OpsNewChain create new chain
	OpsNewChain
	// OpsNewApp create new app
	OpsNewApp
	// OpsRunApp run app
	OpsRunApp
	// OpsUpdateAppLife update app life
	OpsUpdateAppLife
	// OpsRegisterMiner Registered as a miner
	OpsRegisterMiner
	// OpsRegisterAdmin Registered as a admin
	OpsRegisterAdmin
	// OpsVote vote admin
	OpsVote
	// OpsUnvote unvote
	OpsUnvote
	// OpsReportError error block
	OpsReportError
	// OpsConfig config
	OpsConfig
)

var (
	// gPublicAddr The address of a public account for the preservation of additional rewards.
	gPublicAddr = Address{prefixOfPlublcAddr, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
	team        = Address{2, 152, 64, 16, 49, 156, 211, 70, 89, 247, 252, 178, 11, 49, 214, 21, 216, 80, 171, 50, 202, 147, 6, 24}

	firstAdmins = []string{
		"01ccaf415a3a6dc8964bf935a1f40e55654a4243ae99c709",
		"02984010319cd34659f7fcb20b31d615d850ab32ca930618",
	}
)

// Empty Check whether Hash is empty
func (h Hash) Empty() bool {
	return h == (Hash{})
}

// MarshalJSON marshal by base64
func (h Hash) MarshalJSON() ([]byte, error) {
	if h.Empty() {
		return json.Marshal(nil)
	}
	return json.Marshal(h[:])
}

// UnmarshalJSON UnmarshalJSON
func (h *Hash) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var v []byte
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	copy(h[:], v)
	return nil
}

// Empty Check where Address is empty
func (a Address) Empty() bool {
	return a == (Address{})
}

// MarshalJSON marshal by base64
func (a Address) MarshalJSON() ([]byte, error) {
	if a.Empty() {
		return json.Marshal(nil)
	}
	return json.Marshal(a[:])
}

// UnmarshalJSON UnmarshalJSON
func (a *Address) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var v []byte
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	copy(a[:], v)
	return nil
}

// Decode decode hex string
func (a *Address) Decode(hexStr string) {
	d, err := hex.DecodeString(hexStr)
	assert(err == nil)
	runtime.Decode(d, a)
}

func assert(cond bool) {
	if !cond {
		panic("error")
	}
}

func assertMsg(cond bool, msg interface{}) {
	if !cond {
		panic(msg)
	}
}

func (p *processer) initEnv(chain uint64, flag []byte, addrType, address string) {
	bit := 32 << (^uint(0) >> 63)
	assertMsg(bit == 64, "only support 64bit system")
	p.pDbBlockData = p.GetDB(dbBlockData{})
	p.pDbTransactionData = p.GetDB(dbTransactionData{})
	p.pDbTransInfo = p.GetDB(dbTransInfo{})
	p.pDbCoin = p.GetDB(dbCoin{})
	p.pDbAdmin = p.GetDB(dbAdmin{})
	p.pDbStat = p.GetDB(dbStat{})
	p.pDbApp = p.GetDB(dbApp{})
	p.pDbDepend = p.GetDB(dbDepend{})
	p.pDbMiner = p.GetDB(dbMiner{})
	p.pLogBlockInfo = p.GetLog(logBlockInfo{})
	p.pLogSync = p.GetLog(logSync{})
	runt := runtime.NewRuntime(addrType, address)
	runt.SetInfo(chain, flag)
	p.iRuntime = runt
	stream, _ := p.DbGet(p.pDbStat.owner, []byte{StatBaseInfo})
	if len(stream) > 0 {
		p.Decode(0, stream, &p.BaseInfo)
	} else {
		p.BaseInfo = BaseInfo{}
	}
}

// GetHash get data hash
func (p *processer) getHash(data []byte) Hash {
	hashKey := Hash{}
	if len(data) == 0 {
		return hashKey
	}
	hash := p.GetHash(data)
	n := p.Decode(0, hash, &hashKey)
	assert(n == HashLen)
	return hashKey
}

// encoding type
const (
	EncBinary = uint8(iota)
	EncJSON
	EncGob
)

/*-------------------------DB----------------------------------*/

// Set Storage data. the record will be deleted when life=0 or value=nil
func (d *DB) Set(key, value []byte, life uint64) {
	assertMsg(len(key) > 0, "empty key")
	assertMsg(len(value) < 40960, "value size over limit")
	if life == 0 || len(value) == 0 {
		value = nil
		life = 0
	} else {
		life += d.p.Time
	}

	d.p.DbSet(d.owner, key, value, life)
}

// SetValue Storage data
func (d *DB) SetValue(key []byte, value interface{}, life uint64) {
	if life == 0 {
		d.Set(key, nil, 0)
		return
	}
	v := d.p.Encode(EncBinary, value)
	d.Set(key, v, life)
}

// Get Read data from database
func (d *DB) Get(key []byte) ([]byte, uint64) {
	assertMsg(len(key) > 0, "empty key")
	out, life := d.p.DbGet(d.owner, key)
	if life <= d.p.Time {
		return nil, 0
	}
	return out, (life - d.p.Time)
}

// GetValue read data from database
func (d *DB) GetValue(key []byte, value interface{}) bool {
	v, _ := d.Get(key)
	if len(v) == 0 {
		return false
	}
	n := d.p.Decode(EncBinary, v, value)
	assertMsg(n == len(v), "data length not equal")
	return true
}

// GetInt read data from database
func (d *DB) GetInt(key []byte) uint64 {
	var out uint64
	d.GetValue(key, &out)
	return out
}

// GetDB Through the private structure in app, get a DB of app, the parameter must be a structure, not a pointer.
// such as: owner = tAppInfo{}
func (p *processer) GetDB(owner interface{}) *DB {
	out := DB{}
	out.owner = owner
	out.p = p
	return &out
}

// Write Write log,if exist the key,return false.the key and value can't be nil.
func (l *Log) Write(key, value []byte) bool {
	assertMsg(len(key) > 0, "empty key")
	assertMsg(len(value) > 0, "empty value")
	assertMsg(len(value) < 1024, "value size >= 1k")

	life := l.p.LogReadLife(l.owner, key)
	if life+logLockTime >= l.p.Time {
		log.Printf("try to write log again,owner:%t,key:%x,life:%d,now:%d\n",
			l.owner, key, life, l.p.Time)
		return false
	}
	life = TimeYear

	// t := 10 * p.BaseOpsEnergy * uint64(len(key)+len(value)) * life / TimeDay
	// p.ConsumeEnergy(t)
	life += l.p.Time
	l.p.LogWrite(l.owner, key, value, life)
	return true
}

// Read Read log
func (l *Log) Read(chain uint64, key []byte) []byte {
	assertMsg(len(key) > 0, "empty key")
	if chain == 0 {
		chain = l.p.Chain
	}
	dist := getLogicDist(chain, l.p.Chain)
	// p.ConsumeEnergy(p.BaseOpsEnergy * (1 + dist*10))
	minLife := l.p.Time - blockSyncMax*dist
	maxLife := minLife + TimeYear
	out, life := l.p.LogRead(l.owner, chain, key)
	if life < minLife || life > maxLife {
		return nil
	}
	return out
}

// read read log from parent/child/self
func (l *Log) read(chain uint64, key []byte) []byte {
	if chain == 0 {
		chain = l.p.Chain
	}
	assert(chain/2 == l.p.Chain || l.p.Chain/2 == chain || l.p.Chain == chain)
	minLife := l.p.Time
	if chain != l.p.Chain {
		minLife -= blockSyncMin
	}
	maxLife := minLife + TimeYear

	out, life := l.p.LogRead(l.owner, chain, key)
	if life < minLife || life > maxLife {
		return nil
	}
	return out
}

// GetLog Through the private structure in app, get a Log of app, the parameter must be a structure, not a pointer.
func (p *processer) GetLog(owner interface{}) *Log {
	out := Log{}
	out.owner = owner
	out.p = p
	return &out
}

func getLogicDist(c1, c2 uint64) uint64 {
	var dist uint64
	for {
		if c1 == c2 {
			break
		}
		if c1 > c2 {
			c1 /= 2
		} else {
			c2 /= 2
		}
		dist++
	}
	return dist
}

/***************************** app **********************************/

// GetAppName Get the app name based on the private object
func (p *processer) getAppName(in interface{}) Hash {
	// p.ConsumeEnergy(p.BaseOpsEnergy)
	out := Hash{}
	name := p.GetAppName(in)
	n := p.Decode(0, name, &out)
	assert(n == len(name))
	return out
}

// GetAppInfo get app information
func (p *processer) GetAppInfo(name Hash) *AppInfo {
	out := AppInfo{}
	val, _ := p.pDbApp.Get(name[:])
	if len(val) == 0 {
		return nil
	}
	p.Decode(0, val, &out)
	return &out
}

// GetAppAccount  Get the owner Address of the app
func (p *processer) GetAppAccount(in interface{}) Address {
	app := p.getAppName(in)
	assert(!app.Empty())
	info := p.GetAppInfo(app)
	return info.Account
}

/*-------------------------------------Coin------------------------*/
func (p *processer) getAccount(addr Address) (uint64, uint64) {
	v, l := p.pDbCoin.Get(addr[:])
	if len(v) == 0 {
		return 0, 0
	}
	var val uint64
	n := p.Decode(0, v, &val)
	assert(n == len(v))
	return val, l
}

func (p *processer) adminTransfer(payer, payee Address, value uint64) {
	if payer == payee {
		return
	}
	if value == 0 {
		return
	}

	payeeV, payeeL := p.getAccount(payee)
	payeeV += value
	if payeeV < value {
		return
	}
	if !payer.Empty() {
		v := p.pDbCoin.GetInt(payer[:])
		assertMsg(v >= value, "not enough cost")
		v -= value
		if v == 0 {
			p.pDbCoin.SetValue(payer[:], v, 0)
		} else {
			p.pDbCoin.SetValue(payer[:], v, maxDbLife)
		}
	}
	if !payee.Empty() {
		if payeeV == value {
			// p.ConsumeEnergy(p.BaseOpsEnergy * 1000)
			payeeL = maxDbLife
		}
		p.pDbCoin.SetValue(payee[:], payeeV, payeeL)
	}

	p.Event(dbCoin{}, "pTransfer", payer[:], payee[:], p.Encode(0, value), p.Encode(0, payeeV))
}

/*---------------------------process------------------------------*/

type runParam struct {
	Chain uint64
	Key   Hash
}

func getBaseOpsEnergy(chain uint64) uint64 {
	var out uint64 = 1000
	for chain > 0 {
		chain = chain / 2
		out = out * 15 / 16
	}
	return out + 1
}

func run(chain uint64, flag []byte) {
	var proc processer
	proc.initEnv(chain, flag, "", "")
	key := Hash{}
	n := proc.Decode(0, flag, &key)
	assert(n == len(flag))
	assert(chain != 0)
	assert(!key.Empty())
	proc.Chain = chain
	if proc.BaseOpsEnergy == 0 {
		proc.BaseOpsEnergy = getBaseOpsEnergy(chain)
	}

	proc.processBlock(chain, key)
	return
}

// Block Block structure
type Block struct {
	//signLen	  uint8
	//sign	      []byte
	Time          uint64
	Previous      Hash
	Parent        Hash
	LeftChild     Hash
	RightChild    Hash
	TransListHash Hash
	Producer      Address
	Chain         uint64
	Index         uint64
	Nonce         uint64
}

// BlockInfo the block info in database
type BlockInfo struct {
	Parent       Hash
	LeftChild    Hash
	RightChild   Hash
	Index        uint64
	ParentID     uint64
	LeftChildID  uint64
	RightChildID uint64
	Time         uint64
	Producer     Address
}

func (p *processer) processBlock(chain uint64, key Hash) {
	block := Block{}
	data, _ := p.pDbBlockData.Get(key[:])
	signLen := data[0]
	assert(signLen > 30)
	assert(signLen < 250)

	k := p.getHash(data)
	assert(key == k)

	sign := data[1 : signLen+1]
	signData := data[signLen+1:]

	n := p.Decode(0, signData, &block)
	assert(n == len(signData))

	rst := p.Recover(block.Producer[:], sign, signData)
	assert(rst)

	assert(p.Key == block.Previous)
	assert(p.ID+1 == block.Index)
	assert(block.Producer[0] != prefixOfPlublcAddr)

	p.Time = block.Time
	p.Key = key
	p.ID = block.Index
	p.Producer = block.Producer
	p.pDbStat.SetValue([]byte{StatBaseInfo}, p.BaseInfo, maxDbLife)

	//sync info from other chains
	p.syncInfos()

	if p.ID == 1 {
		assert(block.TransListHash.Empty())
		p.processFirstBlock(block)
		return
	}
	assert(chain == block.Chain)
	assert(!block.Previous.Empty())
	info := BlockInfo{}
	preB := p.getBlockLog(0, block.Previous)
	assert(preB != nil)
	if chain > 1 {
		assert(p.getBlockLog(chain/2, preB.Parent) != nil)
	}
	if !preB.LeftChild.Empty() {
		assert(p.getBlockLog(chain*2, preB.LeftChild) != nil)
	}
	if !preB.RightChild.Empty() {
		assert(p.getBlockLog(chain*2+1, preB.RightChild) != nil)
	}

	info.Index = block.Index
	info.Parent = block.Parent
	info.LeftChild = block.LeftChild
	info.RightChild = block.RightChild
	info.Time = block.Time
	info.Producer = block.Producer

	hpLimit := p.pDbStat.GetInt([]byte{StatHashPower})
	decT := block.Time - preB.Time
	if block.Index == 2 && block.Chain > 1 {
		assert(decT == blockSyncMax+blockSyncMin+TimeSecond)
		k, _ := p.pDbStat.Get([]byte{StatParentKey})
		var parent Hash
		p.Decode(0, k, &parent)
		assert(parent == block.Parent)
	} else {
		blockInterval := p.pDbStat.GetInt([]byte{StatBlockInterval})
		assert(decT == blockInterval)
	}

	p.adminReward(preB.Time)

	hp := getHashPower(key)
	assert(hp > 1)

	var adminList [AdminNum]Address
	p.pDbStat.GetValue([]byte{StatAdmin}, &adminList)
	startBlock := p.pDbMiner.GetInt(block.Producer[:])
	if startBlock > 0 && startBlock <= p.ID {
		assert(hp+2 >= hpLimit/1000)
		p.pDbMiner.SetValue(block.Producer[:], startBlock, TimeYear)
	} else {
		for i := 0; i < AdminNum; i++ {
			if block.Producer == adminList[i] {
				p.isAdmin = true
				break
			}
		}
		assertMsg(p.isAdmin, "not miner")

		if chain == 1 && p.ID < 10000 {
			if hpLimit > defaultHashPower+2 {
				hp = hpLimit/hpStep - 2
			} else {
				hp = defaultHashPower / hpStep
			}
		} else {
			hp = defaultHashPower / hpStep
		}
	}
	hp = hp + hpLimit - hpLimit/hpStep
	p.pDbStat.SetValue([]byte{StatHashPower}, hp, maxDbLife)

	if p.Chain == 1 {
		assert(block.Parent.Empty())
	} else {
		assert(!block.Parent.Empty())
		b := p.getBlockLog(p.Chain/2, block.Parent)
		assert(b != nil)
		info.ParentID = b.Index
		p.ParentID = b.Index
		// log.Print("parent:", b.Time, ",self:", block.Time)
		assert(b.Time+blockSyncMax > block.Time)

		var cb *BlockInfo
		if p.Chain%2 == 0 {
			cb = p.getBlockLog(0, b.LeftChild)
		} else {
			cb = p.getBlockLog(0, b.RightChild)
		}
		assert(cb != nil)
	}

	if p.LeftChildID > 0 {
		assert(!block.LeftChild.Empty())
		b := p.getBlockLog(2*p.Chain, block.LeftChild)
		assert(b != nil)
		info.LeftChildID = b.Index
		p.LeftChildID = b.Index
		if b.Index > 1 {
			assert(b.Time+blockSyncMax > block.Time)
			pb := p.getBlockLog(0, b.Parent)
			assert(pb != nil)
		} else {
			firstKey := Hash{}
			p.Event(logBlockInfo{}, "child_time", p.Encode(0, b.Time))
			p.Event(logBlockInfo{}, "child_time", p.Encode(0, block.Time))
			stream, _ := p.pDbStat.Get([]byte{StatFirstBlockKey})
			p.Decode(0, stream, &firstKey)
			assert(firstKey == block.LeftChild)
			assert(b.Time+3*blockSyncMax > block.Time)
		}
	}
	if p.RightChildID > 0 {
		assert(!block.RightChild.Empty())
		b := p.getBlockLog(2*p.Chain+1, block.RightChild)
		assert(b != nil)
		info.RightChildID = b.Index
		p.RightChildID = b.Index
		if b.Index > 1 {
			assert(b.Time+blockSyncMax > block.Time)
			pb := p.getBlockLog(0, b.Parent)
			assert(pb != nil)
		} else {
			firstKey := Hash{}
			p.Event(logBlockInfo{}, "child_time", p.Encode(0, b.Time))
			p.Event(logBlockInfo{}, "child_time", p.Encode(0, block.Time))
			stream, _ := p.pDbStat.Get([]byte{StatFirstBlockKey})
			p.Decode(0, stream, &firstKey)
			assert(firstKey == block.RightChild)
			assert(b.Time+3*blockSyncMax > block.Time)
		}
	}
	assert(info.ParentID >= preB.ParentID)
	assert(info.LeftChildID >= preB.LeftChildID)
	assert(info.RightChildID >= preB.RightChildID)
	assert(p.pLogBlockInfo.Write(key[:], p.Encode(0, info)))
	assert(p.pLogBlockInfo.Write(p.Encode(0, info.Index), key[:]))
	p.pDbStat.SetValue([]byte{StatBaseInfo}, p.BaseInfo, maxDbLife)

	size := p.processTransList(info, block.TransListHash)
	sizeLimit := p.pDbStat.GetInt([]byte{StatBlockSizeLimit})
	assert(size <= sizeLimit)
	avgSize := p.pDbStat.GetInt([]byte{StatAvgBlockSize})
	avgSize = (avgSize*(depositCycle-1) + size) / depositCycle
	p.pDbStat.SetValue([]byte{StatAvgBlockSize}, avgSize, maxDbLife)

	//Mining guerdon
	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
	if p.isAdmin {
		p.adminTransfer(Address{}, gPublicAddr, guerdon)
	} else {
		p.adminTransfer(Address{}, block.Producer, guerdon)
	}
	p.adminTransfer(Address{}, gPublicAddr, guerdon*4/5)
	p.adminTransfer(Address{}, team, guerdon/5)

	old := p.pDbStat.GetInt([]byte{StatTotalCoins})
	old += 2 * guerdon
	p.pDbStat.SetValue([]byte{StatTotalCoins}, old, maxDbLife)

	// Every pre year, the reward is halved
	if block.Index%guerdonUpdateCycle == 0 {
		guerdon = guerdon*9/10 + minGuerdon
		p.pDbStat.SetValue([]byte{StatGuerdon}, guerdon, maxDbLife)
	}

	val := p.pDbCoin.GetInt(gPublicAddr[:]) / guerdonUpdateCycle
	p.adminTransfer(gPublicAddr, block.Producer, val)
	if Switch.SWMiningCount {
		db := p.GetDB(statMining{})
		count := db.GetInt(block.Producer[:])
		db.SetValue(block.Producer[:], count+1, TimeYear)
		db.Set(p.Encode(0, p.ID), block.Producer[:], TimeYear)
	}

	p.Event(logBlockInfo{}, "finish_block", key[:])
}

type tSyncInfo struct {
	ToParentID       uint64
	ToLeftChildID    uint64
	ToRightChildID   uint64
	FromParentID     uint64
	FromLeftChildID  uint64
	FromRightChildID uint64
}

func (p *processer) processFirstBlock(block Block) {
	assert(block.Chain == 0)
	assert(block.Previous.Empty())
	assert(block.Parent.Empty())
	assert(block.LeftChild.Empty())
	assert(block.RightChild.Empty())
	// assert(block.TransListHash.Empty())
	assert(block.Index == 1)
	blockInfo := BlockInfo{}
	blockInfo.Index = block.Index
	blockInfo.Parent = block.Parent
	blockInfo.LeftChild = block.LeftChild
	blockInfo.RightChild = block.RightChild
	blockInfo.Time = p.Time
	blockInfo.ParentID = p.ParentID
	blockInfo.Producer = block.Producer

	p.adminTransfer(Address{}, gPublicAddr, maxGuerdon)

	//save block info
	stream := p.Encode(0, blockInfo)
	empHash := Hash{}
	assert(p.pLogBlockInfo.Write(empHash[:], stream))
	assert(p.pLogBlockInfo.Write(p.Key[:], stream))
	assert(p.pLogBlockInfo.Write(p.Encode(0, block.Index), p.Key[:]))
	assert(block.Producer == team)

	if p.Chain == 1 {
		p.pDbStat.SetValue([]byte{StatGuerdon}, uint64(maxGuerdon), maxDbLife)
		p.pDbStat.SetValue([]byte{StatHashPower}, uint64(defaultHashPower), maxDbLife)
		var admins [AdminNum]Address
		for i, it := range firstAdmins {
			var addr Address
			addr.Decode(it)
			var admin = AdminInfo{1, 0}
			p.pDbAdmin.SetValue(addr[:], admin, maxDbLife)
			if i < AdminNum {
				admins[i] = addr
			}
		}
		p.pDbStat.SetValue([]byte{StatAdmin}, &admins, maxDbLife)

		var redemption map[string]uint64
		data, err := ioutil.ReadFile("./conf/redemption.json")
		assert(err == nil)
		err = json.Unmarshal(data, &redemption)
		assert(err == nil)
		var total uint64
		for k, v := range redemption {
			var addr Address
			addr.Decode(k)
			p.adminTransfer(Address{}, addr, v)
			p.registerMiner(addr)
			total += v
		}
		assertMsg(total == redemptionTotal, "error redemptionTotal")
		total += maxGuerdon
		p.pDbStat.SetValue([]byte{StatTotalCoins}, total, maxDbLife)
	} else {
		old := p.pDbStat.GetInt([]byte{StatTotalCoins})
		old += maxGuerdon
		p.pDbStat.SetValue([]byte{StatTotalCoins}, old, maxDbLife)
	}

	p.pDbStat.SetValue([]byte{StatBlockSizeLimit}, uint64(blockSizeLimit), maxDbLife)
	p.pDbStat.SetValue([]byte{StatBlockInterval}, getBlockInterval(p.Chain), maxDbLife)
	p.pDbStat.Set([]byte{StatFirstBlockKey}, p.Key[:], maxDbLife)

	saveInfo := AppInfo{}
	saveInfo.Flag = AppFlagImport | AppFlagPlublc
	saveInfo.Life = maxDbLife + p.Time
	saveInfo.Account = gPublicAddr
	saveInfo.LineSum = 100
	appName := p.GetAppName(dbStat{})
	p.pDbApp.SetValue(appName[:], saveInfo, maxDbLife)
	p.NewApp(appName[:], nil)

	p.Event(logBlockInfo{}, "finish_block", p.Key[:])
}

func (p *processer) adminReward(preTime uint64) {
	if preTime/TimeDay == p.Time/TimeDay {
		return
	}
	var adminList [AdminNum]Address
	p.pDbStat.GetValue([]byte{StatAdmin}, &adminList)
	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
	totalVotes := p.pDbStat.GetInt([]byte{StatTotalVotes})
	rewDB := p.GetDB(dbVoteReward{})
	if totalVotes != 0 {
		lastID := p.pDbStat.GetInt([]byte{StatLastRewarID})
		p.pDbStat.SetValue([]byte{StatLastRewarID}, p.ID, TimeYear)
		assertMsg(lastID < p.ID, "error lastID")
		var rew RewardInfo
		rew.Reward = (p.ID - lastID) * guerdon * 4 / 5 / totalVotes
		for i := 0; i < AdminNum; i++ {
			if adminList[i].Empty() {
				continue
			}
			var admin AdminInfo
			p.pDbAdmin.GetValue(adminList[i][:], &admin)
			rew.Admins[i] = adminList[i]
			rew.Votes[i] = admin.Votes

			r := rew.Reward * admin.Votes * 3 / 10
			v := p.pDbCoin.GetInt(gPublicAddr[:])
			if v < r {
				r = v
			}
			p.adminTransfer(gPublicAddr, adminList[i], r)
		}
		rewDB.SetValue(p.Encode(0, p.Time/TimeDay), rew, TimeYear)
	}
}

func getBlockInterval(chain uint64) uint64 {
	var out uint64 = maxBlockInterval - minBlockInterval
	for chain > 1 {
		out = out * 15 / 16
		chain = chain / 2
	}
	out += minBlockInterval
	return out
}

func getHashPower(in Hash) uint64 {
	var out uint64
	if in.Empty() {
		return 0
	}
	for i := 0; i < HashLen; i++ {
		out += 8
		item := in[i]
		if item != 0 {
			for item > 0 {
				out--
				item /= 2
			}
			return out
		}
	}
	return out
}

func (p *processer) getBlockLog(chain uint64, key Hash) *BlockInfo {
	if chain == 0 {
		chain = p.Chain
	}
	assert(chain/2 == p.Chain || p.Chain/2 == chain || p.Chain == chain)
	stream := p.pLogBlockInfo.read(chain, key[:])
	if len(stream) == 0 {
		return nil
	}
	out := BlockInfo{}
	p.Decode(0, stream, &out)
	return &out
}

func getHashOfList(in []byte) Hash {
	transList := []Hash{}
	num := len(in) / HashLen
	if num == 0 {
		return Hash{}
	}
	for i := 0; i < num; i++ {
		var h Hash
		n := runtime.Decode(in, &h)
		in = in[n:]
		transList = append(transList, h)
	}
	for len(transList) > 1 {
		tmpList := []Hash{}
		for i := 0; i < len(transList)/2; i++ {
			d := make([]byte, 0, HashLen*2)
			d = append(d, transList[2*i][:]...)
			d = append(d, transList[2*i+1][:]...)
			h := Hash{}
			hash := runtime.GetHash(d)
			n := runtime.Decode(hash, &h)
			assert(n == HashLen)
			tmpList = append(tmpList, h)
		}

		if len(transList)%2 != 0 {
			tmpList = append(tmpList, transList[len(transList)-1])
		}
		transList = tmpList
	}
	return transList[0]
}

// GetHashOfTransList get hash of transaction list
func GetHashOfTransList(transList []Hash) Hash {
	if len(transList) == 0 {
		return Hash{}
	}

	for len(transList) > 1 {
		num := (len(transList) + 1) / 2
		tmpList := make([]Hash, num)
		for i := 0; i < len(transList)/2; i++ {
			d := make([]byte, 0, HashLen*2)
			d = append(d, transList[2*i][:]...)
			d = append(d, transList[2*i+1][:]...)
			h := runtime.GetHash(d)
			runtime.Decode(h, &tmpList[i])
		}
		if len(transList)%2 != 0 {
			tmpList[num-1] = transList[len(transList)-1]
		}
		transList = tmpList
	}

	return transList[0]
}

func (p *processer) processTransList(block BlockInfo, key Hash) uint64 {
	if key.Empty() {
		return 0
	}
	db := p.GetDB(dbTransList{})
	data, _ := db.Get(key[:])
	assert(data != nil)
	size := len(data)
	num := size / HashLen
	assert(num*HashLen == size)

	transList := make([]Hash, num)
	for i := 0; i < num; i++ {
		n := p.Decode(0, data, &transList[i])
		data = data[n:]
	}

	hk := GetHashOfTransList(transList)
	assert(hk == key)

	var out uint64
	for i := 0; i < len(transList); i++ {
		out += p.processTransaction(block, transList[i])
	}

	return out
}

// TransactionHead transaction = sign + head + data
type TransactionHead struct {
	//signLen uint8
	//sing  []byte
	Time   uint64
	User   Address
	Chain  uint64
	Energy uint64
	Cost   uint64
	Ops    uint8
}

// Transaction the transaction data
type Transaction struct {
	TransactionHead
	data []byte
	key  Hash
}

// TransInfo the transaction info
type TransInfo struct {
	User    Address
	Ops     uint8
	BlockID uint64
	Cost    uint64
}

func (p *processer) processTransaction(block BlockInfo, key Hash) uint64 {
	p.Event(dbTransInfo{}, "start_transaction", key[:])
	ti, _ := p.pDbTransInfo.Get(key[:])
	assert(len(ti) == 0)
	data, _ := p.pDbTransactionData.Get(key[:])
	signLen := data[0]
	assert(signLen > 30)
	k := p.getHash(data)
	assert(k == key)
	p.TransKey = key

	sign := data[1 : signLen+1]
	signData := data[signLen+1:]

	trans := Transaction{}
	n := p.Decode(0, signData, &trans.TransactionHead)
	trans.data = signData[n:]
	dataLen := len(trans.data)
	trans.key = key

	assert(p.Recover(trans.User[:], sign, signData))

	// transaction list
	db := p.GetDB(statTransList{})
	id := db.GetInt([]byte{0}) + 1
	db.SetValue([]byte{0}, id, maxDbLife)
	db.Set(p.Encode(0, id), key[:], TimeYear)
	// user
	tid := db.GetInt(trans.User[:]) + 1
	newKey := append(trans.User[:], p.Encode(0, tid)...)
	db.SetValue(trans.User[:], tid, maxDbLife)
	db.Set(newKey, key[:], TimeYear)

	assertMsg(trans.Time <= block.Time+TimeHour, "trans_newer")
	assert(trans.Time+acceptTransTime > block.Time)
	assert(trans.User[0] != prefixOfPlublcAddr)
	if block.Index == 1 {
		assert(trans.Chain == 0)
		trans.Chain = p.Chain
	}
	assert(trans.Chain == p.Chain)
	assert(trans.Energy > uint64(dataLen))
	p.pDbStat.Set([]byte{StatTransKey}, key[:], defauldbLife)
	p.pDbStat.Set([]byte{StatUser}, trans.User[:], defauldbLife)

	info := TransInfo{}
	info.BlockID = block.Index
	info.User = trans.User
	info.Ops = trans.Ops
	info.Cost = trans.Cost
	p.pDbTransInfo.SetValue(key[:], info, defauldbLife)

	trans.Energy /= 2
	p.adminTransfer(trans.User, gPublicAddr, trans.Energy)
	p.adminTransfer(trans.User, block.Producer, trans.Energy)
	switch trans.Ops {
	case OpsTransfer:
		assert(dataLen < 300)
		p.pTransfer(trans)
	case OpsMove:
		assert(dataLen < 300)
		p.pMove(trans)
	case OpsNewChain:
		assert(dataLen < 300)
		p.pNewChain(block.Producer, trans)
	case OpsNewApp:
		assert(dataLen < 81920)
		p.pNewApp(trans)
	case OpsRunApp:
		assert(dataLen < 2048)
		p.pRunApp(trans)
	case OpsUpdateAppLife:
		assert(dataLen < 300)
		p.pUpdateAppLife(trans)
	case OpsRegisterMiner:
		assert(dataLen < 300)
		p.pRegisterMiner(trans)
	case OpsRegisterAdmin:
		assert(dataLen < 300)
		p.pRegisterAdmin(trans.User, trans.Cost)
	case OpsVote:
		assert(dataLen < 300)
		p.pVote(trans)
	case OpsUnvote:
		assert(dataLen < 300)
		p.pUnvote(trans.User, trans.data)
	case OpsReportError:
		p.pReportError(trans.User, trans.data)
	case OpsConfig:
		p.pConfig(trans.User, trans.Cost, trans.data)
	default:
		assert(false)
	}

	p.Event(dbTransInfo{}, "finish_transaction", key[:])
	return uint64(len(data))
}

// t.data = Address + msg
func (p *processer) pTransfer(t Transaction) {
	var payee Address
	p.Decode(0, t.data, &payee)
	assert(t.Cost > 0)
	assert(!payee.Empty())
	p.adminTransfer(t.User, payee, t.Cost)

	if Switch.SWTransferIn {
		db := p.GetDB(statTransferIn{})
		id := db.GetInt(payee[:]) + 1
		db.SetValue(payee[:], id, TimeYear)
		rk := append(payee[:], runtime.Encode(id)...)
		db.Set(rk, t.key[:], TimeMonth)
	}
}

// syncMoveInfo sync Information of move out
type syncMoveInfo struct {
	Key   Hash
	User  Address
	Value uint64
}

//t.data = chain
func (p *processer) pMove(t Transaction) {
	var chain uint64
	p.Decode(0, t.data, &chain)
	assert(chain > 0)
	assert(t.Energy >= 100*p.BaseOpsEnergy)
	if p.Chain > chain {
		assert(p.Chain/2 == chain)
	} else {
		assert(p.Chain == chain/2)
		if chain%2 == 0 {
			assertMsg(p.LeftChildID > 0, "chain not exist")
		} else {
			assertMsg(p.RightChildID > 0, "chain not exist")
		}
	}

	p.adminTransfer(t.User, Address{}, t.Cost)
	stru := syncMoveInfo{t.key, t.User, t.Cost}
	p.addSyncInfo(chain, SyncOpsMoveCoin, p.Encode(0, stru))

	if Switch.SWMove {
		db := p.GetDB(statMove{})
		id := db.GetInt(t.User[:]) + 1
		db.SetValue(t.User[:], id, TimeYear)
		rk := append(t.User[:], runtime.Encode(id)...)
		db.Set(rk, t.key[:], TimeMonth)
	}
}

/*********************** chain ****************************/

type syncNewChain struct {
	Chain      uint64
	BlockKey   Hash
	Guerdon    uint64
	ParentID   uint64
	Time       uint64
	HashPower  uint64
	Producer   Address
	FirstBlock Hash
	AdminList  [AdminNum]Address
}

//t.data = newChain
func (p *processer) pNewChain(producer Address, t Transaction) {
	var newChain uint64
	p.Decode(0, t.data, &newChain)
	assert(newChain/2 == t.Chain)

	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
	si := syncNewChain{}
	si.Chain = newChain
	si.Guerdon = guerdon*9/10 + minGuerdon
	si.BlockKey = p.Key
	si.ParentID = p.ID
	si.Producer = producer
	si.Time = p.Time - blockSyncMax + 1
	si.HashPower = p.pDbStat.GetInt([]byte{StatHashPower})
	if si.HashPower < defaultHashPower {
		si.HashPower = defaultHashPower
	}
	p.pDbStat.GetValue([]byte{StatAdmin}, &si.AdminList)
	var find bool
	for _, it := range si.AdminList {
		if it == t.User {
			find = true
			break
		}
	}
	assertMsg(find, "request admin")
	data, life := p.pDbStat.Get([]byte{StatFirstBlockKey})
	assert(life+blockSyncMax < maxDbLife)
	p.Decode(0, data, &si.FirstBlock)
	p.addSyncInfo(newChain, SyncOpsNewChain, p.Encode(0, si))

	//left child chain
	if newChain == t.Chain*2 {
		assert(p.LeftChildID == 0)
		p.LeftChildID = 1
	} else { //right child chain
		assert(p.RightChildID == 0)
		p.RightChildID = 1
	}

	if newChain > 2 {
		avgSize := p.pDbStat.GetInt([]byte{StatAvgBlockSize})
		scale := avgSize * 10 / blockSizeLimit
		assert(scale > 1)
		cost := 10000*guerdon + 10*maxGuerdon
		cost = cost >> scale
		assert(t.Cost >= cost)
		p.pDbStat.SetValue([]byte{StatAvgBlockSize}, uint64(0), 0)
	}
	p.adminTransfer(t.User, gPublicAddr, t.Cost)

	p.pDbStat.SetValue([]byte{StatBaseInfo}, p.BaseInfo, maxDbLife)
	p.Event(dbTransInfo{}, "new_chain", p.Encode(0, newChain))
}

/*------------------------------app--------------------------------------*/

const (
	// AppFlagRun the app can be call
	AppFlagRun = uint8(1 << iota)
	// AppFlagImport the app code can be included
	AppFlagImport
	// AppFlagPlublc App funds address uses the plublc address, except for app, others have no right to operate the address.
	AppFlagPlublc
	// AppFlagGzipCompress gzip compress
	AppFlagGzipCompress
)

func (p *processer) getPlublcAddress(appName Hash) Address {
	out := Address{}
	p.Decode(0, appName[:], &out)
	out[0] = prefixOfPlublcAddr
	return out
}

type newAppInfo struct {
	LineNum   uint32
	Type      uint16
	Flag      uint8
	DependNum uint8
}

func (p *processer) pNewApp(t Transaction) {
	var (
		appName Hash
		count   uint64 = 1
		deps    []byte
		life    = TimeYear + p.Time
		eng     uint64
	)
	assert(t.Cost == 0)
	code := t.data
	ni := newAppInfo{}
	n := p.Decode(0, code, &ni)
	code = code[n:]
	for i := 0; i < int(ni.DependNum); i++ {
		item := DependItem{}
		n := p.Decode(0, code, &item)
		code = code[n:]

		itemInfo := p.GetAppInfo(item.AppName)
		assertMsg(itemInfo != nil, "not found importation app")
		if life > itemInfo.Life {
			life = itemInfo.Life
		}
		if ni.Flag&AppFlagPlublc != 0 {
			assertMsg(itemInfo.Flag&AppFlagPlublc != 0, "import private app")
		}
		count += itemInfo.LineSum
		assert(count > itemInfo.LineSum)
		assertMsg(itemInfo.Flag&AppFlagImport != 0, "the importation unable import")
		deps = append(deps, item.AppName[:]...)
	}
	assertMsg(life > TimeMonth+p.Time, "the importation not enough life")

	appName = p.getHash(t.data)
	if ni.Flag&AppFlagPlublc == 0 {
		appName = p.getHash(append(appName[:], t.User[:]...))
	}
	assertMsg(p.GetAppInfo(appName) == nil, "the app is exist")

	if ni.Flag&AppFlagImport != 0 {
		eng += uint64(len(code)) * 20
	}
	if ni.Flag&AppFlagRun != 0 {
		eng += uint64(len(code))
	}

	saveInfo := AppInfo{}
	saveInfo.Flag = ni.Flag

	if ni.Flag&AppFlagPlublc != 0 {
		saveInfo.Account = p.getPlublcAddress(appName)
		assert(gPublicAddr != saveInfo.Account)
		if p.ID == 1 {
			saveInfo.Account = gPublicAddr
		}
	} else {
		saveInfo.Account = t.User
	}
	p.NewApp(appName[:], t.data)

	count += uint64(ni.LineNum)
	saveInfo.LineSum = count
	saveInfo.Life = life
	life -= p.Time
	eng += count * p.BaseOpsEnergy
	assertMsg(t.Energy >= eng, "not enough energy")

	stream, _ := p.pDbApp.Get(appName[:])
	assert(len(stream) == 0)
	p.pDbApp.SetValue(appName[:], saveInfo, life)
	if len(deps) > 0 {
		p.pDbDepend.Set(appName[:], deps, life)
	}
}

func (p *processer) pRunApp(t Transaction) {
	var name Hash
	assertMsg(!p.isAdmin, "runApp:the Producer is admin")
	n := p.Decode(0, t.data, &name)
	info := p.GetAppInfo(name)
	assertMsg(info != nil, "app not exist")
	assertMsg(info.Flag&AppFlagRun != 0, "app unable run")
	assertMsg(info.Life >= p.Time, "app expire")
	p.adminTransfer(t.User, info.Account, t.Cost)
	p.RunApp(name[:], t.User[:], t.data[n:], t.Energy, t.Cost)

	if Switch.SWAPPRun {
		db := p.GetDB(statAPPRun{})
		//user
		id := db.GetInt(t.User[:]) + 1
		db.SetValue(t.User[:], id, TimeYear)
		rk := append(t.User[:], runtime.Encode(id)...)
		db.Set(rk, t.key[:], TimeMonth)
		//app
		id = db.GetInt(name[:]) + 1
		db.SetValue(name[:], id, TimeYear)
		rk = append(name[:], runtime.Encode(id)...)
		db.Set(rk, t.key[:], TimeMonth)
	}
}

// UpdateInfo Information of update app life
type UpdateInfo struct {
	Name Hash
	Life uint64
}

func (p *processer) pUpdateAppLife(t Transaction) {
	var info UpdateInfo
	n := p.Decode(0, t.data, &info)
	assert(n == len(t.data))
	f := p.BaseOpsEnergy * (info.Life + TimeDay) / TimeHour
	assert(f <= t.Cost)
	p.UpdateAppLife(info.Name, info.Life)
	p.adminTransfer(t.User, gPublicAddr, t.Cost)
}

// UpdateAppLife update app life
func (p *processer) UpdateAppLife(AppName Hash, life uint64) {
	app := p.GetAppInfo(AppName)
	assert(app != nil)
	assert(app.Life >= p.Time)
	assert(life < 10*TimeYear)
	assert(life > 0)
	app.Life += life
	assert(app.Life > life)
	assert(app.Life < p.Time+10*TimeYear)
	deps, _ := p.pDbDepend.Get(AppName[:])
	p.pDbApp.SetValue(AppName[:], app, app.Life-p.Time)
	if len(deps) == 0 {
		return
	}
	p.pDbDepend.Set(AppName[:], deps, app.Life-p.Time)
	for len(deps) > 0 {
		item := Hash{}
		n := p.Decode(0, deps, &item)
		deps = deps[n:]
		itemInfo := p.GetAppInfo(item)
		assert(itemInfo != nil)
		assert(itemInfo.Life >= app.Life)
	}
}

type syncRegMiner struct {
	Cost uint64
	User Address
}

func (p *processer) registerOtherChainMiner(user Address, chain, cost uint64) {
	assert(chain/2 == p.Chain || p.Chain/2 == chain)
	assert(chain > 0)
	if chain > p.Chain {
		if chain%2 == 0 {
			assert(p.LeftChildID > 0)
		} else {
			assert(p.RightChildID > 0)
		}
	}
	info := syncRegMiner{cost, user}
	p.addSyncInfo(chain, SyncOpsMiner, p.Encode(0, info))
}

func (p *processer) registerMiner(user Address) {
	var startBlock uint64
	startBlock = p.pDbMiner.GetInt(user[:])
	if startBlock != 0 {
		return
	}
	_, life := p.pDbAdmin.Get(user[:])
	if life > 0 {
		return
	}
	p.pDbMiner.SetValue(user[:], p.ID, TimeYear)
}

// dstChain, miner
func (p *processer) pRegisterMiner(t Transaction) {
	miner := t.User
	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
	assertMsg(t.Cost >= guerdon, "not enough cost")
	assertMsg(t.Cost < 100*guerdon, "too much")
	p.adminTransfer(t.User, gPublicAddr, t.Cost)
	var dstChain uint64
	if len(t.data) > 0 {
		n := p.Decode(0, t.data, &dstChain)
		if len(t.data) > AddressLen {
			p.Decode(0, t.data[n:], &miner)
		}
	}

	if dstChain != 0 && dstChain != p.Chain {
		assert(t.Energy > 100000)
		if dstChain > p.Chain {
			assert(dstChain/2 == p.Chain)
		} else {
			assert(dstChain == p.Chain/2)
		}
		p.registerOtherChainMiner(miner, dstChain, t.Cost)
		return
	}

	p.registerMiner(miner)
}

// AdminInfo vote info
type AdminInfo struct {
	Deposit uint64
	Votes   uint64
}

func (p *processer) pRegisterAdmin(user Address, cost uint64) {
	var admin AdminInfo
	p.pDbAdmin.GetValue(user[:], &admin)

	if cost == 0 {
		assertMsg(admin.Deposit > 0, "not admin")
		if admin.Votes > 0 {
			return
		}
		p.pDbAdmin.Set(user[:], nil, 0)
		p.adminTransfer(Address{}, user, admin.Deposit)
		return
	}
	p.adminTransfer(user, Address{}, cost)
	admin.Deposit += cost
	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
	if p.ID > 1 && admin.Deposit < 1000*guerdon {
		return
	}
	p.pDbAdmin.SetValue(user[:], admin, maxDbLife)
	p.pDbMiner.SetValue(user[:], nil, 0)
}

// RewardInfo reward
type RewardInfo struct {
	Admins [AdminNum]Address
	Votes  [AdminNum]uint64
	Reward uint64
}

// VoteInfo vote info
type VoteInfo struct {
	Admin    Address
	Cost     uint64
	StartDay uint64
}

// pVote vote admin
func (p *processer) pVote(trans Transaction) {
	cost := trans.Cost
	user := trans.User
	data := trans.data
	assert(trans.Cost%voteCost == 0)
	voteDB := p.GetDB(dbVote{})
	var vote VoteInfo
	voteDB.GetValue(user[:], &vote)
	p.pVoteRewardValue(user, vote)
	have := p.pDbCoin.GetInt(user[:])
	if cost == 0 || have < cost {
		if have < cost {
			log.Printf("warning vote, chain:%d,have:%d,cost:%d,user:%x\n",
				p.Chain, have, cost, user)
		}
		return
	}

	votes := cost / voteCost
	var addr Address
	p.Decode(0, data, &addr)
	var admin AdminInfo
	p.pDbAdmin.GetValue(addr[:], &admin)
	assertMsg(admin.Deposit > 0, "not admin")

	if vote.Cost > 0 {
		assertMsg(vote.Admin == addr, "different admin")
	}

	totalVotes := p.pDbStat.GetInt([]byte{StatTotalVotes}) + votes
	p.pDbStat.SetValue([]byte{StatTotalVotes}, totalVotes, maxDbLife)
	admin.Votes += votes
	vote.Admin = addr
	vote.Cost += cost
	assert(vote.Cost >= cost)
	// tomorrow
	vote.StartDay = (p.Time + TimeDay) / TimeDay

	p.pDbAdmin.SetValue(addr[:], admin, maxDbLife)
	voteDB.SetValue(user[:], vote, maxDbLife)
	p.adminTransfer(user, Address{}, cost)

	p.Event(dbVote{}, "vote", addr[:], user[:])
	if admin.Votes < totalVotes/1000 {
		return
	}

	var adminList [AdminNum]Address
	p.pDbStat.GetValue([]byte{StatAdmin}, &adminList)
	for i := 0; i < AdminNum; i++ {
		if addr == adminList[i] {
			return
		}
	}
	var replace int = AdminNum + 1
	votes = admin.Votes
	for i := 0; i < AdminNum; i++ {
		if adminList[i].Empty() {
			replace = i
			break
		}
		var next AdminInfo
		p.pDbAdmin.GetValue(adminList[i][:], &next)
		if votes <= next.Votes {
			continue
		}
		replace = i
		votes = next.Votes
	}
	if replace >= AdminNum {
		return
	}
	adminList[replace] = vote.Admin
	p.pDbStat.SetValue([]byte{StatAdmin}, adminList, maxDbLife)
}

func (p *processer) pUnvote(user Address, data []byte) {
	var admin AdminInfo
	var vote VoteInfo
	db := p.GetDB(dbVote{})
	// admin cancel
	if len(data) > 0 {
		var addr Address
		p.Decode(0, data, &addr)
		db.GetValue(addr[:], &vote)
		p.pDbAdmin.GetValue(user[:], &admin)
		if admin.Votes == 0 || vote.Admin != user {
			return
		}
		user = addr
	}
	if vote.Cost == 0 {
		db.GetValue(user[:], &vote)
		p.pDbAdmin.GetValue(vote.Admin[:], &admin)
	}
	if vote.Cost == 0 {
		db.Set(user[:], nil, 0)
		return
	}
	p.pVoteRewardValue(user, vote)

	v := vote.Cost / voteCost
	p.adminTransfer(Address{}, user, vote.Cost)
	assertMsg(admin.Votes >= v, "bug:votes < user votes")
	admin.Votes -= v

	totalVotes := p.pDbStat.GetInt([]byte{StatTotalVotes})
	assertMsg(totalVotes >= v, "totalVotes < user.votes")

	totalVotes -= v
	p.pDbStat.SetValue([]byte{StatTotalVotes}, totalVotes, maxDbLife)

	p.pDbAdmin.SetValue(vote.Admin[:], admin, maxDbLife)
	db.Set(user[:], nil, 0)

	p.Event(dbVote{}, "unvote", vote.Admin[:], user[:])
}

type rewardResult struct {
	User    Address
	BlockID uint64
	Reward  uint64
}

func (p *processer) pVoteRewardValue(user Address, voter VoteInfo) {
	votes := voter.Cost / voteCost
	if votes == 0 {
		return
	}

	end := (p.Time + TimeDay) / TimeDay
	start := (p.Time - 30*TimeDay) / TimeDay
	if start < voter.StartDay {
		start = voter.StartDay
	}
	if start >= end {
		return
	}

	var out uint64
	db := p.GetDB(dbVoteReward{})
	for i := start; i < end; i++ {
		var rew RewardInfo
		var ratio uint64 = 5
		db.GetValue(p.Encode(0, i), &rew)
		for _, it := range rew.Admins {
			if it == voter.Admin {
				ratio = 7
				break
			}
		}

		out += votes * rew.Reward * ratio / 10
	}
	v := p.pDbCoin.GetInt(gPublicAddr[:])
	if v < out {
		out = v
	}
	p.adminTransfer(gPublicAddr, user, out)
	vdb := p.GetDB(dbVote{})
	voter.StartDay = (p.Time + TimeDay) / TimeDay
	vdb.SetValue(user[:], voter, maxDbLife)

	result := rewardResult{user, p.ID, out}
	rdb := p.GetDB(statVoteReward{})
	id := rdb.GetInt([]byte{0}) + 1
	rdb.SetValue([]byte{0}, id, TimeYear)
	rdb.SetValue(p.Encode(0, id), result, TimeYear)
	rdb.SetValue(p.TransKey[:], out, TimeYear)
}

func (p *processer) pReportError(user Address, data []byte) {
	assertMsg(data[0] == 0, "hope error type is 0")
	p.pErrorBlock(user, data[1:])
}

// pErrorBlock admin report error block
func (p *processer) pErrorBlock(user Address, data []byte) {
	block := Block{}
	k := p.getHash(data)
	db := p.GetDB(dbErrorBlock{})
	count := db.GetInt(k[:])
	assert(count == 0)

	signLen := data[0]
	sign := data[1 : signLen+1]
	signData := data[signLen+1:]
	p.Decode(0, signData, &block)
	assert(p.Chain == block.Chain)
	assert(p.Time > block.Time)
	assert(block.Time+TimeDay > p.Time)

	rst := p.Recover(block.Producer[:], sign, signData)
	assert(rst)

	var adminList [AdminNum]Address
	p.pDbStat.GetValue([]byte{StatAdmin}, &adminList)
	var find bool
	for i := 0; i < AdminNum; i++ {
		if adminList[i] == user {
			find = true
			break
		}
	}
	assertMsg(find, "not admin")

	var bErr bool
	if !block.Parent.Empty() {
		if nil == p.getBlockLog(p.Chain/2, block.Parent) {
			bErr = true
		}
	}
	if !bErr && !block.LeftChild.Empty() {
		if nil == p.getBlockLog(p.Chain*2, block.LeftChild) {
			bErr = true
		}
	}
	if !bErr && !block.RightChild.Empty() {
		if nil == p.getBlockLog(p.Chain*2+1, block.RightChild) {
			bErr = true
		}
	}
	if !bErr {
		return
	}

	db.SetValue(k[:], count+1, TimeYear)
	p.pDbMiner.SetValue(block.Producer[:], p.ID+100, TimeYear)
}

// admin change the config
func (p *processer) pConfig(user Address, cost uint64, data []byte) {
	assertMsg(cost >= 100*maxGuerdon, "not enough cost")
	var ops uint8
	var newSize uint64
	ops = data[0]
	p.Decode(0, data[1:], &newSize)
	var adminList [AdminNum]Address
	p.pDbStat.GetValue([]byte{StatAdmin}, &adminList)
	var isAdmin bool
	for _, it := range adminList {
		if it == user {
			isAdmin = true
			break
		}
	}
	assertMsg(isAdmin, "request admin")
	assert(p.pDbStat.GetInt([]byte{StatChangingConfig}) == 0)
	p.pDbStat.SetValue([]byte{StatChangingConfig}, uint64(1), TimeMillisecond)
	var min, max uint64
	switch ops {
	case StatBlockSizeLimit:
		min = blockSizeLimit
		max = 1<<32 - 1
	case StatBlockInterval:
		min = minBlockInterval
		max = maxBlockInterval
	default:
		assert(false)
	}
	v := newSize
	s := p.pDbStat.GetInt([]byte{ops})
	if s == 0 {
		s = max + min/2
	}
	rangeVal := s/100 + 1
	assert(v <= s+rangeVal)
	assert(v >= s-rangeVal)
	assert(v >= min)
	assert(v <= max)
	p.pDbStat.SetValue([]byte{ops}, v, maxDbLife)
	p.adminTransfer(user, gPublicAddr, cost)
	p.Event(dbStat{}, "config", []byte{ops}, p.Encode(0, v))
}

// ops of sync
const (
	SyncOpsMoveCoin = iota
	SyncOpsNewChain
	SyncOpsMiner
	SyncOpsBroadcast
	SyncOpsBroadcastAck
)

type syncHead struct {
	BlockID uint64
	Ops     uint8
}

func (p *processer) getSyncKey(typ byte, index uint64) []byte {
	var key = []byte{typ}
	key = append(key, p.Encode(0, index)...)
	return key
}

func (p *processer) syncInfos() {
	stream, _ := p.pDbStat.Get([]byte{StatSyncInfo})
	if len(stream) > 0 {
		p.Decode(0, stream, &p.sInfo)
	}

	p.syncFromParent()
	p.syncFromLeftChild()
	p.syncFromRightChild()
	p.pDbStat.SetValue([]byte{StatSyncInfo}, p.sInfo, maxDbLife)
}

func (p *processer) addSyncInfo(chain uint64, ops uint8, data []byte) {
	stream, _ := p.pDbStat.Get([]byte{StatSyncInfo})
	if len(stream) > 0 {
		p.Decode(0, stream, &p.sInfo)
	}
	var key []byte
	switch chain {
	case p.Chain / 2:
		key = p.getSyncKey('p', p.sInfo.ToParentID)
		p.sInfo.ToParentID++
	case 2 * p.Chain:
		key = p.getSyncKey('l', p.sInfo.ToLeftChildID)
		p.sInfo.ToLeftChildID++
	case 2*p.Chain + 1:
		key = p.getSyncKey('r', p.sInfo.ToRightChildID)
		p.sInfo.ToRightChildID++
	default:
		assert(false)
	}
	head := syncHead{p.ID, ops}
	d := p.Encode(0, head)
	d = append(d, data...)
	p.pLogSync.Write(key, d)
	p.pDbStat.SetValue([]byte{StatSyncInfo}, p.sInfo, maxDbLife)
	p.Event(logSync{}, "addSyncInfo", []byte{ops}, data)
}

func (p *processer) syncFromParent() {
	if p.Chain == 1 {
		return
	}
	if p.ID == 1 {
		var head syncHead
		var stream []byte
		if p.Chain%2 == 0 {
			stream, _ = p.LogRead(p.pLogSync.owner, p.Chain/2, p.getSyncKey('l', 0))
		} else {
			stream, _ = p.LogRead(p.pLogSync.owner, p.Chain/2, p.getSyncKey('r', 0))
		}
		assert(len(stream) > 0)
		n := p.Decode(0, stream, &head)
		assert(head.Ops == SyncOpsNewChain)
		p.sInfo.FromParentID = 1
		p.syncInfo(p.Chain/2, head.Ops, stream[n:])
	}
	for {
		var head syncHead
		var stream []byte
		if p.Chain%2 == 0 {
			stream = p.pLogSync.read(p.Chain/2, p.getSyncKey('l', p.sInfo.FromParentID))
		} else {
			stream = p.pLogSync.read(p.Chain/2, p.getSyncKey('r', p.sInfo.FromParentID))
		}
		if len(stream) == 0 {
			break
		}
		n := p.Decode(0, stream, &head)
		if head.BlockID > p.ParentID {
			break
		}
		p.sInfo.FromParentID++
		p.pDbStat.SetValue([]byte{StatSyncInfo}, p.sInfo, maxDbLife)
		p.syncInfo(p.Chain/2, head.Ops, stream[n:])
	}
}

func (p *processer) syncFromLeftChild() {
	if p.LeftChildID <= 1 {
		return
	}
	for {
		var head syncHead
		var stream []byte
		stream = p.pLogSync.read(p.Chain*2, p.getSyncKey('p', p.sInfo.FromLeftChildID))
		if len(stream) == 0 {
			break
		}
		n := p.Decode(0, stream, &head)
		if head.BlockID > p.LeftChildID {
			break
		}
		p.sInfo.FromLeftChildID++
		p.pDbStat.SetValue([]byte{StatSyncInfo}, p.sInfo, maxDbLife)
		p.syncInfo(p.Chain*2, head.Ops, stream[n:])
	}
}

func (p *processer) syncFromRightChild() {
	if p.RightChildID <= 1 {
		return
	}
	for {
		var head syncHead
		var stream []byte
		stream = p.pLogSync.read(p.Chain*2+1, p.getSyncKey('p', p.sInfo.FromRightChildID))
		if len(stream) == 0 {
			break
		}
		n := p.Decode(0, stream, &head)
		if head.BlockID > p.RightChildID {
			break
		}
		p.sInfo.FromRightChildID++
		p.pDbStat.SetValue([]byte{StatSyncInfo}, p.sInfo, maxDbLife)
		p.syncInfo(p.Chain*2+1, head.Ops, stream[n:])
	}
}

// BroadcastInfo broadcase info
type BroadcastInfo struct {
	BlockKey Hash
	App      Hash
	LFlag    byte
	RFlag    byte
	// Other   []byte
}

func (p *processer) syncInfo(from uint64, ops uint8, data []byte) {
	p.Event(logSync{}, "syncInfo", []byte{ops}, p.Encode(0, from), data)
	switch ops {
	case SyncOpsMoveCoin:
		var mi syncMoveInfo
		p.Decode(0, data, &mi)
		p.adminTransfer(Address{}, mi.User, mi.Value)
		if Switch.SWMove {
			db := p.GetDB(statMove{})
			id := db.GetInt(mi.User[:]) + 1
			db.SetValue(mi.User[:], id, TimeYear)
			rk := append(mi.User[:], runtime.Encode(id)...)
			db.Set(rk, data, TimeMonth)
		}
	case SyncOpsNewChain:
		var nc syncNewChain
		p.Decode(0, data, &nc)
		assert(p.Chain == nc.Chain)
		assert(p.ID == 1)
		assert(p.Key == nc.FirstBlock)
		p.ParentID = nc.ParentID
		p.Time = nc.Time
		p.pDbStat.Set([]byte{StatParentKey}, nc.BlockKey[:], TimeDay)
		p.pDbStat.SetValue([]byte{StatBaseInfo}, p.BaseInfo, maxDbLife)
		p.pDbStat.SetValue([]byte{StatGuerdon}, nc.Guerdon, maxDbLife)
		p.pDbStat.SetValue([]byte{StatAdmin}, nc.AdminList, maxDbLife)
		p.pDbStat.SetValue([]byte{StatHashPower}, nc.HashPower, maxDbLife)
		for _, it := range nc.AdminList {
			if it.Empty() {
				continue
			}
			var admin = AdminInfo{1, 0}
			p.pDbAdmin.SetValue(it[:], admin, maxDbLife)
		}
		p.registerMiner(nc.Producer)
		p.Event(dbTransInfo{}, "new_chain_ack", []byte{2})
	case SyncOpsMiner:
		rm := syncRegMiner{}
		p.Decode(0, data, &rm)
		p.registerMiner(rm.User)
	case SyncOpsBroadcast:
		br := BroadcastInfo{}
		n := p.Decode(0, data, &br)
		if p.LeftChildID > 0 {
			p.addSyncInfo(p.Chain*2, ops, data)
		} else {
			br.LFlag = 1
		}
		if p.RightChildID > 0 {
			p.addSyncInfo(p.Chain*2+1, ops, data)
		} else {
			br.RFlag = 1
		}
		d := p.Encode(0, br)
		d = append(d, data[n:]...)
		if br.LFlag > 0 && br.RFlag > 0 {
			p.addSyncInfo(p.Chain/2, SyncOpsBroadcastAck, nil)
			p.Event(logSync{}, "SyncOpsBroadcastAck", d)
		}
		p.pDbStat.Set([]byte{StatBroadcast}, d, logLockTime*2)
	case SyncOpsBroadcastAck:
		data, life := p.pDbStat.Get([]byte{StatBroadcast})
		if len(data) == 0 {
			return
		}
		br := BroadcastInfo{}
		n := p.Decode(0, data, &br)
		if from%2 == 0 {
			br.LFlag = 1
		} else {
			br.RFlag = 1
		}
		d := p.Encode(0, br)
		d = append(d, data[n:]...)
		p.pDbStat.Set([]byte{StatBroadcast}, d, life)
		if br.LFlag > 0 && br.RFlag > 0 {
			if p.Chain > 1 {
				p.addSyncInfo(p.Chain/2, SyncOpsBroadcastAck, nil)
			}
			p.Event(logSync{}, "SyncOpsBroadcastAck", d)
		}
	}
}
