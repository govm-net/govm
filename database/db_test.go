package database

import (
	"bytes"
	"os"
	"testing"
)

func TestOpenMgr(t *testing.T) {
	defer os.RemoveAll("./TestOpenMgr")
	m := OpenMgr("./TestOpenMgr")
	if m == nil {
		t.Error("hope open mgr")
		return
	}
	m.Close()

	m = OpenMgr("./TestOpenMgr")
	if m == nil {
		t.Error("hope open mgr")
		return
	}
	m.Close()
}

func TestOpenFlag(t *testing.T) {
	defer os.RemoveAll("./TestOpenFlag")
	m := OpenMgr("./TestOpenFlag")
	if m == nil {
		t.Error("hope open mgr")
		return
	}
	defer m.Close()
	// 打开标签
	err := m.OpenFlag([]byte{1, 2, 3})
	if err != nil {
		t.Error("fail to open flag")
		return
	}

	// 已经打开标签，再次打开会失败
	err = m.OpenFlag([]byte{1, 2, 3})
	if err == nil {
		t.Error("open same flag twice,return nil")
		return
	}
	err = m.OpenFlag([]byte{2, 3, 4})
	if err == nil {
		t.Error("open diff flag twice,return nil")
		return
	}

	//尝试关闭不存在的标签，会返回错误
	err = m.Commit([]byte{2, 3, 4})
	if err == nil {
		t.Error("close difference flag,return nil")
		return
	}

	//正常关闭标签
	err = m.Commit([]byte{1, 2, 3})
	if err != nil {
		t.Error("close flag,return error:", err)
		return
	}
}

func TestTDbMgr_SetWithFlag(t *testing.T) {
	var flag = []byte{1, 2, 3}
	var tbName = []byte("tabel")
	var key = []byte("key")
	var value1 = []byte("value1")
	var value2 = []byte("value2")
	defer os.RemoveAll("./TestTDbMgr_SetWithFlag")
	m := OpenMgr("./TestTDbMgr_SetWithFlag")
	if m == nil {
		t.Error("hope open mgr")
		return
	}
	defer m.Close()
	// 打开标签
	err := m.OpenFlag(flag)
	if err != nil {
		t.Fatal("fail to open flag")
		return
	}

	err = m.SetWithFlag(flag, tbName, key, value1)
	if err != nil {
		t.Fatal("fail to put value")
	}

	v := m.Get(tbName, key)
	if bytes.Compare(v, value1) != 0 {
		t.Fatal("wrong value of get")
	}
	err = m.SetWithFlag(flag, tbName, key, value2)
	if err != nil {
		t.Fatal("fail to put value")
	}
	v = m.Get(tbName, key)
	if bytes.Compare(v, value2) != 0 {
		t.Fatal("wrong value of get.2")
	}
	err = m.Commit(flag)
	if err != nil {
		t.Fatal("fail to commit")
	}

	err = m.Set(tbName, key, value1)
	if err != nil {
		t.Fatal("fail to put value")
	}
	v = m.Get(tbName, key)
	if bytes.Compare(v, value1) != 0 {
		t.Fatal("wrong value of get.2")
	}

}

func TestTDbMgr_GetLastFlag(t *testing.T) {
	var flag1 = []byte{1, 2, 3}
	var flag2 = []byte{2, 3, 4}
	defer os.RemoveAll("./TestTDbMgr_Rollback")
	m := OpenMgr("./TestTDbMgr_Rollback")
	if m == nil {
		t.Error("hope open mgr")
		return
	}
	defer m.Close()
	// 打开标签
	err := m.OpenFlag(flag1)
	if err != nil {
		t.Fatal("fail to open flag")
		return
	}

	err = m.Commit(flag1)
	if err != nil {
		t.Fatal("fail to commit")
	}

	flag := m.GetLastFlag()
	if bytes.Compare(flag, flag1) != 0 {
		t.Fatal("fail to GetLastFlag")
	}

	err = m.OpenFlag(flag2)
	if err != nil {
		t.Fatal("fail to open flag")
		return
	}

	err = m.Commit(flag2)
	if err != nil {
		t.Fatal("fail to commit")
	}

	flag = m.GetLastFlag()
	if bytes.Compare(flag, flag2) != 0 {
		t.Fatal("fail to GetLastFlag")
	}
}

