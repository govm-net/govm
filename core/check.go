package acb2fb3994c274446f5dd4d8397d2f73ad68f32f649e2577c23877f3a4d7e1a05

import (
	"errors"
	"fmt"

	"github.com/lengzhao/govm/runtime"
)

// CheckTransaction check
func CheckTransaction(chain uint64, key []byte) (uint32, error) {
	if chain == 0 {
		return 0, errors.New("not support,chain == 0")
	}
	stream, _ := runtime.DbGet(dbTransactionData{}, chain, key)
	if stream == nil {
		return 0, errors.New("transaction data is not exist")
	}

	trans := DecodeTrans(stream)
	if trans == nil {
		return 0, errors.New("fail to decode transaction")
	}
	if trans.Chain != chain {
		if trans.Chain != 0 {
			return 0, errors.New("error chain id")
		}
		if trans.User != author {
			return 0, errors.New("error chain id")
		}
	}
	if trans.Energy <= uint64(len(stream)) {
		return 0, errors.New("not enough energy")
	}
	var cost uint64
	getDataFormDB(chain, dbCoin{}, trans.User[:], &cost)
	if cost == 0 || trans.Cost+trans.Energy > cost {
		if trans.User != author {
			return 0, errors.New("not enough cost")
		}
	}
	switch trans.Ops {
	case OpsTransfer:
		if len(trans.Data) >= 100 {
			return 0, errors.New("data length over 100")
		}
	case OpsMove:
		if len(trans.Data) >= 100 {
			return 0, errors.New("data length over 100")
		}
		if trans.Energy <= 200000 {
			return 0, errors.New("not enough energy")
		}
		var dst uint64
		runtime.Decode(trans.Data, &dst)
		if dst == 0 {
			return 0, errors.New("error dst chain = 0")
		}
		if dst/2 != chain && chain/2 != dst {
			return 0, errors.New("error dst chain")
		}

		id := GetLastBlockIndex(dst)
		if id == 0 {
			return 0, errors.New("error,no exist dst chain")
		}

	case OpsNewChain:
	case OpsNewApp:
	case OpsRunApp:
		info := AppInfo{}
		getDataFormDB(chain, dbApp{}, trans.Data[:HashLen], &info)
		if info.Flag&AppFlagRun == 0 {
			return 0, errors.New("could not run the app")
		}

	case OpsUpdateAppLife:
	case OpsRegisterMiner:
		var guerdon uint64
		getDataFormDB(chain, dbStat{}, []byte{StatGuerdon}, &guerdon)
		if trans.Cost < 3*guerdon {
			return 0, fmt.Errorf("no enough cost,hope:%d,have:%d", 3*guerdon, trans.Cost)
		}
	case OpsDisableAdmin:
		if len(trans.Data) != 1 {
			return 0, fmt.Errorf("OpsDisableAdmin,error data len:%d", len(trans.Data))
		}
	default:
		return 0, errors.New("not support ops")
	}
	return uint32(len(stream)), nil
}
