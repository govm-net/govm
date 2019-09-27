package a9edcee1a25950643c09476b7c039eb8aec09141a8d0e80051fd52a0e37bc60fe

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

// IRuntime The interface that the executive needs to register
type IRuntime interface {
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
	//Consume energy
	ConsumeEnergy(energy uint64)
	//OtherOps  extensional api
	OtherOps(user interface{}, ops int, data []byte) []byte
}

// DB Type definition of a database.
type DB struct {
	owner interface{}
	free  bool
}

// Log Type definition of a log. Log data can be read on other chains. Unable to overwrite the existing data.
type Log struct {
	owner interface{}
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
	StatMax
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
	//db instance
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
	gBS                BaseInfo
	gTransacKey        Hash

	// gPublicAddr The address of a public account for the preservation of additional rewards.
	gPublicAddr = Address{prefixOfPlublcAddr, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
	author      = Address{2, 152, 64, 16, 49, 156, 211, 70, 89, 247, 252, 178, 11, 49, 214, 21, 216, 80, 171, 50, 202, 147, 6, 24}
	gRuntime    IRuntime
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

func init() {
	bit := 32 << (^uint(0) >> 63)
	assert(bit == 64)
	pDbBlockData = GetDB(dbBlockData{})
	pDbTransactionData = GetDB(dbTransactionData{})
	pDbTransInfo = GetDB(dbTransInfo{})
	pDbCoin = GetDB(dbCoin{})
	pDbCoin.free = true
	pDbAdmin = GetDB(dbAdmin{})
	pDbAdmin.free = true
	pDbStat = GetDB(dbStat{})
	pDbStat.free = true
	pDbApp = GetDB(dbApp{})
	pDbApp.free = true
	pDbDepend = GetDB(dbDepend{})
	pDbDepend.free = true
	pDbMining = GetDB(dbMining{})
	pDbMining.free = true
	pLogBlockInfo = GetLog(logBlockInfo{})
	pLogSync = GetLog(logSync{})
}

// RegisterRuntime Registration processing interface
func RegisterRuntime(in IRuntime) {
	assert(gRuntime == nil)
	gRuntime = in
	getEnv()
}

func getEnv() {
	stream, _ := gRuntime.DbGet(pDbStat.owner, []byte{StatBaseInfo})
	if len(stream) > 0 {
		Decode(0, stream, &gBS)
	}
	stream, _ = pDbStat.Get([]byte{StatTransKey})
	if len(stream) > 0 {
		Decode(0, stream, &gTransacKey)
	}
}

// GetHash get data hash
func GetHash(data []byte) Hash {
	hashKey := Hash{}
	if data == nil {
		return hashKey
	}
	gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy * 20)
	hash := gRuntime.GetHash(data)
	n := Decode(0, hash, &hashKey)
	assert(n == HashLen)
	return hashKey
}

// Encode Encoding data into data streams.
func Encode(typ uint8, in interface{}) []byte {
	return gRuntime.Encode(typ, in)
}

// Decode The data stream is filled into a variable of the specified type.
func Decode(typ uint8, in []byte, out interface{}) int {
	return gRuntime.Decode(typ, in, out)
}

// Recover recover sign
func Recover(address, sign, msg []byte) bool {
	gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy * 50)
	return gRuntime.Recover(address, sign, msg)
}

/*-------------------------DB----------------------------------*/

// Set Storage data. the record will be deleted when life=0 or value=nil
func (d *DB) Set(key, value []byte, life uint64) {
	assert(life <= maxDbLife)
	assert(len(key) > 0)
	assert(len(value) < 40960)
	size := uint64(len(key) + len(value))
	if d.free {
		gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy)
	} else if life == 0 || len(value) == 0 {
		value = nil
		life = 0
		gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy)
	} else if size > 100 {
		assert(life <= 100*TimeYear)
		t := gBS.BaseOpsEnergy * size * (life + TimeHour) / (TimeHour * 10)
		gRuntime.ConsumeEnergy(t)
	} else {
		l := gRuntime.DbGetLife(d.owner, key)
		if l < gBS.Time {
			l = 0
		} else {
			l -= gBS.Time
		}
		var t uint64
		if life > l {
			t = gBS.BaseOpsEnergy * 10 * (life + TimeHour - l) / TimeHour
		} else {
			t = gBS.BaseOpsEnergy * 10
		}
		gRuntime.ConsumeEnergy(t)
	}
	life += gBS.Time
	gRuntime.DbSet(d.owner, key, value, life)
}

// SetInt Storage uint64 data
func (d *DB) SetInt(key []byte, value uint64, life uint64) {
	v := Encode(0, value)
	d.Set(key, v, life)
}

// Get Read data from database
func (d *DB) Get(key []byte) ([]byte, uint64) {
	assert(len(key) > 0)
	gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy)
	out, life := gRuntime.DbGet(d.owner, key)
	if life <= gBS.Time {
		return nil, 0
	}
	return out, (life - gBS.Time)
}

// GetInt read uint64 data from database
func (d *DB) GetInt(key []byte) uint64 {
	v, _ := d.Get(key)
	if v == nil {
		return 0
	}
	var val uint64
	n := Decode(0, v, &val)
	assert(n == len(v))
	return val
}

// GetDB Through the private structure in app, get a DB of app, the parameter must be a structure, not a pointer.
// such as: owner = tAppInfo{}
func GetDB(owner interface{}) *DB {
	out := DB{}
	out.owner = owner
	return &out
}

// Write Write log,if exist the key,return false.the key and value can't be nil.
func (l *Log) Write(key, value []byte) bool {
	assert(len(key) > 0)
	assert(len(value) > 0)
	assert(len(value) < 1024)

	life := gRuntime.LogReadLife(l.owner, key)
	if life+logLockTime >= gBS.Time {
		return false
	}
	life = TimeYear

	t := 10 * gBS.BaseOpsEnergy * uint64(len(key)+len(value)) * life / TimeDay
	gRuntime.ConsumeEnergy(t)
	life += gBS.Time
	gRuntime.LogWrite(l.owner, key, value, life)
	return true
}