func TestTDbMgr_Rollback(t *testing.T) {
	var flag1 = []byte{1, 2, 3}
	var flag2 = []byte{2, 3, 4}
	var tbName = []byte("tabel")
	var key = []byte("key")
	var value1 = []byte("value1")
	var value2 = []byte("value2")
	var value3 = []byte("value3")
	defer os.RemoveAll("./TestTDbMgr_Rollback")
	m := OpenMgr("./TestTDbMgr_Rollback")
	if m == nil {
		t.Error("hope open mgr")
		return
	}
	defer m.Close()
	// 打开标签
	err := m.OpenFlag(flag1)
	if err != nil {
		t.Fatal("fail to open flag")
		return
	}

	err = m.SetWithFlag(flag1, tbName, key, value1)
	if err != nil {
		t.Fatal("fail to put value")
	}

	v := m.Get(tbName, key)
	if bytes.Compare(v, value1) != 0 {
		t.Fatal("wrong value of get")
	}
	err = m.Commit(flag1)
	if err != nil {
		t.Fatal("fail to commit")
	}

	//-----------------
	err = m.OpenFlag(flag2)
	if err != nil {
		t.Fatal("fail to open flag")
		return
	}

	err = m.SetWithFlag(flag2, tbName, key, value2)
	if err != nil {
		t.Fatal("fail to put value")
	}

	v = m.Get(tbName, key)
	if bytes.Compare(v, value2) != 0 {
		t.Fatal("wrong value of get")
	}
	err = m.Commit(flag2)
	if err != nil {
		t.Fatal("fail to commit")
	}

	err = m.Rollback(flag1)
	if err == nil {
		t.Fatal("fail to rollback,not last falg,hope error")
	}

	lastFlag := m.GetLastFlag()
	if bytes.Compare(lastFlag, flag2) != 0 {
		t.Fatal("error last flag")
	}

	err = m.Rollback(flag2)
	if err != nil {
		t.Fatal("fail to rollback", err)
	}

	lastFlag = m.GetLastFlag()
	if bytes.Compare(lastFlag, flag1) != 0 {
		t.Fatal("error last flag")
	}

	v = m.Get(tbName, key)
	if bytes.Compare(v, value1) != 0 {
		t.Fatal("wrong value of get")
	}

	//-----------------

	err = m.OpenFlag(flag2)
	if err != nil {
		t.Fatal("fail to open flag")
		return
	}

	err = m.SetWithFlag(flag2, tbName, key, value3)
	if err != nil {
		t.Fatal("fail to put value")
	}

	err = m.Commit(flag2)
	if err != nil {
		t.Fatal("fail to commit")
	}

	lastFlag = m.GetLastFlag()
	if bytes.Compare(lastFlag, flag2) != 0 {
		t.Fatal("error last flag")
	}

	v = m.Get(tbName, key)
	if bytes.Compare(v, value3) != 0 {
		t.Fatal("wrong value of get")
	}

	err = m.Rollback(flag2)
	if err != nil {
		t.Fatal("fail to rollback", err)
	}

	lastFlag = m.GetLastFlag()
	if bytes.Compare(lastFlag, flag1) != 0 {
		t.Fatal("error last flag")
	}

	v = m.Get(tbName, key)
	if bytes.Compare(v, value1) != 0 {
		t.Fatal("wrong value of get")
	}

	err = m.Rollback(flag1)
	if err != nil {
		t.Fatal("fail to rollback", err)
	}

	lastFlag = m.GetLastFlag()
	if lastFlag == nil {
		t.Fatal("error last flag")
	}

	v = m.Get(tbName, key)
	if v != nil {
		t.Fatalf("wrong value of get:%x\n", v)
	}
}
