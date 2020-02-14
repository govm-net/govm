package runtime

import (
	"encoding/json"
	"fmt"
	"github.com/lengzhao/database/client"
	"github.com/lengzhao/govm/conf"
	"github.com/lengzhao/govm/counter"
	db "github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/wallet"
	"log"
	"reflect"
	"strings"
)

// TRuntime 执行机的结构体定义
type TRuntime struct {
	Chain    uint64
	Flag     []byte
	testMode bool
	db       *client.Client
}

const (
	startOfDB  = 'd'
	startOfLog = 'l'
)

func assert(cond bool) {
	if !cond {
		panic("error")
	}
}

// NewRuntime input address of database
func NewRuntime(addrType, address string) *TRuntime {
	out := new(TRuntime)
	if address != "" {
		out.db = client.New(addrType, address, 1)
	} else {
		out.db = db.GetClient()
	}
	return out
}

// SetInfo 设置参数
func (r *TRuntime) SetInfo(chain uint64, flag []byte) {
	r.Flag = flag
	r.Chain = chain
}

// SetTestMode set test mode,it will not write data to database
func (r *TRuntime) SetTestMode() {
	c := conf.GetConf()
	r.testMode = true
	r.db = client.New(c.DbAddrType, c.AddrForTest, 1)
}

// GetHash 计算hash值
func (r *TRuntime) GetHash(in []byte) []byte {
	return GetHash(in)
}

// encoding type
const (
	EncBinary = uint8(iota)
	EncJSON
	EncGob
)

// Encode 将interface{}转成字符流，不支持可变长度类型
func (r *TRuntime) Encode(typ uint8, in interface{}) []byte {
	var out []byte
	switch typ {
	case EncBinary:
		out = Encode(in)
	case EncJSON:
		out = JSONEncode(in)
	case EncGob:
		out = GobEncode(in)
	default:
		panic("not support encode type")
	}
	return out
}

// Decode 将字符流填充到指定结构体
func (r *TRuntime) Decode(typ uint8, in []byte, out interface{}) int {
	var rst int
	switch typ {
	case EncBinary:
		rst = Decode(in, out)
	case EncJSON:
		rst = JSONDecode(in, out)
	case EncGob:
		rst = GobDecode(in, out)
	default:
		panic("not support decode type")
	}
	return rst
}

// JSONEncode 将结构体转成json格式的字符串
func (r *TRuntime) JSONEncode(in interface{}) []byte {
	out, err := json.Marshal(in)
	if err != nil {
		panic(in)
	}
	return out
}

// JSONDecode 将json格式的字符串转成结构体
func (r *TRuntime) JSONDecode(in []byte, out interface{}) {
	err := json.Unmarshal(in, out)
	if err != nil {
		panic(in)
	}
}

// AdminDbSet write data to the chain
func AdminDbSet(owner interface{}, chain uint64, key, value []byte, life uint64) error {
	assert(chain > 0)
	tbName := GetStructName(owner)
	if len(value) == 0 || life == 0 {
		return db.GetClient().Set(chain, tbName, key, nil)
	}
	value = append(value, Encode(life)...)
	err := db.GetClient().Set(chain, tbName, key, value)
	if err != nil {
		return err
	}
	return nil
}

// DbGet get data form db
func DbGet(owner interface{}, chain uint64, key []byte) ([]byte, uint64) {
	assert(chain > 0)
	tbName := GetStructName(owner)
	data := db.GetClient().Get(chain, tbName, key)
	if len(data) == 0 {
		return nil, 0
	}
	n := len(data)
	lifeBytes := data[n-8:]
	var life uint64
	Decode(lifeBytes, &life)
	return data[:n-8], life
}

// DbExist return true if exist
func DbExist(owner interface{}, chain uint64, key []byte) bool {
	assert(chain > 0)
	tbName := GetStructName(owner)
	return db.GetClient().Exist(chain, tbName, key)
}

// DbSet 数据库保存数据
func (r *TRuntime) DbSet(owner interface{}, key, value []byte, life uint64) {
	assert(r.Chain > 0)
	assert(r.Flag != nil)
	tbName := GetStructName(owner)
	err := r.db.SetWithFlag(r.Chain, r.Flag, tbName, key, value)
	if err != nil {
		panic(err)
	}
}

// DbGet 数据库读取数据
func (r *TRuntime) DbGet(owner interface{}, key []byte) ([]byte, uint64) {
	assert(r.Chain > 0)
	var data []byte
	tbName := GetStructName(owner)
	data = r.db.Get(r.Chain, tbName, key)

	if len(data) == 0 {
		return nil, 0
	}
	n := len(data)
	lifeBytes := data[n-8:]
	var life uint64
	r.Decode(0, lifeBytes, &life)
	return data[:n-8], life
}

// DbGetLife get life of the db data
func (r *TRuntime) DbGetLife(owner interface{}, key []byte) uint64 {
	_, life := r.DbGet(owner, key)
	return life
}

