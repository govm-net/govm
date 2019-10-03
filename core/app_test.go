package ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c

import (
	"log"
	"os"
	"testing"

	"github.com/lengzhao/govm/runtime"
)

func TestCreateAppFromSourceCode(t *testing.T) {
	// go test -v -run TestCreateAppFromSourceCode
	var code []byte
	var ln uint64
	if _, err := os.Stat("./core_linux.go"); os.IsNotExist(err) {
		code, ln = CreateAppFromSourceCode("./core.go", AppFlagImport|AppFlagPlublc)
	} else {
		code, ln = CreateAppFromSourceCode("./core_linux.go", AppFlagImport|AppFlagPlublc)
	}
	if code == nil {
		t.Error("fail to create app")
	}
	log.Printf("app: %x %d\n", runtime.GetHash(code), ln)
}

func TestCreateAppFromSourceCode2(t *testing.T) {
	// go test -v -run TestCreateAppFromSourceCode
	code, _ := CreateAppFromSourceCode("./test_data/app1/app1.go", AppFlagImport|AppFlagPlublc)
	if code == nil {
		t.Error("fail to create app")
	}
	// log.Println("code:", string(code))
	log.Printf("app:%x\n", runtime.GetHash(code))
}
