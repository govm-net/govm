package zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f

import (
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/govm-net/govm/runtime"
	"github.com/govm-net/govm/wallet"
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
	out.Time = uint64(time.Now().Unix() * 1000)
	out.Energy = 1000000
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
}

// CreateMove 转出货币到其他相邻链
func (t *StTrans) CreateMove(dstChain, value uint64) {
	t.Cost = value
	t.Ops = OpsMove
	t.Data = runtime.Encode(dstChain)
}

// CreateNewChain 创建新链
func (t *StTrans) CreateNewChain(chain, value uint64) {
	t.Cost = value
	t.Ops = OpsNewChain
	t.Data = runtime.Encode(chain)
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
	if len(data) > 0 {
		t.Data = append(t.Data, data...)
	}
	energy := 20*uint64(len(t.Data)) + 10000
	if energy > t.Energy {
		t.Energy = energy
	}
}

// CreateUpdateAppLife update app life
func (t *StTrans) CreateUpdateAppLife(app Hash, life uint64) {
	t.Cost = (life/TimeHour + 1) * 2000
	t.Ops = OpsUpdateAppLife
	info := UpdateInfo{}
	info.Name = app
	info.Life = life
	t.Data = runtime.Encode(info)
}

// RegisterMiner RegisterMiner
func (t *StTrans) RegisterMiner(chain, cost uint64, peer []byte) {
	t.Cost = cost
	t.Ops = OpsRegisterMiner
	if chain != 0 && chain != t.Chain {
		t.Data = runtime.Encode(chain)
	}
	if len(peer) > 0 {
		t.Data = runtime.Encode(chain)
		t.Data = append(t.Data, peer...)
	}
}

// RegisterAdmin RegisterAdmin(candidates)
func (t *StTrans) RegisterAdmin(cost uint64) {
	t.Cost = cost
	t.Ops = OpsRegisterAdmin
}

// VoteAdmin VoteAdmin
func (t *StTrans) VoteAdmin(cost uint64, admin []byte) {
	t.Cost = cost
	t.Ops = OpsVote
	t.Data = admin
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
	trans := DecodeTrans(data)
	if trans == nil {
		return errors.New("error trans")
	}
	exist := runtime.DbExist(dbTransactionData{}, chain, trans.Key)
	if exist {
		return nil
	}
	if trans.Chain != chain {
		return errors.New("error chain")
	}

	return runtime.AdminDbSet(dbTransactionData{}, chain, trans.Key, data, 2<<50)
}

// DeleteTransaction delete Transaction
func DeleteTransaction(chain uint64, key []byte) error {
	return runtime.AdminDbSet(dbTransactionData{}, chain, key, nil, 0)
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
		out["ops"] = "Transfer"
		out["peer"] = hex.EncodeToString(data[:AddressLen])
	case OpsMove:
		out["ops"] = "Move"
		var chain uint64
		runtime.Decode(data, &chain)
		out["peer"] = chain
	case OpsNewApp:
		out["ops"] = "NewAPP"
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
	case OpsRunApp:
		out["ops"] = "RunAPP"
		name := Hash{}
		n := runtime.Decode(data, &name)
		out["name"] = hex.EncodeToString(name[:])
		out["data"] = hex.EncodeToString(data[n:])
	case OpsNewChain:
		out["ops"] = "NewChain"
		var chain uint64
		runtime.Decode(data, &chain)
		out["peer"] = chain
	case OpsUpdateAppLife:
		info := UpdateInfo{}
		runtime.Decode(data, &info)
		out["Name"] = hex.EncodeToString(info.Name[:])
		out["Life"] = info.Life
	case OpsRegisterMiner:
		out["ops"] = "RegisterMiner"
		var dstChain uint64
		if len(data) > 0 {
			n := runtime.Decode(data, &dstChain)
			if len(data) > AddressLen {
				var miner Address
				runtime.Decode(data[n:], &miner)
				out["miner"] = hex.EncodeToString(miner[:])
			}
		}
		out["chain"] = dstChain
	case OpsRegisterAdmin:
		out["ops"] = "RegisterAdmin"
	case OpsUnvote:
		out["ops"] = "Unvote"
		if len(data) > 0 {
			out["voter"] = hex.EncodeToString(data)
		}
	case OpsVote:
		out["ops"] = "Vote"
		if len(data) < AddressLen {
			break
		}
		out["admin"] = hex.EncodeToString(data[:AddressLen])
	}
	return out
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

// GetAdminList get admin list
func GetAdminList(chain uint64) []Address {
	var out [AdminNum]Address
	getDataFormDB(chain, dbStat{}, []byte{StatAdmin}, &out)
	return out[:]
}

// GetAdminInfo get admin info
func GetAdminInfo(chain uint64, addr Address) AdminInfo {
	var out AdminInfo
	getDataFormDB(chain, dbAdmin{}, addr[:], &out)
	return out
}

// GetVoteInfo get vote info
func GetVoteInfo(chain uint64, addr Address) VoteInfo {
	var out VoteInfo
	getDataFormDB(chain, dbVote{}, addr[:], &out)
	return out
}

// GetVoteReward get reward info
func GetVoteReward(chain, day uint64) RewardInfo {
	var out RewardInfo
	getDataFormDB(chain, dbVoteReward{}, runtime.Encode(day), &out)
	return out
}

func getDataFormDB(chain uint64, db interface{}, key []byte, out interface{}) {
	if chain == 0 {
		return
	}
	stream, _ := runtime.DbGet(db, chain, key)
	if len(stream) > 0 {
		runtime.Decode(stream, out)
	}
}