// Read Read log
func (l *Log) Read(chain uint64, key []byte) []byte {
	assert(len(key) > 0)
	if chain == 0 {
		chain = gBS.Chain
	}
	dist := getLogicDist(chain, gBS.Chain)
	gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy * (1 + dist*10))
	minLife := gBS.Time - blockSyncMax*dist
	maxLife := minLife + TimeYear
	out, life := gRuntime.LogRead(l.owner, chain, key)
	if life < minLife || life > maxLife {
		return nil
	}
	return out
}

// read read log from parent/child/self
func (l *Log) read(chain uint64, key []byte) []byte {
	if chain == 0 {
		chain = gBS.Chain
	}
	assert(chain/2 == gBS.Chain || gBS.Chain/2 == chain || gBS.Chain == chain)
	minLife := gBS.Time
	if chain != gBS.Chain {
		minLife -= blockSyncMin
	}
	maxLife := minLife + TimeYear

	out, life := gRuntime.LogRead(l.owner, chain, key)
	if life < minLife || life > maxLife {
		return nil
	}
	return out
}

// GetLog Through the private structure in app, get a Log of app, the parameter must be a structure, not a pointer.
func GetLog(owner interface{}) *Log {
	out := Log{}
	out.owner = owner
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
func GetAppName(in interface{}) Hash {
	gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy)
	out := Hash{}
	name := gRuntime.GetAppName(in)
	n := Decode(0, name, &out)
	assert(n == len(name))
	return out
}

// GetAppInfo get app information
func GetAppInfo(name Hash) *AppInfo {
	out := AppInfo{}
	val, _ := pDbApp.Get(name[:])
	if val == nil {
		return nil
	}
	Decode(0, val, &out)
	return &out
}

// GetAppAccount  Get the owner Address of the app
func GetAppAccount(in interface{}) Address {
	app := GetAppName(in)
	assert(!app.Empty())
	info := GetAppInfo(app)
	return info.Account
}

/*-------------------------------------Coin------------------------*/

// TransferAccounts pTransfer based on the app private object
func TransferAccounts(owner interface{}, payee Address, value uint64) {
	payer := GetAppAccount(owner)
	assert(!payee.Empty())
	assert(!payer.Empty())
	adminTransfer(payer, payee, value)
}

func getAccount(addr Address) (uint64, uint64) {
	v, l := pDbCoin.Get(addr[:])
	if v == nil {
		return 0, 0
	}
	var val uint64
	n := Decode(0, v, &val)
	assert(n == len(v))
	return val, l
}

