package runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/lengzhao/govm/database"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"sync"
	"time"
)

// EventFilter event filter, show or drop app event
type EventFilter struct {
	sw map[string]string
	mu sync.Mutex
}

// const module = "govm.net/lengzhao/govm"
const module = "github.com/lengzhao/govm"

var projectRoot string
var packPath string
var filter EventFilter

func init() {
	projectRoot = path.Join(os.Getenv("GOPATH"), "src", module)
	// projectRoot = path.Join("vendor", module, "apps")
	packPath = module
	loadEventFilter()
}

// GetHash 计算hash值
func GetHash(in []byte) []byte {
	var h = sha256.New()
	h.Write(in)
	return h.Sum(nil)
}

// Encode 将interface{}转成字符流，不支持可变长度类型
func Encode(in interface{}) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, in)
	if err != nil {
		log.Println("fail to encode interface:", reflect.TypeOf(in).String(), in)
		return nil
	}
	return buf.Bytes()
}

// Decode 将字符流填充到指定结构体
func Decode(in []byte, out interface{}) int {
	buf := bytes.NewReader(in)
	err := binary.Read(buf, binary.BigEndian, out)
	if err != nil {
		log.Println("fail to decode interface:", in[:20])
		panic(err)
		//return 0
	}
	return len(in) - buf.Len()
}

// GetPackPath get the package path on golang packages
func GetPackPath(chain uint64, name []byte) string {
	nameStr := hex.EncodeToString(name)
	nameStr = "a" + nameStr
	return path.Join(packPath, fmt.Sprintf("chain%d", chain), nameStr)
}

// GetFullPathOfApp get the full path of app
func GetFullPathOfApp(chain uint64, name []byte) string {
	nameStr := hex.EncodeToString(name)
	nameStr = "a" + nameStr
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

// RunApp run app
func RunApp(flag []byte, chain uint64, appName, user, data []byte, energy, cost uint64) {
	args := TRunParam{chain, flag, user, data, cost, energy, ""}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(args)

	err := database.Set(chain, []byte("app_run"), []byte("key"), buf.Bytes())
	if err != nil {
		log.Println("[db]fail to write data.", err)
		panic("retry")
	}
	appPath := GetFullPathOfApp(chain, appName)
	appPath = path.Join(appPath, "app.exe")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, appPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Println("fail to exec app.", err)
		panic("retry")
	}

	d := database.Get(chain, []byte("app_run"), []byte("key"))
	if len(d) == 0 {
		log.Println("[db]fail to get data.")
		panic("retry")
	}

	rst := bytes.NewBuffer(d)
	dec := gob.NewDecoder(rst)
	err = dec.Decode(&args)
	if err != nil {
		log.Println("decode error:", err)
		panic("retry")
	}

	if args.ErrorInfo != "ok" {
		panic(args.ErrorInfo)
	}
}

func createDir(dirName string) {
	_, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		createDir(path.Dir(dirName))
		os.Mkdir(dirName, 666)
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