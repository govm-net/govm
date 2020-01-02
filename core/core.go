package ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c

import (
	"github.com/lengzhao/govm/runtime"
)

type dbBlockData struct{}
type dbTransactionData struct{}
type dbTransInfo struct{}
type dbCoin struct{}
type dbAdmin struct{}
type dbStat struct{}
type dbApp struct{}
type dbDepend struct{}
type dbMining struct{}
type logBlockInfo struct{}
type logSync struct{}

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
	free  bool
	p     *processer
}

// Log Type definition of a log. Log data can be read on other chains. Unable to overwrite the existing data.
type Log struct {
	owner interface{}
	p     *processer
}

// AppInfo App info in database
type AppInfo struct {
	Account Address
	LineSum uint64
	Life    uint64
	Flag    uint8
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
	sInfo              tSyncInfo
	pDbBlockData       *DB
	pDbTransactionData *DB
	pDbTransInfo       *DB
	pDbCoin            *DB
	pDbAdmin           *DB
	pDbStat            *DB
	pDbApp             *DB
	pDbDepend          *DB
	pDbMining          *DB
	pLogSync           *Log
	pLogBlockInfo      *Log
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

	maxBlockInterval   = 1 * TimeMinute
	minBlockInterval   = 10 * TimeMillisecond
	blockSizeLimit     = 1 << 20
	blockSyncMin       = 8 * TimeMinute
	blockSyncMax       = 10 * TimeMinute
	defauldbLife       = 6 * TimeMonth
	adminLife          = 10 * TimeYear
	acceptTransTime    = 10 * TimeDay
	logLockTime        = 3 * TimeDay
	maxDbLife          = 1 << 50
	maxGuerdon         = 5000000000000
	minGuerdon         = 50000
	prefixOfPlublcAddr = 255
	guerdonUpdateCycle = 500000
	depositCycle       = 50000
	hateRatioMax       = 1 << 30
	minerNum           = 11
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
	StatHateRatio
	StatParentKey
	StatSyncTime
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
	// OpsDisableAdmin disable admin
	OpsDisableAdmin
)