// LogWrite log write
func (r *TRuntime) LogWrite(owner interface{}, key, value []byte, life uint64) {
	assert(r.Chain > 0)
	assert(r.Flag != nil)
	tbName := getNameOfLogDB(owner)
	value = append(value, r.Encode(0, life)...)
	err := r.db.SetWithFlag(r.Chain, r.Flag, tbName, key, value)
	if err != nil {
		panic(err)
	}
	// log.Printf("write log data.chain:%d,tb:%s,key:%x\n", r.Chain, tbName, key)
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

// LogRead The reading interface of the log
func (r *TRuntime) LogRead(owner interface{}, chain uint64, key []byte) ([]byte, uint64) {
	assert(r.Chain > 0)
	var data []byte
	tbName := getNameOfLogDB(owner)
	if chain == 0 {
		chain = r.Chain
	}
	if chain != r.Chain {
		assert(r.Chain < 8*chain)
		assert(8*r.Chain > chain)
		dist := getLogicDist(r.Chain, chain)
		if dist > 4 {
			assert(r.Chain+3 > chain)
			assert(r.Chain < chain+3)
		}
	}
	data = r.db.Get(chain, tbName, key)

	if len(data) == 0 {
		// log.Printf("fail to read log data.self:%d,chain:%d,tb:%s,key:%x\n", r.Chain, chain, tbName, key)
		return nil, 0
	}
	n := len(data)
	// log.Printf("read log data.self:%d,chain:%d,tb:%s,key:%x,len:%d\n", r.Chain, chain, tbName, key, n)
	lifeBytes := data[n-8:]
	var life uint64
	r.Decode(0, lifeBytes, &life)
	return data[:n-8], life
}

// LogReadLife get life of the log data
func (r *TRuntime) LogReadLife(owner interface{}, key []byte) uint64 {
	_, life := r.LogRead(owner, r.Chain, key)
	return life
}

// LogRead get data form log db
func LogRead(owner interface{}, chain uint64, key []byte) ([]byte, uint64) {
	assert(chain > 0)
	tbName := getNameOfLogDB(owner)
	data := db.GetClient().Get(chain, tbName, key)
	if len(data) == 0 {
		return nil, 0
	}
	n := len(data)
	lifeBytes := data[n-8:]
	var life uint64
	Decode(lifeBytes, &life)
	return data[:n-8], life
}

// GetNextKey get next key
func GetNextKey(chain uint64, isDb bool, appName, structName string, preKey []byte) []byte {
	var tbName string
	if isDb {
		tbName = string(startOfDB)
	} else {
		tbName = string(startOfLog)
	}
	tbName += appName + "." + structName
	// log.Printf("GetNextKey,tbName:%s\n", string(tbName))
	return db.GetClient().GetNextKey(chain, []byte(tbName), preKey)
}

// GetValue get value of key
func GetValue(chain uint64, isDb bool, appName, structName string, key []byte) ([]byte, uint64) {
	var tbName string
	if isDb {
		tbName = string(startOfDB)
	} else {
		tbName = string(startOfLog)
	}
	tbName += appName + "." + structName
	// log.Printf("GetNextKey,tbName:%s\n", string(tbName))
	data := db.GetClient().Get(chain, []byte(tbName), key)
	if len(data) == 0 {
		return nil, 0
	}
	n := len(data)
	lifeBytes := data[n-8:]
	var life uint64
	Decode(lifeBytes, &life)
	return data[:n-8], life
}

// Recover 校验签名信息
func (r *TRuntime) Recover(address, sign, msg []byte) bool {
	return wallet.Recover(address, sign, msg)
}

// GetStructName 通过包的私有对象，获取私有对象名字
func GetStructName(owner interface{}) []byte {
	kind := reflect.ValueOf(owner).Kind()
	if kind != reflect.Struct {
		panic(owner)
	}
	typ := reflect.TypeOf(owner).String()
	typeSplic := strings.Split(typ, ".")
	if len(typeSplic) != 2 {
		panic(typ)
	}
	startChar := typeSplic[1][0]
	if startChar < 'a' || startChar > 'z' {
		panic(typ)
	}
	out := []byte(typ)
	out[0] = startOfDB
	return out
}

// getNameOfLogDB 通过包的私有对象，获取日志对象的名字
func getNameOfLogDB(owner interface{}) []byte {
	kind := reflect.ValueOf(owner).Kind()
	if kind != reflect.Struct {
		panic(owner)
	}
	typ := reflect.TypeOf(owner).String()
	typeSplic := strings.Split(typ, ".")
	if len(typeSplic) != 2 {
		panic(typ)
	}
	startChar := typeSplic[1][0]
	if startChar < 'a' || startChar > 'z' {
		panic(typ)
	}
	out := []byte(typ)
	out[0] = startOfLog
	return out
}

// GetAppName 用app的私有结构体，获取app的Hash名字
func (r *TRuntime) GetAppName(owner interface{}) []byte {
	return GetAppName(owner)
}

// NewApp 新建app，返回可运行的代码行数
func (r *TRuntime) NewApp(name []byte, code []byte) {
	NewApp(r.Chain, name, code)
}

// RunApp 执行app，返回执行的指令数量
func (r *TRuntime) RunApp(name, user, data []byte, energy, cost uint64) {
	// log.Println("run app:", "a"+hex.EncodeToString(name))
	if r.testMode {
		RunApp(r.db, r.Flag, r.Chain, "test", name, user, data, energy, cost)
	} else {
		RunApp(r.db, r.Flag, r.Chain, "", name, user, data, energy, cost)
	}
}

// Event event
func (r *TRuntime) Event(user interface{}, event string, param ...[]byte) {
	pn := fmt.Sprintf("%T.%s", user, event)
	filter.mu.Lock()
	defer filter.mu.Unlock()
	if filter.sw == nil {
		log.Printf("event:%d,%s,%x\n", r.Chain, pn, param)
		return
	}
	alias := filter.sw[pn]
	if alias != "" {
		log.Printf("event:%d,%s,%x\n", r.Chain, alias, param)
	}
}

// ConsumeEnergy consume energy
func (r *TRuntime) ConsumeEnergy(energy uint64) {
	counter.ConsumeEnergy(energy)
}

// OtherOps extesion api
func (r *TRuntime) OtherOps(user interface{}, ops int, data []byte) []byte {
	panic("not support")
}