func adminTransfer(payer, payee Address, value uint64) {
	if payer == payee {
		return
	}
	if value == 0 {
		return
	}

	payeeV, payeeL := getAccount(payee)
	payeeV += value
	if payeeV < value {
		return
	}
	if !payer.Empty() {
		v := pDbCoin.GetInt(payer[:])
		assert(v >= value)
		v -= value
		if v == 0 {
			pDbCoin.SetInt(payer[:], 0, 0)
		} else {
			pDbCoin.SetInt(payer[:], v, maxDbLife)
		}
	}
	if !payee.Empty() {
		if payeeV == value {
			gRuntime.ConsumeEnergy(gBS.BaseOpsEnergy * 1000)
			payeeL = maxDbLife
		}
		pDbCoin.SetInt(payee[:], payeeV, payeeL)
	}

	Event(dbCoin{}, "pTransfer", payer[:], payee[:], Encode(0, value), Encode(0, payeeV))
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

func run(user, in []byte, cost uint64) {
	info := runParam{}
	n := Decode(0, in, &info)
	assert(n == len(in))
	assert(info.Chain != 0)
	assert(!info.Key.Empty())
	if gBS.Chain == 0 {
		gBS.Chain = info.Chain
	}
	if gBS.BaseOpsEnergy == 0 {
		gBS.BaseOpsEnergy = getBaseOpsEnergy(info.Chain)
	}

	assert(gBS.Chain > 0)
	assert(gBS.Chain == info.Chain)

	processBlock(info.Chain, info.Key)
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

func processBlock(chain uint64, key Hash) {
	block := Block{}
	data, _ := pDbBlockData.Get(key[:])
	signLen := data[0]
	assert(signLen > 30)
	assert(signLen < 250)

	k := GetHash(data)
	assert(key == k)

	sign := data[1 : signLen+1]
	signData := data[signLen+1:]

	n := Decode(0, signData, &block)
	transList := signData[n:]

	rst := Recover(block.Producer[:], sign, signData)
	assert(rst)

	assert(gBS.Key == block.Previous)
	assert(gBS.ID+1 == block.Index)
	assert(block.Producer[0] != prefixOfPlublcAddr)

	gBS.Time = block.Time
	gBS.Key = key
	gBS.ID = block.Index
	pDbStat.Set([]byte{StatBaseInfo}, Encode(0, gBS), maxDbLife)

	//sync info from other chains
	syncInfos()

	if gBS.ID == 1 {
		processFirstBlock(block, transList)
		return
	}
	assert(chain == block.Chain)
	assert(!block.Previous.Empty())
	info := BlockInfo{}
	preB := getBlockLog(0, block.Previous)
	assert(preB != nil)

	info.Index = block.Index
	info.Parent = block.Parent
	info.LeftChild = block.LeftChild
	info.RightChild = block.RightChild
	info.Time = block.Time
	info.Producer = block.Producer

	sizeLimit := pDbStat.GetInt([]byte{StatBlockSizeLimit})
	assert(block.Size <= uint32(sizeLimit))
	avgSize := pDbStat.GetInt([]byte{StatAvgBlockSize})
	avgSize = (avgSize*(depositCycle-1) + uint64(block.Size)) / depositCycle
	pDbStat.SetInt([]byte{StatAvgBlockSize}, avgSize, maxDbLife)

	hpLimit := pDbStat.GetInt([]byte{StatHashPower})
	blockInterval := pDbStat.GetInt([]byte{StatBlockInterval})
	decT := block.Time - preB.Time
	if block.Index == 2 && block.Chain > 1 {
		assert(decT == blockSyncMax+blockSyncMin+maxBlockInterval)
	} else {
		assert(decT == blockInterval)
	}
	hp := getHashPower(key)
	assert(hp > 2)
	assert(hp+3 >= hpLimit/1000)
	hp = hp + hpLimit - hpLimit/1000
	pDbStat.SetInt([]byte{StatHashPower}, hp, maxDbLife)

	if gBS.Chain == 1 {
		assert(block.Parent.Empty())
	} else {
		assert(!block.Parent.Empty())
		b := getBlockLog(gBS.Chain/2, block.Parent)
		assert(b != nil)
		info.ParentID = b.Index
		gBS.ParentID = b.Index
		assert(b.Time+blockSyncMax > block.Time)

		var cb *BlockInfo
		if gBS.Chain%2 == 0 {
			cb = getBlockLog(0, b.LeftChild)
		} else {
			cb = getBlockLog(0, b.RightChild)
		}
		assert(cb != nil)
	}

	if gBS.LeftChildID > 0 {
		assert(!block.LeftChild.Empty())
		b := getBlockLog(2*gBS.Chain, block.LeftChild)
		assert(b != nil)
		info.LeftChildID = b.Index
		gBS.LeftChildID = b.Index
		if b.Index > 1 {
			assert(b.Time+blockSyncMax > block.Time)
			pb := getBlockLog(0, b.Parent)
			assert(pb != nil)
		} else {
			firstKey := Hash{}
			stream, _ := pDbStat.Get([]byte{StatFirstBlockKey})
			Decode(0, stream, &firstKey)
			assert(firstKey == block.LeftChild)
			assert(b.Time+3*blockSyncMax > block.Time)
		}
	}
	if gBS.RightChildID > 0 {
		assert(!block.RightChild.Empty())
		b := getBlockLog(2*gBS.Chain+1, block.RightChild)
		assert(b != nil)
		info.RightChildID = b.Index
		gBS.RightChildID = b.Index
		if b.Index > 1 {
			assert(b.Time+blockSyncMax > block.Time)
			pb := getBlockLog(0, b.Parent)
			assert(pb != nil)
		} else {
			firstKey := Hash{}
			stream, _ := pDbStat.Get([]byte{StatFirstBlockKey})
			Decode(0, stream, &firstKey)
			assert(firstKey == block.LeftChild)
			assert(b.Time+3*blockSyncMax > block.Time)
		}
	}
	assert(info.ParentID >= preB.ParentID)
	assert(info.LeftChildID >= preB.LeftChildID)
	assert(info.RightChildID >= preB.RightChildID)
	assert(pLogBlockInfo.Write(key[:], Encode(0, info)))
	assert(pLogBlockInfo.Write(Encode(0, info.Index), key[:]))
	pDbStat.Set([]byte{StatBaseInfo}, Encode(0, gBS), maxDbLife)

	var size uint32
	if !block.TransListHash.Empty() {
		size = processTransList(info, block.TransListHash, transList)
	}
	assert(size == block.Size)

	//Mining guerdon
	guerdon := pDbStat.GetInt([]byte{StatGuerdon})
	if gBS.ID < 10 {
		registerMiner(block.Producer, gBS.ID+1, guerdon/10)
	}
	adminTransfer(Address{}, block.Producer, guerdon)
	adminTransfer(Address{}, author, guerdon/100)

	// Every pre year, the reward is halved
	if block.Index%guerdonUpdateCycle == 0 {
		guerdon = guerdon*9/10 + minGuerdon
		pDbStat.SetInt([]byte{StatGuerdon}, guerdon, maxDbLife)
	}

	val := pDbCoin.GetInt(gPublicAddr[:]) / depositCycle
	if val > 0 {
		adminTransfer(gPublicAddr, block.Producer, val)
	}

	Event(logBlockInfo{}, "finish_block", key[:])
}

type tSyncInfo struct {
	ToParentID       uint64
	ToLeftChildID    uint64
	ToRightChildID   uint64
	FromParentID     uint64
	FromLeftChildID  uint64
	FromRightChildID uint64
}

func processFirstBlock(block Block, transList []byte) {
	assert(block.Chain == 0)
	assert(block.Producer == author)
	assert(block.Previous.Empty())
	assert(block.Parent.Empty())
	assert(block.LeftChild.Empty())
	assert(block.RightChild.Empty())
	assert(!block.TransListHash.Empty())
	assert(block.Index == 1)
	blockInfo := BlockInfo{}
	blockInfo.Index = block.Index
	blockInfo.Parent = block.Parent
	blockInfo.LeftChild = block.LeftChild
	blockInfo.RightChild = block.RightChild
	blockInfo.Time = gBS.Time
	blockInfo.ParentID = gBS.ParentID
	blockInfo.Producer = block.Producer

	//Used to create the first app
	adminTransfer(Address{}, author, maxGuerdon)
	assert(HashLen == len(transList))
	processTransList(blockInfo, block.TransListHash, transList)
	adminTransfer(Address{}, gPublicAddr, maxGuerdon)

	//save block info
	stream := Encode(0, blockInfo)
	empHash := Hash{}
	assert(pLogBlockInfo.Write(empHash[:], stream))
	assert(pLogBlockInfo.Write(gBS.Key[:], stream))
	assert(pLogBlockInfo.Write(Encode(0, block.Index), gBS.Key[:]))

	if gBS.Chain == 1 {
		pDbStat.SetInt([]byte{StatGuerdon}, maxGuerdon, maxDbLife)
	}
	guerdon := pDbStat.GetInt([]byte{StatGuerdon})

	pDbStat.SetInt([]byte{StatBlockSizeLimit}, blockSizeLimit, maxDbLife)
	pDbStat.SetInt([]byte{StatHashPower}, 5000, maxDbLife)
	pDbStat.SetInt([]byte{StatBlockInterval}, getBlockInterval(gBS.Chain), maxDbLife)
	pDbStat.Set([]byte{StatFirstBlockKey}, gBS.Key[:], maxDbLife)
	pDbStat.SetInt([]byte{StatHateRatio}, hateRatioMax, maxDbLife)

	registerMiner(block.Producer, 2, guerdon/10)
	Event(logBlockInfo{}, "finish_block", gBS.Key[:])
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

func getBlockLog(chain uint64, key Hash) *BlockInfo {
	if chain == 0 {
		chain = gBS.Chain
	}
	assert(chain/2 == gBS.Chain || gBS.Chain/2 == chain || gBS.Chain == chain)
	stream := pLogBlockInfo.read(chain, key[:])
	if len(stream) == 0 {
		return nil
	}
	out := BlockInfo{}
	Decode(0, stream, &out)
	return &out
}

func processTransList(block BlockInfo, key Hash, data []byte) uint32 {
	assert(data != nil)
	size := len(data)
	num := size / HashLen
	assert(num*HashLen == size)

	transList := []Hash{}
	for i := 0; i < num; i++ {
		var h Hash
		n := Decode(0, data, &h)
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
			h := GetHash(d)
			tmpList = append(tmpList, h)
		}
		transList = tmpList
	}
	assert(len(transList) == 1)
	assert(transList[0] == key)

	var out uint32
	for i := 0; i < len(transListBack); i++ {
		out += processTransaction(block, transListBack[i])
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

func processTransaction(block BlockInfo, key Hash) uint32 {
	Event(dbTransInfo{}, "start_transaction", key[:])
	ti, _ := pDbTransInfo.Get(key[:])
	assert(ti == nil)
	data, _ := pDbTransactionData.Get(key[:])
	signLen := data[0]
	assert(signLen > 30)
	k := GetHash(data)
	assert(k == key)

	sign := data[1 : signLen+1]
	signData := data[signLen+1:]

	trans := Transaction{}
	n := Decode(0, signData, &trans.TransactionHead)
	trans.data = signData[n:]
	dataLen := len(trans.data)

	assert(Recover(trans.User[:], sign, signData))

	assert(trans.Time <= block.Time)
	assert(trans.Time+acceptTransTime > block.Time)
	assert(trans.User[0] != prefixOfPlublcAddr)
	if block.Index == 1 {
		assert(trans.Chain == 0)
		trans.Chain = gBS.Chain
	}
	assert(trans.Chain == gBS.Chain)
	assert(trans.Energy > uint64(dataLen))
	gTransacKey = key
	pDbStat.Set([]byte{StatTransKey}, key[:], defauldbLife)

	info := TransInfo{}
	info.BlockID = block.Index
	info.User = trans.User
	info.Ops = trans.Ops
	info.Cost = trans.Cost
	pDbTransInfo.Set(key[:], Encode(0, info), defauldbLife)

	trans.Energy /= 2
	adminTransfer(trans.User, gPublicAddr, trans.Energy)
	adminTransfer(trans.User, block.Producer, trans.Energy)
	switch trans.Ops {
	case OpsTransfer:
		assert(dataLen < 300)
		pTransfer(trans)
	case OpsMove:
		assert(dataLen < 300)
		pMove(trans)
	case OpsNewChain:
		assert(dataLen < 300)
		pNewChain(block.Producer, trans)
	case OpsNewApp:
		assert(dataLen < 81920)
		pNewApp(trans)
	case OpsRunApp:
		assert(dataLen < 2048)
		pRunApp(trans)
	case OpsUpdateAppLife:
		assert(dataLen < 300)
		pUpdateAppLife(trans)
	case OpsRegisterMiner:
		assert(dataLen < 300)
		pRegisterMiner(trans)
	case OpsDisableAdmin:
		assert(dataLen == 1)
		pDisableAdmin(trans.User, trans.data[0], trans.Cost)
	default:
		assert(false)
	}

	Event(dbTransInfo{}, "finish_transaction", key[:])
	return uint32(len(data))
}

// t.data = Address + msg
func pTransfer(t Transaction) {
	var payee Address
	Decode(0, t.data, &payee)
	assert(t.Cost > 0)
	assert(!payee.Empty())
	adminTransfer(t.User, payee, t.Cost)
}

// syncMoveInfo sync Information of move out
type syncMoveInfo struct {
	User  Address
	Value uint64
}

//t.data = chain
func pMove(t Transaction) {
	var chain uint64
	Decode(0, t.data, &chain)
	assert(chain > 0)
	assert(t.Energy >= 100*gBS.BaseOpsEnergy)
	if gBS.Chain > chain {
		assert(gBS.Chain/2 == chain)
	} else {
		assert(gBS.Chain == chain/2)
		if chain%2 == 0 {
			assert(gBS.LeftChildID > 0)
		} else {
			assert(gBS.RightChildID > 0)
		}
	}

	adminTransfer(t.User, Address{}, t.Cost)
	stru := syncMoveInfo{t.User, t.Cost}
	addSyncInfo(chain, SyncOpsMoveCoin, Encode(0, stru))
}

// MoveCost move app cost to other chain(child chain or parent chain)
func MoveCost(user interface{}, chain, cost uint64) {
	assert(chain > 0)
	if gBS.Chain > chain {
		assert(gBS.Chain/2 == chain)
	} else {
		assert(gBS.Chain == chain/2)
		if chain%2 == 0 {
			assert(gBS.LeftChildID > 0)
		} else {
			assert(gBS.RightChildID > 0)
		}
	}
	gRuntime.ConsumeEnergy(1000 * gBS.BaseOpsEnergy)
	addr := GetAppAccount(user)
	adminTransfer(addr, Address{}, cost)
	stru := syncMoveInfo{addr, cost}
	addSyncInfo(chain, SyncOpsMoveCoin, Encode(0, stru))
}

/*********************** chain ****************************/

type syncNewChain struct {
	Chain      uint64
	Guerdon    uint64
	ParentID   uint64
	Time       uint64
	Producer   Address
	FirstBlock Hash
}

//t.data = newChain
func pNewChain(producer Address, t Transaction) {
	var newChain uint64
	Decode(0, t.data, &newChain)
	assert(newChain/2 == t.Chain)

	guerdon := pDbStat.GetInt([]byte{StatGuerdon})
	si := syncNewChain{}
	si.Chain = newChain
	si.Guerdon = guerdon*9/10 + minGuerdon
	si.ParentID = gBS.ID
	si.Producer = producer
	si.Time = gBS.Time - blockSyncMax + 1
	data, life := pDbStat.Get([]byte{StatFirstBlockKey})
	assert(life+blockSyncMax < maxDbLife)
	Decode(0, data, &si.FirstBlock)
	addSyncInfo(newChain, SyncOpsNewChain, Encode(0, si))

	//left child chain
	if newChain == t.Chain*2 {
		assert(gBS.LeftChildID == 0)
		gBS.LeftChildID = 1
	} else { //right child chain
		assert(gBS.RightChildID == 0)
		assert(gBS.LeftChildID > depositCycle)
		gBS.RightChildID = 1
	}

	if newChain > 3 {
		avgSize := pDbStat.GetInt([]byte{StatAvgBlockSize})
		scale := avgSize * 10 / blockSizeLimit
		assert(scale > 2)
		cost := 10000*guerdon + 10*maxGuerdon
		cost = cost >> scale
		assert(t.Cost >= cost)
	}
	adminTransfer(t.User, gPublicAddr, t.Cost)

	pDbStat.Set([]byte{StatBaseInfo}, Encode(0, gBS), maxDbLife)
	Event(dbTransInfo{}, "new_chain", Encode(0, newChain))
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

func getPlublcAddress(appName Hash) Address {
	out := Address{}
	Decode(0, appName[:], &out)
	out[0] = prefixOfPlublcAddr
	return out
}

type newAppInfo struct {
	LineNum   uint32
	Type      uint16
	Flag      uint8
	DependNum uint8
}

func pNewApp(t Transaction) {
	var (
		appName Hash
		count   uint64 = 1
		deps    []byte
		life    = TimeYear + gBS.Time
		eng     uint64
	)
	assert(t.Cost == 0)
	code := t.data
	ni := newAppInfo{}
	n := Decode(0, code, &ni)
	code = code[n:]
	for i := 0; i < int(ni.DependNum); i++ {
		item := DependItem{}
		n := Decode(0, code, &item)
		code = code[n:]

		itemInfo := GetAppInfo(item.AppName)
		assert(itemInfo != nil)
		if life > itemInfo.Life {
			life = itemInfo.Life
		}
		if ni.Flag&AppFlagPlublc != 0 {
			assert(itemInfo.Flag&AppFlagPlublc != 0)
		}
		count += itemInfo.LineSum
		assert(count > itemInfo.LineSum)
		assert(itemInfo.Flag&AppFlagImport != 0)
		deps = append(deps, item.AppName[:]...)
	}
	assert(life > TimeMonth+gBS.Time)

	appName = GetHash(t.data)
	if ni.Flag&AppFlagPlublc == 0 {
		appName = GetHash(append(appName[:], t.User[:]...))
	}
	assert(GetAppInfo(appName) == nil)

	if ni.Flag&AppFlagImport != 0 {
		eng += uint64(len(code)) * 20
	}
	if ni.Flag&AppFlagRun != 0 {
		eng += uint64(len(code))
	}

	saveInfo := AppInfo{}
	saveInfo.Flag = ni.Flag
	if gBS.ID == 1 {
		// not support call by user
		assert(ni.Flag&AppFlagRun == 0)
		assert(ni.Flag&AppFlagPlublc != 0)
		assert(appName == GetAppName(dbStat{}))
		ni.Flag = ni.Flag | AppFlagRun
		d := Encode(0, ni)
		copy(t.data, d)
		life = maxDbLife + gBS.Time
	}

	if ni.Flag&AppFlagPlublc != 0 {
		saveInfo.Account = getPlublcAddress(appName)
		assert(gPublicAddr != saveInfo.Account)
		if gBS.ID == 1 {
			saveInfo.Account = gPublicAddr
		}
	} else {
		saveInfo.Account = t.User
	}

	gRuntime.NewApp(appName[:], t.data)

	count += uint64(ni.LineNum)
	saveInfo.LineSum = count
	saveInfo.Life = life
	life -= gBS.Time
	eng += count * gBS.BaseOpsEnergy
	assert(t.Energy >= eng)

	pDbApp.Set(appName[:], Encode(0, saveInfo), life)
	if len(deps) > 0 {
		pDbDepend.Set(appName[:], deps, life)
	}
}

func pRunApp(t Transaction) {
	var name Hash
	n := Decode(0, t.data, &name)
	info := GetAppInfo(name)
	assert(info != nil)
	assert(info.Flag&AppFlagRun != 0)
	assert(info.Life >= gBS.Time)

	adminTransfer(t.User, info.Account, t.Cost)
	gRuntime.RunApp(name[:], t.User[:], t.data[n:], t.Energy, t.Cost)
}

// UpdateInfo Information of update app life
type UpdateInfo struct {
	Name Hash
	Life uint64
}

func pUpdateAppLife(t Transaction) {
	var info UpdateInfo
	n := Decode(0, t.data, &info)
	assert(n == len(t.data))
	f := gBS.BaseOpsEnergy * (info.Life + TimeDay) / TimeHour
	assert(f <= t.Cost)
	UpdateAppLife(info.Name, info.Life)
	adminTransfer(t.User, gPublicAddr, t.Cost)
}

// UpdateAppLife update app life
func UpdateAppLife(AppName Hash, life uint64) {
	app := GetAppInfo(AppName)
	assert(app != nil)
	assert(app.Life >= gBS.Time)
	assert(life < 10*TimeYear)
	assert(life > 0)
	app.Life += life
	assert(app.Life > life)
	assert(app.Life < gBS.Time+10*TimeYear)
	deps, _ := pDbDepend.Get(AppName[:])
	pDbApp.Set(AppName[:], Encode(0, app), app.Life-gBS.Time)
	t := gBS.BaseOpsEnergy * (life + TimeDay) / TimeHour
	gRuntime.ConsumeEnergy(t)
	if len(deps) == 0 {
		return
	}
	pDbDepend.Set(AppName[:], deps, app.Life-gBS.Time)
	for len(deps) > 0 {
		item := Hash{}
		n := Decode(0, deps, &item)
		deps = deps[n:]
		itemInfo := GetAppInfo(item)
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

func registerOtherChainMiner(user Address, chain, index, cost uint64) {
	assert(chain/2 == gBS.Chain || gBS.Chain/2 == chain)
	assert(chain > 0)
	if chain > gBS.Chain {
		if chain%2 == 0 {
			assert(gBS.LeftChildID > 0)
		} else {
			assert(gBS.RightChildID > 0)
		}
	}
	adminTransfer(user, Address{}, cost)
	info := syncRegMiner{index, cost, user}
	addSyncInfo(chain, SyncOpsMiner, Encode(0, info))
}

func registerMiner(user Address, index, cost uint64) bool {
	miner := Miner{}
	stream, _ := pDbMining.Get(Encode(0, index))
	if len(stream) > 0 {
		Decode(0, stream, &miner)
		if cost <= miner.Cost[minerNum-1] {
			return false
		}
		for i := 0; i < minerNum; i++ {
			if user == miner.Miner[i] {
				return false
			}
		}
		adminTransfer(Address{}, miner.Miner[minerNum-1], miner.Cost[minerNum-1])
	}
	m := user
	c := cost
	for i := 0; i < minerNum; i++ {
		if c > miner.Cost[i] {
			miner.Miner[i], m = m, miner.Miner[i]
			miner.Cost[i], c = c, miner.Cost[i]
		}
	}
	pDbMining.Set(Encode(0, index), Encode(0, miner), defauldbLife)
	return true
}

func pRegisterMiner(t Transaction) {
	guerdon := pDbStat.GetInt([]byte{StatGuerdon})
	assert(t.Cost >= 3*guerdon)
	info := RegMiner{}
	Decode(0, t.data, &info)

	if info.Chain != 0 && info.Chain != gBS.Chain {
		assert(t.Energy > 1000000)
		if info.Chain > gBS.Chain {
			assert(info.Chain/2 == gBS.Chain)
		} else {
			assert(info.Chain == gBS.Chain/2)
		}
		registerOtherChainMiner(t.User, info.Chain, info.Index, t.Cost)
		return
	}

	assert(info.Index > gBS.ID+20)
	assert(gBS.ID+2*depositCycle > info.Index)

	rst := registerMiner(t.User, info.Index, t.Cost)
	if rst {
		adminTransfer(t.User, Address{}, t.Cost)
	}
}

/*------------------------------api---------------------------------------*/

// AdminInfo register as a admin
type AdminInfo struct {
	App   Hash
	Cost  uint64
	Index uint8
}

// RegisterAdmin app register as a admin
func RegisterAdmin(app interface{}, index uint8, cost uint64) {
	info := AdminInfo{}
	guerdon := pDbStat.GetInt([]byte{StatGuerdon})

	c := (adminLife/TimeDay)*guerdon + maxGuerdon
	assert(cost > c)

	owner := GetAppAccount(app)
	assert(owner[0] == prefixOfPlublcAddr)

	adminTransfer(owner, gPublicAddr, cost)
	info.App = GetAppName(app)
	info.Index = index
	info.Cost = cost
	stream, _ := pDbAdmin.Get([]byte{index})
	if len(stream) > 0 {
		older := AdminInfo{}
		Decode(0, stream, &older)
		if older.App != info.App {
			assert(info.Cost > older.Cost+guerdon)
			pDbAdmin.Set(older.App[:], nil, 0)
		} else {
			info.Cost += older.Cost / 2
		}
	}

	pDbAdmin.Set([]byte{index}, Encode(0, info), adminLife)
	pDbAdmin.SetInt(info.App[:], 1, adminLife)
	Event(dbAdmin{}, "new_admin", info.App[:])
}

// pDisableAdmin disable admin
func pDisableAdmin(user Address, index uint8, cost uint64) {
	assert(cost > minGuerdon)
	stream, life := pDbAdmin.Get([]byte{index})
	if life <= gBS.Time {
		return
	}

	adminTransfer(user, gPublicAddr, cost)
	cost = cost / 10

	info := AdminInfo{}
	Decode(0, stream, &info)

	if cost >= info.Cost {
		pDbAdmin.Set(info.App[:], nil, 0)
		pDbAdmin.Set([]byte{index}, nil, 0)
		Event(dbStat{}, "disable admin", []byte{index}, info.App[:])
		return
	}
	info.Cost = info.Cost - cost
	pDbAdmin.Set([]byte{index}, Encode(0, info), life)
}

// Event send event
func Event(user interface{}, event string, param ...[]byte) {
	gRuntime.Event(user, event, param...)
}

// UpdateConfig admin change the config
func UpdateConfig(user interface{}, ops uint8, newSize uint32) {
	app := GetAppName(user)
	assert(pDbAdmin.GetInt(app[:]) == 1)
	assert(pDbStat.GetInt([]byte{StatChangingConfig}) == 0)
	pDbStat.SetInt([]byte{StatChangingConfig}, 1, TimeMillisecond)
	var min, max uint64
	switch ops {
	case StatBlockSizeLimit:
		min = blockSizeLimit
		max = 1 << 40
	case StatBlockInterval:
		min = minBlockInterval
		max = maxBlockInterval
	case StatHateRatio:
		assert(gBS.Chain == 1)
		min = 50
		max = hateRatioMax
	default:
		assert(ops >= StatMax)
		min = 1 << 10
		max = 1 << 50
	}
	v := uint64(newSize)
	s := pDbStat.GetInt([]byte{ops})
	if s == 0 {
		s = max + min/2
	}
	rangeVal := s/100 + 1
	assert(v <= s+rangeVal)
	assert(v >= s-rangeVal)
	assert(v >= min)
	assert(v <= max)
	pDbStat.SetInt([]byte{ops}, v, maxDbLife)

	owner := GetAppAccount(user)
	guerdon := pDbStat.GetInt([]byte{StatGuerdon})
	adminTransfer(owner, gPublicAddr, maxGuerdon+guerdon*(defauldbLife/TimeDay))
	if ops == StatHateRatio {
		if gBS.LeftChildID > 0 {
			addSyncInfo(2, ops, Encode(0, v))
		}
		if gBS.RightChildID > 0 {
			addSyncInfo(3, ops, Encode(0, v))
		}
	}
	Event(dbStat{}, "ChangeConfig", []byte{ops}, Encode(0, v))
}

// BroadcastInfo broadcase info
type BroadcastInfo struct {
	BlockKey Hash
	App      Hash
	LFlag    byte
	RFlag    byte
	// Other   []byte
}

// Broadcast admin broadcast info to all chain from first chain
func Broadcast(user interface{}, msg []byte) {
	app := GetAppName(user)
	assert(pDbAdmin.GetInt(app[:]) == 1)
	assert(gBS.Chain == 1)
	data, _ := pDbStat.Get([]byte{StatBroadcast})
	assert(len(data) == 0)
	assert(gBS.LeftChildID > 0)
	gRuntime.ConsumeEnergy(maxGuerdon)
	info := BroadcastInfo{}
	info.BlockKey = gBS.Key
	info.App = app
	if gBS.RightChildID == 0 {
		info.RFlag = 1
	}
	d := Encode(0, info)
	d = append(d, msg...)
	addSyncInfo(2, SyncOpsBroadcast, d)
	if gBS.RightChildID > 0 {
		addSyncInfo(3, SyncOpsBroadcast, d)
	}
	pDbStat.Set([]byte{StatBroadcast}, d, logLockTime*2)
}

// DeleteAccount Delete long unused accounts(more than 5 years),call by admin
func DeleteAccount(user interface{}, addr Address) {
	app := GetAppName(user)
	assert(pDbAdmin.GetInt(app[:]) == 1)
	assert(!addr.Empty())
	val, life := getAccount(addr)
	if life+5*TimeYear > maxDbLife {
		return
	}
	l := maxDbLife - life
	t := gBS.BaseOpsEnergy << (l / TimeYear)
	if t < val {
		return
	}
	adminTransfer(addr, gPublicAddr, t)
	Event(dbStat{}, "DeleteAccount", addr[:])
}

// IHateYou It's just a joke.
func IHateYou(user interface{}, peer Address, cost uint64) {
	app := GetAppName(user)
	assert(pDbAdmin.GetInt(app[:]) == 1)
	assert(!peer.Empty())
	ratio := pDbStat.GetInt([]byte{StatHateRatio})
	pc := cost / ratio
	assert(pc > 0)
	TransferAccounts(user, gPublicAddr, cost)
	adminTransfer(peer, gPublicAddr, pc)
	Event(dbAdmin{}, "IHateYou", peer[:], Encode(0, cost))
}

//OtherOps  extensional api
func OtherOps(user interface{}, ops int, energy uint64, data []byte) []byte {
	assert(energy >= gBS.BaseOpsEnergy)
	gRuntime.ConsumeEnergy(energy)
	return gRuntime.OtherOps(user, ops, data)
}

// GetDBData get data by name.
// name list:dbTransInfo,dbCoin,dbAdmin,dbStat,dbApp,dbHate,dbMining,depend,logBlockInfo,logSync
func GetDBData(name string, key []byte) ([]byte, uint64) {
	var db *DB
	switch name {
	case "dbTransInfo":
		db = pDbTransInfo
	case "dbCoin":
		db = pDbCoin
	case "dbAdmin":
		db = pDbAdmin
	case "dbStat":
		db = pDbStat
	case "dbApp":
		db = pDbApp
	case "dbMining":
		db = pDbMining
	case "depend":
		db = pDbDepend
	case "logBlockInfo":
		return pLogBlockInfo.Read(0, key), 0
	case "logSync":
		return pLogSync.Read(0, key), 0
	default:
		return nil, 0
	}
	return db.Get(key)
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

func getSyncKey(typ byte, index uint64) []byte {
	var key = []byte{typ}
	key = append(key, Encode(0, index)...)
	return key
}

func addSyncInfo(chain uint64, ops uint8, data []byte) {
	var info tSyncInfo
	stream, _ := pDbStat.Get([]byte{StatSyncInfo})
	if len(stream) > 0 {
		Decode(0, stream, &info)
	}
	info.addSyncInfo(chain, ops, data)
}

func syncInfos() {
	var info tSyncInfo
	stream, _ := pDbMining.Get(Encode(0, gBS.ID-depositCycle))
	if len(stream) > 0 {
		mi := Miner{}
		Decode(0, stream, &mi)
		for i := 0; i < minerNum; i++ {
			adminTransfer(Address{}, mi.Miner[i], mi.Cost[i]+minGuerdon)
		}
	}

	stream, _ = pDbStat.Get([]byte{StatSyncInfo})
	if len(stream) > 0 {
		Decode(0, stream, &info)
	}

	info.syncFromParent()
	info.syncFromLeftChild()
	info.syncFromRightChild()
	pDbStat.Set([]byte{StatSyncInfo}, Encode(0, info), maxDbLife)
}

func (info *tSyncInfo) addSyncInfo(chain uint64, ops uint8, data []byte) {
	var key []byte
	switch chain {
	case gBS.Chain / 2:
		key = getSyncKey('p', info.ToParentID)
		info.ToParentID++
	case 2 * gBS.Chain:
		key = getSyncKey('l', info.ToLeftChildID)
		info.ToLeftChildID++
	case 2*gBS.Chain + 1:
		key = getSyncKey('r', info.ToRightChildID)
		info.ToRightChildID++
	default:
		assert(false)
	}
	head := syncHead{gBS.ID, ops}
	d := Encode(0, head)
	d = append(d, data...)
	pLogSync.Write(key, d)
	pDbStat.Set([]byte{StatSyncInfo}, Encode(0, *info), maxDbLife)
	Event(logSync{}, "addSyncInfo", []byte{ops}, data)
}

func (info *tSyncInfo) syncFromParent() {
	if gBS.Chain == 1 {
		return
	}
	if gBS.ID == 1 {
		var head syncHead
		var stream []byte
		if gBS.Chain%2 == 0 {
			stream, _ = gRuntime.LogRead(pLogSync.owner, gBS.Chain/2, getSyncKey('l', 0))
		} else {
			stream, _ = gRuntime.LogRead(pLogSync.owner, gBS.Chain/2, getSyncKey('r', 0))
		}
		assert(len(stream) > 0)
		n := Decode(0, stream, &head)
		assert(head.Ops == SyncOpsNewChain)
		info.FromParentID = 1
		info.syncInfo(gBS.Chain/2, head.Ops, stream[n:])
	}
	for {
		var head syncHead
		var stream []byte
		if gBS.Chain%2 == 0 {
			stream = pLogSync.read(gBS.Chain/2, getSyncKey('l', info.FromParentID))
		} else {
			stream = pLogSync.read(gBS.Chain/2, getSyncKey('r', info.FromParentID))
		}
		if len(stream) == 0 {
			break
		}
		n := Decode(0, stream, &head)
		if head.BlockID > gBS.ParentID {
			break
		}
		info.FromParentID++
		info.syncInfo(gBS.Chain/2, head.Ops, stream[n:])
	}
}

func (info *tSyncInfo) syncFromLeftChild() {
	if gBS.LeftChildID <= 1 {
		return
	}
	for {
		var head syncHead
		var stream []byte
		stream = pLogSync.read(gBS.Chain*2, getSyncKey('p', info.FromLeftChildID))
		if len(stream) == 0 {
			break
		}
		n := Decode(0, stream, &head)
		if head.BlockID > gBS.LeftChildID {
			break
		}
		info.FromLeftChildID++
		info.syncInfo(gBS.Chain*2, head.Ops, stream[n:])
	}
}

func (info *tSyncInfo) syncFromRightChild() {
	if gBS.RightChildID <= 1 {
		return
	}
	for {
		var head syncHead
		var stream []byte
		stream = pLogSync.read(gBS.Chain*2+1, getSyncKey('p', info.FromRightChildID))
		if len(stream) == 0 {
			break
		}
		n := Decode(0, stream, &head)
		if head.BlockID > gBS.RightChildID {
			break
		}
		info.FromRightChildID++
		info.syncInfo(gBS.Chain*2+1, head.Ops, stream[n:])
	}
}

func (info *tSyncInfo) syncInfo(from uint64, ops uint8, data []byte) {
	Event(logSync{}, "syncInfo", []byte{ops}, Encode(0, from), data)
	switch ops {
	case SyncOpsMoveCoin:
		var mi syncMoveInfo
		Decode(0, data, &mi)
		adminTransfer(Address{}, mi.User, mi.Value)
	case SyncOpsNewChain:
		var nc syncNewChain
		Decode(0, data, &nc)
		assert(gBS.Chain == nc.Chain)
		assert(gBS.ID == 1)
		assert(gBS.Key == nc.FirstBlock)
		gBS.ParentID = nc.ParentID
		gBS.Time = nc.Time
		pDbStat.Set([]byte{StatBaseInfo}, Encode(0, gBS), maxDbLife)
		pDbStat.SetInt([]byte{StatGuerdon}, nc.Guerdon, maxDbLife)
		registerMiner(nc.Producer, 2, nc.Guerdon)
		Event(dbTransInfo{}, "new_chain_ack", []byte{2})
	case SyncOpsMiner:
		rm := syncRegMiner{}
		Decode(0, data, &rm)

		if rm.Index < gBS.ID+20 || rm.Index > gBS.ID+2*depositCycle {
			adminTransfer(Address{}, rm.User, rm.Cost)
			return
		}
		guerdon := pDbStat.GetInt([]byte{StatGuerdon})
		if rm.Cost < 3*guerdon {
			adminTransfer(Address{}, rm.User, rm.Cost)
			return
		}
		rst := registerMiner(rm.User, rm.Index, rm.Cost)
		if !rst {
			adminTransfer(Address{}, rm.User, rm.Cost)
		}
	case SyncOpsBroadcast:
		br := BroadcastInfo{}
		n := Decode(0, data, &br)
		if gBS.LeftChildID > 0 {
			info.addSyncInfo(gBS.Chain*2, ops, data)
		} else {
			br.LFlag = 1
		}
		if gBS.RightChildID > 0 {
			info.addSyncInfo(gBS.Chain*2+1, ops, data)
		} else {
			br.RFlag = 1
		}
		d := Encode(0, br)
		d = append(d, data[n:]...)
		if br.LFlag > 0 && br.RFlag > 0 {
			info.addSyncInfo(gBS.Chain/2, SyncOpsBroadcastAck, nil)
			Event(logSync{}, "SyncOpsBroadcastAck", d)
		}
		pDbStat.Set([]byte{StatBroadcast}, d, logLockTime*2)
	case SyncOpsBroadcastAck:
		data, life := pDbStat.Get([]byte{StatBroadcast})
		if len(data) == 0 {
			return
		}
		br := BroadcastInfo{}
		n := Decode(0, data, &br)
		if from%2 == 0 {
			br.LFlag = 1
		} else {
			br.RFlag = 1
		}
		d := Encode(0, br)
		d = append(d, data[n:]...)
		pDbStat.Set([]byte{StatBroadcast}, d, life)
		if br.LFlag > 0 && br.RFlag > 0 {
			if gBS.Chain > 1 {
				info.addSyncInfo(gBS.Chain/2, SyncOpsBroadcastAck, nil)
			}
			Event(logSync{}, "SyncOpsBroadcastAck", d)
		}
	case SyncOpsHateRatio:
		var ratio uint64
		Decode(0, data, &ratio)
		pDbStat.SetInt([]byte{StatHateRatio}, ratio, maxDbLife)
		if gBS.LeftChildID > 0 {
			info.addSyncInfo(2*gBS.Chain, SyncOpsHateRatio, data)
		}
		if gBS.RightChildID > 0 {
			info.addSyncInfo(2*gBS.Chain+1, SyncOpsHateRatio, data)
		}
	}
}
