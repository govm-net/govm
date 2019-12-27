package runtime

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestNewApp(t *testing.T) {
	appName := []byte("app001")
	realPath := GetFullPathOfApp(1, appName)
	defer os.RemoveAll(realPath)
	code, _ := ioutil.ReadFile("./code_for_test.txt")
	nInfo := TAppNewInfo{}
	nInfo.LineNum = 3
	nInfo.DependNum = 0
	nInfo.Flag = AppFlagRun | AppFlagImport
	head := Encode(nInfo.TAppNewHead)
	code = append(head, code...)
	NewApp(1, appName, code)
	RunApp(nil, 1, "", appName, []byte("user1"), []byte("data1"), 1<<50, 1)
}
