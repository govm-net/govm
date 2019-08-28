package database

import (
	"fmt"
	"os"
	"path"
	"sync"
)

// TDb rpc接口
type TDb struct {
	mu   sync.Mutex
	mgrs map[uint64]*TDbMgr
}

// SetArgs Set接口的入参
type SetArgs struct {
	Chain  uint64
	TbName []byte
	Key    []byte
	Value  []byte
}

// SetWithFlagArgs Set接口的入参
type SetWithFlagArgs struct {
	Chain  uint64
	Flag   []byte
	TbName []byte
	Key    []byte
	Value  []byte
}

// GetArgs Get接口的入参
type GetArgs struct {
	Chain  uint64
	TbName []byte
	Key    []byte
}

// FlagArgs flag操作的参数
type FlagArgs struct {
	Chain uint64
	Flag  []byte
}

const (
	gDbRoot      = "db_dir"
	gDbDirPrefic = "db_"
)

// Init init TDb
func (t *TDb) Init() {
	t.mgrs = make(map[uint64]*TDbMgr)
	os.Mkdir(gDbRoot, 0600)
}

// Close close
func (t *TDb) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, mgr := range t.mgrs {
		mgr.Close()
	}
}

func (t *TDb) getMgr(id uint64) *TDbMgr {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := t.mgrs[id]
	if out == nil {
		dir := fmt.Sprintf("%s%d", gDbDirPrefic, id)
		dir = path.Join(gDbRoot, dir)
		out = OpenMgr(dir)
		t.mgrs[id] = out
	}
	return out
}

// Set Set
func (t *TDb) Set(args *SetArgs, reply *bool) error {
	dbm := t.getMgr(args.Chain)
	return dbm.Set(args.TbName, args.Key, args.Value)
}

// SetWithFlag SetWithFlag
func (t *TDb) SetWithFlag(args *SetWithFlagArgs, reply *bool) error {
	dbm := t.getMgr(args.Chain)
	return dbm.SetWithFlag(args.Flag, args.TbName, args.Key, args.Value)
}

// Get Get
func (t *TDb) Get(args *GetArgs, reply *([]byte)) error {
	dbm := t.getMgr(args.Chain)
	*reply = dbm.Get(args.TbName, args.Key)
	return nil
}

// Exist Exist
func (t *TDb) Exist(args *GetArgs, reply *bool) error {
	dbm := t.getMgr(args.Chain)
	*reply = dbm.Exist(args.TbName, args.Key)
	return nil
}

// OpenFlag OpenFlag
func (t *TDb) OpenFlag(args *FlagArgs, reply *bool) error {
	dbm := t.getMgr(args.Chain)
	return dbm.OpenFlag(args.Flag)
}

// CommitFlag CommitFlag
func (t *TDb) CommitFlag(args *FlagArgs, reply *bool) error {
	dbm := t.getMgr(args.Chain)
	return dbm.Commit(args.Flag)
}

// CancelFlag CancelFlag
func (t *TDb) CancelFlag(args *FlagArgs, reply *bool) error {
	dbm := t.getMgr(args.Chain)
	return dbm.Cancel(args.Flag)
}

// Rollback Rollback
func (t *TDb) Rollback(args *FlagArgs, reply *bool) error {
	dbm := t.getMgr(args.Chain)
	return dbm.Rollback(args.Flag)
}

// GetLastFlag GetLastFlag
func (t *TDb) GetLastFlag(chain *uint64, reply *([]byte)) error {
	dbm := t.getMgr(*chain)
	*reply = dbm.GetLastFlag()
	return nil
}