var (
	// gPublicAddr The address of a public account for the preservation of additional rewards.
	gPublicAddr = Address{prefixOfPlublcAddr, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
	author      = Address{2, 152, 64, 16, 49, 156, 211, 70, 89, 247, 252, 178, 11, 49, 214, 21, 216, 80, 171, 50, 202, 147, 6, 24}
)

// Empty Check whether Hash is empty
func (h Hash) Empty() bool {
	return h == (Hash{})
}

// Empty Check where Address is empty
func (a Address) Empty() bool {
	return a == (Address{})
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

func (p *processer) initEnv(chain uint64, flag []byte) {
	bit := 32 << (^uint(0) >> 63)
	assertMsg(bit == 64, "only support 64bit system")
	p.pDbBlockData = p.GetDB(dbBlockData{})
	p.pDbTransactionData = p.GetDB(dbTransactionData{})
	p.pDbTransInfo = p.GetDB(dbTransInfo{})
	p.pDbCoin = p.GetDB(dbCoin{})
	p.pDbCoin.free = true
	p.pDbAdmin = p.GetDB(dbAdmin{})
	p.pDbAdmin.free = true
	p.pDbStat = p.GetDB(dbStat{})
	p.pDbStat.free = true
	p.pDbApp = p.GetDB(dbApp{})
	p.pDbApp.free = true
	p.pDbDepend = p.GetDB(dbDepend{})
	p.pDbDepend.free = true
	p.pDbMining = p.GetDB(dbMining{})
	p.pDbMining.free = true
	p.pLogBlockInfo = p.GetLog(logBlockInfo{})
	p.pLogSync = p.GetLog(logSync{})
	runt := new(runtime.TRuntime)
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
	if data == nil {
		return hashKey
	}
	hash := p.GetHash(data)
	n := p.Decode(0, hash, &hashKey)
	assert(n == HashLen)
	return hashKey
}

/*-------------------------DB----------------------------------*/

// Set Storage data. the record will be deleted when life=0 or value=nil
func (d *DB) Set(key, value []byte, life uint64) {
	assertMsg(life <= maxDbLife, "too long of life")
	assertMsg(len(key) > 0, "empty key")
	assertMsg(len(value) < 40960, "value size over limit")
	size := uint64(len(key) + len(value))
	if d.free {
	} else if life == 0 || len(value) == 0 {
		value = nil
		life = 0
	} else if size > 100 {
		assertMsg(life <= 100*TimeYear, "too long of life")
	}
	life += d.p.Time
	d.p.DbSet(d.owner, key, value, life)
}

// SetInt Storage uint64 data
func (d *DB) SetInt(key []byte, value uint64, life uint64) {
	v := d.p.Encode(0, value)
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

// GetInt read uint64 data from database
func (d *DB) GetInt(key []byte) uint64 {
	v, _ := d.Get(key)
	if v == nil {
		return 0
	}
	var val uint64
	n := d.p.Decode(0, v, &val)
	assertMsg(n == len(v), "data length != 8")
	return val
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
	if val == nil {
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

// TransferAccounts pTransfer based on the app private object
func (p *processer) TransferAccounts(owner interface{}, payee Address, value uint64) {
	payer := p.GetAppAccount(owner)
	assert(!payee.Empty())
	assert(!payer.Empty())
	p.adminTransfer(payer, payee, value)
}

func (p *processer) getAccount(addr Address) (uint64, uint64) {
	v, l := p.pDbCoin.Get(addr[:])
	if v == nil {
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
			p.pDbCoin.SetInt(payer[:], 0, 0)
		} else {
			p.pDbCoin.SetInt(payer[:], v, maxDbLife)
		}
	}
	if !payee.Empty() {
		if payeeV == value {
			// p.ConsumeEnergy(p.BaseOpsEnergy * 1000)
			payeeL = maxDbLife
		}
		p.pDbCoin.SetInt(payee[:], payeeV, payeeL)
	}

	p.Event(dbCoin{}, "pTransfer", payer[:], payee[:], p.Encode(0, value), p.Encode(0, payeeV))
}

// Miner miner info
type Miner struct {
	Miner [minerNum]Address
	Cost  [minerNum]uint64
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
	return out + 10
}

func run(chain uint64, flag []byte) {
	var proc processer
	proc.initEnv(chain, flag)
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

// Block Block structure, full block data need +(Sign []byte) + (TransList []Hash)
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
	Size          uint32
	//transList   []Hash
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
	transList := signData[n:]

	rst := p.Recover(block.Producer[:], sign, signData)
	assert(rst)

	assert(p.Key == block.Previous)
	assert(p.ID+1 == block.Index)
	assert(block.Producer[0] != prefixOfPlublcAddr)

	p.Time = block.Time
	p.Key = key
	p.ID = block.Index
	p.pDbStat.Set([]byte{StatBaseInfo}, p.Encode(0, p.BaseInfo), maxDbLife)

	//sync info from other chains
	p.syncInfos()

	if p.ID == 1 {
		p.processFirstBlock(block, transList)
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

	sizeLimit := p.pDbStat.GetInt([]byte{StatBlockSizeLimit})
	assert(uint64(block.Size) <= sizeLimit)
	avgSize := p.pDbStat.GetInt([]byte{StatAvgBlockSize})
	avgSize = (avgSize*(depositCycle-1) + uint64(block.Size)) / depositCycle
	p.pDbStat.SetInt([]byte{StatAvgBlockSize}, avgSize, maxDbLife)

	hpLimit := p.pDbStat.GetInt([]byte{StatHashPower})
	blockInterval := p.pDbStat.GetInt([]byte{StatBlockInterval})
	decT := block.Time - preB.Time
	if block.Index == 2 && block.Chain > 1 {
		assert(decT == blockSyncMax+blockSyncMin+maxBlockInterval)
		k, _ := p.pDbStat.Get([]byte{StatParentKey})
		var parent Hash
		p.Decode(0, k, &parent)
		assert(parent == block.Parent)
	} else {
		assert(decT == blockInterval)
	}
	hp := getHashPower(key)
	assert(hp > 2)
	assert(hp >= hpLimit*8/10000)
	hp = hp + hpLimit - hpLimit/1000
	p.pDbStat.SetInt([]byte{StatHashPower}, hp, maxDbLife)

	if p.Chain == 1 {
		assert(block.Parent.Empty())
	} else {
		assert(!block.Parent.Empty())
		b := p.getBlockLog(p.Chain/2, block.Parent)
		assert(b != nil)
		info.ParentID = b.Index
		p.ParentID = b.Index
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
	p.pDbStat.Set([]byte{StatBaseInfo}, p.Encode(0, p.BaseInfo), maxDbLife)

	var size uint64
	if !block.TransListHash.Empty() {
		size = p.processTransList(info, block.TransListHash, transList)
	}
	assert(size == uint64(block.Size))

	//Mining guerdon
	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
	p.adminTransfer(Address{}, block.Producer, guerdon)
	p.adminTransfer(Address{}, author, guerdon/100)

	// Every pre year, the reward is halved
	if block.Index%guerdonUpdateCycle == 0 {
		guerdon = guerdon*9/10 + minGuerdon
		p.pDbStat.SetInt([]byte{StatGuerdon}, guerdon, maxDbLife)
	}

	val := p.pDbCoin.GetInt(gPublicAddr[:]) / depositCycle
	p.adminTransfer(gPublicAddr, author, val/100)
	p.adminTransfer(gPublicAddr, block.Producer, val)

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

func (p *processer) processFirstBlock(block Block, transList []byte) {
	assert(block.Chain == 0)
	assert(block.Producer == author)
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

	//Used to create the first app
	p.adminTransfer(Address{}, author, maxGuerdon)
	p.adminTransfer(Address{}, gPublicAddr, maxGuerdon)

	//save block info
	stream := p.Encode(0, blockInfo)
	empHash := Hash{}
	assert(p.pLogBlockInfo.Write(empHash[:], stream))
	assert(p.pLogBlockInfo.Write(p.Key[:], stream))
	assert(p.pLogBlockInfo.Write(p.Encode(0, block.Index), p.Key[:]))

	if p.Chain == 1 {
		p.pDbStat.SetInt([]byte{StatGuerdon}, maxGuerdon, maxDbLife)
	}
	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})

	p.pDbStat.SetInt([]byte{StatBlockSizeLimit}, blockSizeLimit, maxDbLife)
	p.pDbStat.SetInt([]byte{StatHashPower}, 10000, maxDbLife)
	p.pDbStat.SetInt([]byte{StatBlockInterval}, getBlockInterval(p.Chain), maxDbLife)
	p.pDbStat.Set([]byte{StatFirstBlockKey}, p.Key[:], maxDbLife)
	p.pDbStat.SetInt([]byte{StatHateRatio}, hateRatioMax, maxDbLife)

	p.registerMiner(block.Producer, 2, guerdon/10)

	saveInfo := AppInfo{}
	saveInfo.Flag = AppFlagImport | AppFlagPlublc
	saveInfo.Life = maxDbLife + p.Time
	saveInfo.Account = gPublicAddr
	saveInfo.LineSum = 945
	appName := p.GetAppName(dbStat{})
	p.pDbApp.Set(appName[:], p.Encode(0, saveInfo), maxDbLife)

	p.Event(logBlockInfo{}, "finish_block", p.Key[:])
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

func (p *processer) processTransList(block BlockInfo, key Hash, data []byte) uint64 {
	assert(data != nil)
	size := len(data)
	num := size / HashLen
	assert(num*HashLen == size)

	transList := []Hash{}
	for i := 0; i < num; i++ {
		var h Hash
		n := p.Decode(0, data, &h)
		data = data[n:]
		transList = append(transList, h)
	}

	transListBack := transList

	for len(transList) > 1 {
		tmpList := []Hash{}
		if len(transList)%2 != 0 {
			transList = append(transList, Hash{})
		}
		for i := 0; i < len(transList)/2; i++ {
			d := make([]byte, 0, HashLen*2)
			d = append(d, transList[2*i][:]...)
			d = append(d, transList[2*i+1][:]...)
			h := p.getHash(d)
			tmpList = append(tmpList, h)
		}
		transList = tmpList
	}
	assert(len(transList) == 1)
	assert(transList[0] == key)

	var out uint64
	for i := 0; i < len(transListBack); i++ {
		out += p.processTransaction(block, transListBack[i])
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
	assert(ti == nil)
	data, _ := p.pDbTransactionData.Get(key[:])
	signLen := data[0]
	assert(signLen > 30)
	k := p.getHash(data)
	assert(k == key)

	sign := data[1 : signLen+1]
	signData := data[signLen+1:]

	trans := Transaction{}
	n := p.Decode(0, signData, &trans.TransactionHead)
	trans.data = signData[n:]
	dataLen := len(trans.data)

	assert(p.Recover(trans.User[:], sign, signData))

	assertMsg(trans.Time <= block.Time, "trans_newer")
	assert(trans.Time+acceptTransTime > block.Time)
	assert(trans.User[0] != prefixOfPlublcAddr)
	if block.Index == 1 {
		assert(trans.Chain == 0)
		trans.Chain = p.Chain
	}
	assert(trans.Chain == p.Chain)
	assert(trans.Energy > uint64(dataLen))
	p.pDbStat.Set([]byte{StatTransKey}, key[:], defauldbLife)

	info := TransInfo{}
	info.BlockID = block.Index
	info.User = trans.User
	info.Ops = trans.Ops
	info.Cost = trans.Cost
	p.pDbTransInfo.Set(key[:], p.Encode(0, info), defauldbLife)

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
	case OpsDisableAdmin:
		assert(dataLen < 300)
		p.pDisableAdmin(trans.User, trans.data, trans.Cost)
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
}

// syncMoveInfo sync Information of move out
type syncMoveInfo struct {
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
			assert(p.LeftChildID > 0)
		} else {
			assert(p.RightChildID > 0)
		}
	}

	p.adminTransfer(t.User, Address{}, t.Cost)
	stru := syncMoveInfo{t.User, t.Cost}
	p.addSyncInfo(chain, SyncOpsMoveCoin, p.Encode(0, stru))
}

/*********************** chain ****************************/

type syncNewChain struct {
	Chain      uint64
	BlockKey   Hash
	Guerdon    uint64
	ParentID   uint64
	Time       uint64
	Producer   Address
	FirstBlock Hash
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
		// assert(p.LeftChildID > depositCycle)
		p.RightChildID = 1
	}

	if newChain > 3 {
		avgSize := p.pDbStat.GetInt([]byte{StatAvgBlockSize})
		scale := avgSize * 10 / blockSizeLimit
		assert(scale > 1)
		cost := 10000*guerdon + 10*maxGuerdon
		cost = cost >> scale
		assert(t.Cost >= cost)
		p.pDbStat.SetInt([]byte{StatAvgBlockSize}, 0, maxDbLife)
	}
	p.adminTransfer(t.User, gPublicAddr, t.Cost)

	p.pDbStat.Set([]byte{StatBaseInfo}, p.Encode(0, p.BaseInfo), maxDbLife)
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
	p.pDbApp.Set(appName[:], p.Encode(0, saveInfo), life)
	if len(deps) > 0 {
		p.pDbDepend.Set(appName[:], deps, life)
	}
}

func (p *processer) pRunApp(t Transaction) {
	var name Hash
	n := p.Decode(0, t.data, &name)
	info := p.GetAppInfo(name)
	assertMsg(info != nil, "app not exist")
	assertMsg(info.Flag&AppFlagRun != 0, "app unable run")
	assertMsg(info.Life >= p.Time, "app expire")
	val, _ := p.getAccount(t.User)
	assertMsg(t.Cost <= val, "not enough cost")
	p.RunApp(name[:], t.User[:], t.data[n:], t.Energy, t.Cost)
	p.adminTransfer(t.User, info.Account, t.Cost)
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
	p.pDbApp.Set(AppName[:], p.Encode(0, app), app.Life-p.Time)
	// t := p.BaseOpsEnergy * (life + TimeDay) / TimeHour
	// p.ConsumeEnergy(t)
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

// RegMiner info of register miner
type RegMiner struct {
	Chain uint64
	Index uint64
}

type syncRegMiner struct {
	Index uint64
	Cost  uint64
	User  Address
}

func (p *processer) registerOtherChainMiner(user Address, chain, index, cost uint64) {
	assert(chain/2 == p.Chain || p.Chain/2 == chain)
	assert(chain > 0)
	if chain > p.Chain {
		if chain%2 == 0 {
			assert(p.LeftChildID > 0)
		} else {
			assert(p.RightChildID > 0)
		}
	}
	p.adminTransfer(user, Address{}, cost)
	info := syncRegMiner{index, cost, user}
	p.addSyncInfo(chain, SyncOpsMiner, p.Encode(0, info))
}

func (p *processer) registerMiner(user Address, index, cost uint64) bool {
	miner := Miner{}
	stream, _ := p.pDbMining.Get(p.Encode(0, index))
	if len(stream) > 0 {
		p.Decode(0, stream, &miner)
		if cost <= miner.Cost[minerNum-1] {
			return false
		}
		for i := 0; i < minerNum; i++ {
			if user == miner.Miner[i] {
				return false
			}
		}
		p.adminTransfer(Address{}, miner.Miner[minerNum-1], miner.Cost[minerNum-1])
	}
	m := user
	c := cost
	for i := 0; i < minerNum; i++ {
		if c > miner.Cost[i] {
			miner.Miner[i], m = m, miner.Miner[i]
			miner.Cost[i], c = c, miner.Cost[i]
		}
	}
	p.pDbMining.Set(p.Encode(0, index), p.Encode(0, miner), defauldbLife)
	return true
}

func (p *processer) pRegisterMiner(t Transaction) {
	guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
	assert(t.Cost >= 3*guerdon)
	info := RegMiner{}
	p.Decode(0, t.data, &info)

	if info.Chain != 0 && info.Chain != p.Chain {
		assert(t.Energy > 1000000)
		if info.Chain > p.Chain {
			assert(info.Chain/2 == p.Chain)
		} else {
			assert(info.Chain == p.Chain/2)
		}
		p.registerOtherChainMiner(t.User, info.Chain, info.Index, t.Cost)
		return
	}

	assert(info.Index > p.ID+20)
	assert(p.ID+2*depositCycle > info.Index)

	rst := p.registerMiner(t.User, info.Index, t.Cost)
	if rst {
		p.adminTransfer(t.User, Address{}, t.Cost)
	}
}

/*------------------------------api---------------------------------------*/

// AdminInfo register as a admin
type AdminInfo struct {
	App   Hash
	Cost  uint64
	Index uint8
}

// pDisableAdmin disable admin
func (p *processer) pDisableAdmin(user Address, data []byte, cost uint64) {
	assert(cost > minGuerdon)
	index := data[0]
	stream, life := p.pDbAdmin.Get([]byte{index})
	if life <= p.Time {
		return
	}
	var app Hash
	p.Decode(0, data[1:], &app)

	p.adminTransfer(user, gPublicAddr, cost)
	cost = cost / 10

	info := AdminInfo{}
	p.Decode(0, stream, &info)
	if info.App != app {
		return
	}

	if cost >= info.Cost {
		p.pDbAdmin.Set(info.App[:], nil, 0)
		p.pDbAdmin.Set([]byte{index}, nil, 0)
		p.Event(dbStat{}, "disable admin", []byte{index}, info.App[:])
		return
	}
	info.Cost = info.Cost - cost
	p.pDbAdmin.Set([]byte{index}, p.Encode(0, info), life)
}

// ops of sync
const (
	SyncOpsMoveCoin = iota
	SyncOpsNewChain
	SyncOpsMiner
	SyncOpsBroadcast
	SyncOpsBroadcastAck
	SyncOpsHateRatio
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
	stream, _ := p.pDbMining.Get(p.Encode(0, p.ID-depositCycle))
	if len(stream) > 0 {
		mi := Miner{}
		p.Decode(0, stream, &mi)
		for i := 0; i < minerNum; i++ {
			p.adminTransfer(Address{}, mi.Miner[i], mi.Cost[i]+minGuerdon)
		}
	}

	stream, _ = p.pDbStat.Get([]byte{StatSyncInfo})
	if len(stream) > 0 {
		p.Decode(0, stream, &p.sInfo)
	}

	p.syncFromParent()
	p.syncFromLeftChild()
	p.syncFromRightChild()
	p.pDbStat.Set([]byte{StatSyncInfo}, p.Encode(0, p.sInfo), maxDbLife)
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
	p.pDbStat.Set([]byte{StatSyncInfo}, p.Encode(0, p.sInfo), maxDbLife)
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
		p.pDbStat.Set([]byte{StatSyncInfo}, p.Encode(0, p.sInfo), maxDbLife)
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
		p.pDbStat.Set([]byte{StatSyncInfo}, p.Encode(0, p.sInfo), maxDbLife)
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
		p.pDbStat.Set([]byte{StatSyncInfo}, p.Encode(0, p.sInfo), maxDbLife)
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
	case SyncOpsNewChain:
		var nc syncNewChain
		p.Decode(0, data, &nc)
		assert(p.Chain == nc.Chain)
		assert(p.ID == 1)
		assert(p.Key == nc.FirstBlock)
		p.ParentID = nc.ParentID
		p.Time = nc.Time
		p.pDbStat.Set([]byte{StatParentKey}, nc.BlockKey[:], TimeDay)
		p.pDbStat.Set([]byte{StatBaseInfo}, p.Encode(0, p.BaseInfo), maxDbLife)
		p.pDbStat.SetInt([]byte{StatGuerdon}, nc.Guerdon, maxDbLife)
		p.registerMiner(nc.Producer, 2, nc.Guerdon)
		p.Event(dbTransInfo{}, "new_chain_ack", []byte{2})
	case SyncOpsMiner:
		rm := syncRegMiner{}
		p.Decode(0, data, &rm)

		if rm.Index < p.ID+20 || rm.Index > p.ID+2*depositCycle {
			p.adminTransfer(Address{}, rm.User, rm.Cost)
			return
		}
		guerdon := p.pDbStat.GetInt([]byte{StatGuerdon})
		if rm.Cost < 3*guerdon {
			p.adminTransfer(Address{}, rm.User, rm.Cost)
			return
		}
		rst := p.registerMiner(rm.User, rm.Index, rm.Cost)
		if !rst {
			p.adminTransfer(Address{}, rm.User, rm.Cost)
		}
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
	case SyncOpsHateRatio:
		var ratio uint64
		p.Decode(0, data, &ratio)
		p.pDbStat.SetInt([]byte{StatHateRatio}, ratio, maxDbLife)
		if p.LeftChildID > 0 {
			p.addSyncInfo(2*p.Chain, SyncOpsHateRatio, data)
		}
		if p.RightChildID > 0 {
			p.addSyncInfo(2*p.Chain+1, SyncOpsHateRatio, data)
		}
	}
}
