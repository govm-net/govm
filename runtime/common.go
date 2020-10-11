package runtime

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"

	"github.com/govm-net/govm/wallet"
)

// EventFilter event filter, show or drop app event
type EventFilter struct {
	sw map[string]string
	mu sync.Mutex
}

const (
	tbOfRunParam  = "app_run"
	tbOfRunResult = "app_result"
)

// Module go.mod
var (
	Module      = "github.com/govm-net/govm"
	projectRoot = "app"
	filter      EventFilter
	AppPath     = "."
	RunDir      = "."
	BuildDir    = "."
	NotRebuild  bool
)

func init() {
	loadEventFilter()
}

// GetHash 计算hash值
func GetHash(in []byte) []byte {
	return wallet.GetHash(in)
}

// Encode 将interface{}转成字符流，不支持可变长度类型
func Encode(in interface{}) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, in)
	if err != nil {
		log.Panic("fail to encode interface:", reflect.TypeOf(in).String(), in)
	}
	return buf.Bytes()
}

// Decode 将字符流填充到指定结构体
func Decode(in []byte, out interface{}) int {
	buf := bytes.NewReader(in)
	err := binary.Read(buf, binary.BigEndian, out)
	if err != nil {
		log.Panicf("fail to decode interface,data len:%d,data:%x,type:%T",
			len(in), in[:20], out)
	}
	return len(in) - buf.Len()
}

// JSONEncode encode json
func JSONEncode(in interface{}) []byte {
	d, err := json.Marshal(in)
	if err != nil {
		log.Panic("fail to encode interface:", reflect.TypeOf(in).String(), err)
	}
	return d
}

// JSONDecode decode json
func JSONDecode(in []byte, out interface{}) int {
	err := json.Unmarshal(in, out)
	if err != nil {
		log.Panic("fail to decode interface:", reflect.TypeOf(out).String())
	}
	return 0
}

// GobEncode gob encode
func GobEncode(in interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(in)
	if err != nil {
		log.Panic("fail to encode interface:", reflect.TypeOf(in).String(), err)
	}
	return buf.Bytes()
}

// GobDecode gob decode
func GobDecode(in []byte, out interface{}) int {
	buf := bytes.NewReader(in)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(out)
	if err != nil {
		log.Panic("fail to decode interface:", reflect.TypeOf(out).String())
	}
	return len(in) - buf.Len()
}

// GetPackPath get the package path on golang packages
func GetPackPath(chain uint64, name []byte) string {
	nameStr := hex.EncodeToString(name)
	nameStr = "z" + nameStr
	return path.Join(Module, projectRoot, fmt.Sprintf("chain%d", chain), nameStr)
}

// GetFullPathOfApp get the full path of app
func GetFullPathOfApp(chain uint64, name []byte) string {
	nameStr := hex.EncodeToString(name)
	nameStr = "z" + nameStr
	nameStr = path.Join(projectRoot, fmt.Sprintf("chain%d", chain), nameStr)
	return nameStr
}

// TRunParam Run接口的入参
type TRunParam struct {
	Chain     uint64
	Flag      []byte
	User      []byte
	Data      []byte
	Cost      uint64
	Energy    uint64
	ErrorInfo string
}

func createDir(dirName string) {
	_, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		createDir(path.Dir(dirName))
		os.Mkdir(dirName, 0777)
	}
}

func loadEventFilter() {
	data, err := ioutil.ReadFile("./conf/event_filter.json")
	if err != nil {
		log.Println("fail to read file,event_filter.json")
		return
	}
	filter.mu.Lock()
	defer filter.mu.Unlock()
	err = json.Unmarshal(data, &filter.sw)
	if err != nil {
		log.Println("fail to Unmarshal configure,event_filter.json")
		return
	}
}

// GetAppName 用app的私有结构体，获取app的Hash名字
func GetAppName(owner interface{}) []byte {
	structName := GetStructName(owner)
	structName = structName[1:]
	typeSplic := strings.Split(string(structName), ".")
	appName, _ := hex.DecodeString(typeSplic[0])
	//log.Printf("GetAppName:%x", appName)
	return appName
}
