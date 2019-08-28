package runtime

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/lengzhao/govm/conf"
)

func TestNewApp(t *testing.T) {
	c := conf.GetConf()
	realPath := GetFullPathOfApp(1, c.CorePackName)
	defer os.RemoveAll(realPath)
	code, _ := ioutil.ReadFile("./code_for_test.txt")
	nInfo := TAppNewInfo{}
	nInfo.LineNum = 3
	nInfo.DependNum = 0
	nInfo.Flag = AppFlagRun | AppFlagImport
	head := Encode(nInfo.TAppNewHead)
	code = append(head, code...)
	NewApp(1, c.CorePackName, code)
	RunApp(nil, 1, c.CorePackName, []byte("user1"), []byte("data1"), 1<<50, 1)
}
