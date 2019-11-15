package ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/lengzhao/govm/conf"
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/govm/wallet"
	"log"
	"time"
)

// StTrans 交易的结构体
type StTrans struct {
	TransactionHead
	Sign []byte
	Data []byte
	Key  []byte
}

// NewTransaction create a new transaction struct
/* 1. NewTransaction
   2. Create*:CreateTransfer,CreateMove...
   3. update energy,time...
   4. GetSignData
   5. SetSign
   6. Output
*/
func NewTransaction(chain uint64, user Address) *StTrans {
	out := StTrans{}
	out.Chain = chain
	out.User = user
	out.Time = uint64(time.Now().Unix()*1000) - maxBlockInterval
	out.Energy = 10000
	return &out
}

// GetSignData 提取数据用于签名
func (t *StTrans) GetSignData() []byte {
	data := runtime.Encode(t.TransactionHead)
	data = append(data, t.Data...)
	return data
}

// SetSign 设置签名信息
func (t *StTrans) SetSign(in []byte) error {
	t.Sign = in
	return nil
}

// Output 输出数据，可以发布
func (t *StTrans) Output() []byte {
	out := make([]byte, 1, 1000)
	out[0] = uint8(len(t.Sign))
	out = append(out, t.Sign...)
	out = append(out, runtime.Encode(t.TransactionHead)...)
	out = append(out, t.Data...)
	t.Key = runtime.GetHash(out)
	return out
}

// CreateTransfer 创建转账交易
func (t *StTrans) CreateTransfer(payee Address, value uint64) {
	t.Cost = value
	t.Ops = OpsTransfer
	t.Data = payee[:]
	t.Energy = 10000000
}

// CreateMove 转出货币到其他相邻链
func (t *StTrans) CreateMove(dstChain, value uint64) {
	t.Cost = value
	t.Ops = OpsMove
	t.Data = runtime.Encode(dstChain)
	t.Energy = 100000000
}

// CreateNewChain 创建新链
func (t *StTrans) CreateNewChain(chain, value uint64) {
	t.Cost = value
	t.Ops = OpsNewChain
	t.Data = runtime.Encode(chain)
	t.Energy = 10000000
}

// CreateNewApp new app
func (t *StTrans) CreateNewApp(code []byte, lineNum uint64) {
	t.Energy = (lineNum+1)*TimeYear*2000/TimeDay + uint64(len(t.Data))*25 + 20000
	t.Energy *= 2
	t.Ops = OpsNewApp
	t.Data = code
}

// CreateRunApp run app
func (t *StTrans) CreateRunApp(app Hash, cost uint64, data []byte) {
	t.Cost = cost
	t.Ops = OpsRunApp
	t.Data = app[:]
	if data != nil {
		t.Data = append(t.Data, data...)
	}
	t.Energy = 20*uint64(len(t.Data)) + 10000
}

// CreateUpdateAppLife update app life
func (t *StTrans) CreateUpdateAppLife(app Hash, life uint64) {
	t.Cost = (life/TimeHour + 1) * 2000
	t.Ops = OpsUpdateAppLife
	info := UpdateInfo{}
	info.Name = app
	info.Life = life
	t.Data = runtime.Encode(info)
	t.Energy = 10000000
}

const (
	// MinerNum number of miner pre block
	MinerNum = 11
)

// CreateRegisterMiner RegisterMiner
func (t *StTrans) CreateRegisterMiner(chain, index, cost uint64) {
	info := RegMiner{chain, index}
	t.Cost = cost
	t.Ops = OpsRegisterMiner
	t.Data = runtime.Encode(info)
}

// DecodeTrans decode transaction data
func DecodeTrans(data []byte) *StTrans {
	out := new(StTrans)
	signLen := data[0]
	if signLen <= 30 || signLen >= 250 {
		return nil
	}

	out.Key = runtime.GetHash(data)
	out.Sign = data[1 : signLen+1]
	signData := data[signLen+1:]

	n := runtime.Decode(signData, &out.TransactionHead)
	out.Data = signData[n:]

	rst := wallet.Recover(out.User[:], out.Sign, signData)
	if !rst {
		log.Println("error sign")
		return nil
	}

	if out.User[0] == prefixOfPlublcAddr {
		return nil
	}

	if out.Energy < uint64(len(data)) {
		log.Println("not enough energy:", out.Energy)
		return nil
	}
	return out
}

