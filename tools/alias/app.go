package a1000000000000000000000000000000000000000000000000000000000000000

import core "github.com/govm-net/govm/tools/debugger/zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

type tApp struct {
}

const (
	costLimit = 1000000000
)

// OpsType ops type define
type OpsType byte

// type of ops
const (
	SetAlias = OpsType(iota)
	SetUI
	DeleteAlias
	SyncToChild
	AckFromParent
	UpdateLife
	Transfer
	DeleteUI
)

// AliasInfo alias of app
type aliasInfo struct {
	Alias       string                 `json:"alias,omitempty"`
	App         core.Hash              `json:"app,omitempty"`
	Creator     core.Address           `json:"creator,omitempty"`
	Time        uint64                 `json:"time,omitempty"`
	Description string                 `json:"desc,omitempty"`
	Others      map[string]interface{} `json:"others,omitempty"`
}

type uiInfo struct {
	Info      map[string]interface{} `json:"info,omitempty"`
	Items     []interface{}          `json:"items,omitempty"`
	ViewItems []interface{}          `json:"view_items,omitempty"`
}

var owner = core.Address{2, 152, 64, 16, 49, 156, 211, 70, 89, 247, 252, 178, 11, 49, 214, 21, 216, 80, 171, 50, 202, 147, 6, 24}

const dfLife = core.TimeYear * 2

func assert(cond bool, msg interface{}) {
	if !cond {
		panic(msg)
	}
}

func run(user, in []byte, cost uint64) {
	core.Event(tApp{}, "start_app", user, in)
	t := in[0]
	in = in[1:]
	switch OpsType(t) {
	case SetAlias:
		assert(cost >= costLimit, "cost < 1t9")
		fSetAlias(user, in)
	case DeleteAlias:
		fDeleteAlias(user, in)
	case SyncToChild:
		assert(cost >= costLimit*10, "cost < 10t9")
		fSyncToChild(user, in)
	case AckFromParent:
		fAckFromParent(user, in)
	case SetUI:
		assert(cost >= costLimit, "cost < 1t9")
		fSetUI(user, in)
	case UpdateLife:
		assert(cost >= costLimit, "cost < 1t9")
		fUpdateLife(in)
	case Transfer:
		var c uint64
		core.Decode(0, in, &c)
		core.TransferAccounts(tApp{}, owner, c)
	case DeleteUI:
		fDeleteUI(user, in)
	default:
		assert(false, "not support")
	}
}
func getTime() uint64 {
	stream, _ := core.GetDBData("dbStat", []byte{core.StatBaseInfo})
	assert(len(stream) > 0, "fail to read baseInfo")
	info := core.BaseInfo{}
	core.Decode(0, stream, &info)
	return info.Time
}

func fSetAlias(user, in []byte) error {
	info := aliasInfo{}
	core.Decode(core.EncJSON, in, &info)
	core.Decode(0, user, &info.Creator)
	appInfo := core.GetAppInfo(info.App)
	assert(appInfo != nil, "not found app")
	assert(len(info.Description) < 200, "desc too long(>=200)")
	db := core.GetDB(aliasInfo{})

	stream, _ := db.Get([]byte(info.Alias))
	if len(stream) > 0 {
		old := aliasInfo{}
		core.Decode(core.EncJSON, stream, &old)
		assert(info.Creator == old.Creator, "exist alias")
		assert(info.App == old.App, "different app")
		info.Time = old.Time
	} else {
		info.Time = getTime()
	}
	array := []rune(info.Alias)
	for i := 0; i < len(array); i++ {
		if array[i] == '.' {
			assert(false, "include point in alias")
		}
	}
	db.Set([]byte(info.Alias), core.Encode(core.EncJSON, info), dfLife)
	core.Event(tApp{}, "SetAlias", []byte(info.Alias))
	return nil
}

