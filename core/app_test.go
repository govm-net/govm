package zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f

import (
	"log"
	"testing"

	"github.com/govm-net/govm/runtime"
)

func TestCreateAppFromSourceCode(t *testing.T) {
	log.Println("testing start", t.Name())
	// go test -v -run TestCreateAppFromSourceCode
	code, ln := CreateAppFromSourceCode("./test_data/app1/app1.go", AppFlagImport|AppFlagPlublc)
	if code == nil {
		t.Error("fail to create app")
	}
	if ln != 3 {
		t.Errorf("error code line number,hope 3, get %d", ln)
	}
	// log.Println("code:", string(code))
	log.Printf("app:%x\n", runtime.GetHash(code))
}