// IsExistTransaction Determine whether transaction exists
func IsExistTransaction(chain uint64, key []byte) bool {
	return runtime.DbExist(dbTransactionData{}, chain, key)
}

// WriteTransaction write transaction data to database
func WriteTransaction(chain uint64, data []byte) error {
	key := runtime.GetHash(data)

	exist := runtime.DbExist(dbTransactionData{}, chain, key)
	if exist {
		return nil
	}
	trans := DecodeTrans(data)
	if trans == nil {
		return errors.New("error trans")
	}

	if trans.Chain != chain {
		c := conf.GetConf()
		if bytes.Compare(key, c.FirstTransName) != 0 {
			return errors.New("error trans")
		}
	}

	return runtime.AdminDbSet(dbTransactionData{}, chain, key, data, 2<<50)
}

// ReadTransactionData read transaction data
func ReadTransactionData(chain uint64, key []byte) []byte {
	stream, _ := runtime.DbGet(dbTransactionData{}, chain, key)
	return stream
}

// GetUserCoin get user coin number
func GetUserCoin(chain uint64, addr []byte) uint64 {
	var out uint64
	getDataFormDB(chain, dbCoin{}, addr, &out)
	return out
}

// DecodeOpsDataOfTrans decode ops data
func DecodeOpsDataOfTrans(ops uint8, data []byte) map[string]interface{} {
	out := make(map[string]interface{})
	switch ops {
	case OpsTransfer:
		peer := Address{}
		runtime.Decode(data, &peer)
		out["peer"] = peer
		out["info"] = hex.EncodeToString(data[AddressLen:])
		return out
	case OpsMove:
		var chain uint64
		runtime.Decode(data, &chain)
		out["peer"] = chain
		return out
	case OpsNewApp:
		ni := newAppInfo{}
		runtime.Decode(data, &ni)
		key := runtime.GetHash(data)
		out["code_hash"] = hex.EncodeToString(key)
		if ni.Flag&AppFlagPlublc != 0 {
			out["is_public"] = true
			out["app_name"] = out["code_hash"]
		} else {
			out["is_public"] = false
		}
		if ni.Flag&AppFlagRun != 0 {
			out["enable_run"] = true
		} else {
			out["enable_run"] = false
		}
		if ni.Flag&AppFlagImport != 0 {
			out["enable_import"] = true
		} else {
			out["enable_import"] = false
		}
		out["depend_num"] = ni.DependNum
		out["line_num"] = ni.LineNum
		return out
	case OpsRunApp:
		name := Hash{}
		n := runtime.Decode(data, &name)
		out["name"] = name
		out["data"] = hex.EncodeToString(data[n:])
		return out
	case OpsNewChain:
		var chain uint64
		runtime.Decode(data, &chain)
		out["peer"] = chain
		return out
	case OpsUpdateAppLife:
		info := UpdateInfo{}
		runtime.Decode(data, &info)
		out["Name"] = info.Name
		out["Life"] = info.Life
		return out
	case OpsRegisterMiner:
		info := RegMiner{}
		runtime.Decode(data, &info)
		out["index"] = info.Index
		out["chain"] = info.Chain
		return out
	}
	return nil
}

// GetAppInfoOfChain get app information
func GetAppInfoOfChain(chain uint64, name []byte) *AppInfo {
	out := AppInfo{}
	getDataFormDB(chain, dbApp{}, name, &out)
	return &out
}

// GetTransInfo get info of transaction
func GetTransInfo(chain uint64, key []byte) TransInfo {
	var out TransInfo
	getDataFormDB(chain, dbTransInfo{}, key, &out)
	return out
}

func getDataFormDB(chain uint64, db interface{}, key []byte, out interface{}) {
	if chain == 0 {
		return
	}
	stream, _ := runtime.DbGet(db, chain, key)
	if stream != nil {
		runtime.Decode(stream, out)
	}
}