func fDeleteAlias(user, in []byte) {
	creator := core.Address{}
	core.Decode(0, user, &creator)
	db := core.GetDB(aliasInfo{})
	stream, _ := db.Get(in)
	assert(len(stream) > 0, "alias not exist")
	old := aliasInfo{}
	core.Decode(core.EncJSON, stream, &old)
	assert(old.Creator == creator, "different creator")
	db.Set(in, nil, 0)

	in = append(in, '.')
	in = append(in, 0)
	db = core.GetDB(uiInfo{})
	db.Set(in, nil, 0)
	core.Event(tApp{}, "DeleteAlias", in)
}

func fSyncToChild(user, in []byte) {
	db := core.GetDB(aliasInfo{})
	stream, _ := db.Get(in)
	assert(len(stream) > 0, "alias not exist")
	info := aliasInfo{}
	core.Decode(core.EncJSON, stream, &info)
	log := core.GetLog(aliasInfo{})
	log.Write(in, core.Encode(core.EncJSON, info))
	core.Event(tApp{}, "SyncToChild", in)
}

func getChainID() uint64 {
	stream, _ := core.GetDBData("dbStat", []byte{core.StatBaseInfo})
	assert(len(stream) > 0, "fail to read baseInfo")
	info := core.BaseInfo{}
	core.Decode(0, stream, &info)
	return info.Chain
}
func fAckFromParent(user, in []byte) {
	db := core.GetDB(aliasInfo{})
	_, life := db.Get(in)
	if life > 0 {
		assert(life+core.TimeMonth > dfLife, "exist alias")
		core.Event(tApp{}, "replace", in)
	}
	log := core.GetLog(aliasInfo{})
	chain := getChainID() / 2
	stream := log.Read(chain, in)
	assert(len(stream) > 0, "not found alias in log")
	db.Set(in, stream, dfLife)
	core.Event(tApp{}, "AckFromParent", in)
}

type setUIInfo struct {
	Index byte   `json:"index,omitempty"`
	Alias string `json:"alias,omitempty"`
	UI    uiInfo `json:"ui,omitempty"`
}

func fSetUI(user, in []byte) {
	info := setUIInfo{}
	creator := core.Address{}
	core.Decode(core.EncJSON, in, &info)
	core.Decode(0, user, &creator)
	db := core.GetDB(aliasInfo{})
	stream, _ := db.Get([]byte(info.Alias))
	assert(len(stream) > 0, "alias not exist")
	old := aliasInfo{}
	core.Decode(core.EncJSON, stream, &old)
	assert(creator == old.Creator, "exist alias")
	assert(info.Index < 250, "too many UIs")
	uiDB := core.GetDB(uiInfo{})
	if info.Index > 0 {
		key := []byte(info.Alias)
		key = append(key, '.')
		key = append(key, info.Index-1)
		_, life := uiDB.Get(key)
		assert(life > 0, "not exist pre index")
	}
	key := []byte(info.Alias)
	key = append(key, '.')
	key = append(key, info.Index)
	uiDB.Set(key, core.Encode(core.EncJSON, info.UI), dfLife)
	core.Event(tApp{}, "SetUI", key)
}

func fDeleteUI(user, in []byte) {
	creator := core.Address{}
	core.Decode(0, user, &creator)
	db := core.GetDB(aliasInfo{})
	alias := in[:len(in)-3]
	stream, _ := db.Get(alias)
	assert(len(stream) > 0, "alias not exist")
	old := aliasInfo{}
	core.Decode(core.EncJSON, stream, &old)
	assert(creator == old.Creator, "exist alias")
	uiDB := core.GetDB(uiInfo{})
	uiDB.Set(in, nil, 0)
	core.Event(tApp{}, "fDeleteUI", in)
}

// type,life,key
func fUpdateLife(in []byte) {
	var life uint64
	var typ byte
	var key []byte
	typ = in[0]
	n := core.Decode(0, in[1:], &life)
	key = in[n+1:]
	var db *core.DB
	if typ == 0 {
		db = core.GetDB(aliasInfo{})
	} else {
		db = core.GetDB(uiInfo{})
	}
	stream, ol := db.Get(key)
	assert(life > ol, "error life")
	db.Set(key, stream, life)
	core.Event(tApp{}, "fUpdateLife", in)
}
