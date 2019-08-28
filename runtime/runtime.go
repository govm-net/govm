package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/lengzhao/govm/counter"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/wallet"
	"log"
	"reflect"
	"strings"
)

// TRuntime 执行机的结构体定义
type TRuntime struct {
	Chain uint64
	Flag  []byte
}

func assert(cond bool) {
	if !cond {
		panic("error")
	}
}

// SetInfo 设置参数
func (r *TRuntime) SetInfo(chain uint64, flag []byte) {
	r.Flag = flag
	r.Chain = chain
}

// GetHash 计算hash值
func (r *TRuntime) GetHash(in []byte) []byte {
	var h = sha256.New()
	h.Write(in)
	return h.Sum(nil)
}

// Encode 将interface{}转成字符流，不支持可变长度类型
func (r *TRuntime) Encode(typ uint8, in interface{}) []byte {
	return Encode(in)
}

// Decode 将字符流填充到指定结构体
func (r *TRuntime) Decode(typ uint8, in []byte, out interface{}) int {
	return Decode(in, out)
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
	value = append(value, Encode(life)...)
	err := database.Set(chain, tbName, key, value)
	if err != nil {
		return err
	}
	return nil
}

// DbGet get data form db
func DbGet(owner interface{}, chain uint64, key []byte) ([]byte, uint64) {
	assert(chain > 0)
	tbName := GetStructName(owner)
	data := database.Get(chain, tbName, key)
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
	return database.Exist(chain, tbName, key)
}

// DbSet 数据库保存数据
func (r *TRuntime) DbSet(owner interface{}, key, value []byte, life uint64) {
	assert(r.Chain > 0)
	assert(r.Flag != nil)
	tbName := GetStructName(owner)
	value = append(value, r.Encode(0, life)...)
	err := database.SetWithFlag(r.Chain, r.Flag, tbName, key, value)
	if err != nil {
		panic(err)
	}
}

// DbGet 数据库读取数据
func (r *TRuntime) DbGet(owner interface{}, key []byte) ([]byte, uint64) {
	assert(r.Chain > 0)
	tbName := GetStructName(owner)
	data := database.Get(r.Chain, tbName, key)
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
	err := database.SetWithFlag(r.Chain, r.Flag, tbName, key, value)
	if err != nil {
		panic(err)
	}
	// log.Printf("write log data.chain:%d,tb:%s,key:%x\n", r.Chain, tbName, key)
}

// LogRead The reading interface of the log
func (r *TRuntime) LogRead(owner interface{}, chain uint64, key []byte) ([]byte, uint64) {
	assert(r.Chain > 0)
	tbName := getNameOfLogDB(owner)
	if chain == 0 {
		chain = r.Chain
	}
	data := database.Get(chain, tbName, key)
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
	data := database.Get(chain, tbName, key)
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
	out[0] = 'd'
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
	out[0] = 'l'
	return out
}

// GetAppName 用app的私有结构体，获取app的Hash名字
func (r *TRuntime) GetAppName(owner interface{}) []byte {
	structName := GetStructName(owner)
	structName = structName[1:]
	typeSplic := strings.Split(string(structName), ".")
	appName, _ := hex.DecodeString(typeSplic[0])
	//log.Printf("GetAppName:%x", appName)
	return appName
}

// NewApp 新建app，返回可运行的代码行数
func (r *TRuntime) NewApp(name []byte, code []byte) {
	NewApp(r.Chain, name, code)
}

// RunApp 执行app，返回执行的指令数量
func (r *TRuntime) RunApp(name, user, data []byte, energy, cost uint64) {
	// log.Println("run app:", "a"+hex.EncodeToString(name))
	RunApp(r.Flag, r.Chain, name, user, data, energy, cost)
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