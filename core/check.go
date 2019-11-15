package ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c

import (
	"errors"
	"fmt"
	"github.com/lengzhao/govm/conf"
	"log"
	"runtime/debug"

	"github.com/lengzhao/govm/runtime"
)

// CheckTransaction check trans for mine
func CheckTransaction(chain uint64, key []byte) (l uint32, err error) {
	defer func() {
		e := recover()
		if e != nil {
			log.Println("something error,", e)
			log.Println(string(debug.Stack()))
			err = fmt.Errorf("recover:%s", e)
		}
	}()
	if chain == 0 {
		return 0, errors.New("not support,chain == 0")
	}

	if runtime.DbExist(dbTransInfo{}, chain, key) {
		return 0, errors.New("transaction is exist")
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
			return 0, errors.New("error trans user")
		}
	}

	if trans.Energy < conf.GetConf().EnergyOfTrans {
		return 0, errors.New("not enough energy1")
	}

	if trans.Energy < uint64(len(stream)) {
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
		if len(stream) >= 300 {
			return 0, errors.New("transaction length over 300")
		}
		if len(trans.Data) < AddressLen {
			return 0, errors.New("error peer address of transfer")
		}
	case OpsMove:
		if len(stream) >= 300 {
			return 0, errors.New("transaction length over 300")
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
		if len(stream) >= 300 {
			return 0, errors.New("transaction length over 300")
		}
		var newChain uint64
		runtime.Decode(trans.Data, &newChain)
		if newChain/2 != chain {
			return 0, errors.New("error chain id")
		}
		id := GetLastBlockIndex(newChain)
		if id > 0 {
			return 0, errors.New("the chain is exist")
		}
		if newChain%2 == 1 {
			id := GetLastBlockIndex(2 * chain)
			if id == 0 {
				return 0, errors.New("the left child chain is not exist")
			}
		}
	case OpsNewApp:
		if len(stream) >= 81920 {
			return 0, errors.New("transaction length over 81920")
		}
		appCode := trans.Data
		appName := runtime.GetHash(appCode)
		ni := newAppInfo{}
		runtime.Decode(appCode, &ni)
		if ni.Flag&AppFlagPlublc == 0 {
			appName = runtime.GetHash(append(appName, trans.User[:]...))
		}
		appInfo := AppInfo{}
		getDataFormDB(chain, dbApp{}, appName, &appInfo)
		if appInfo.Life > trans.Time {
			return 0, fmt.Errorf("exist app:%x", appName)
		}
		runtime.NewApp(chain, appName, appCode)

	case OpsRunApp:
		if len(stream) >= 2048 {
			return 0, errors.New("transaction length over 2048")
		}
		info := AppInfo{}
		getDataFormDB(chain, dbApp{}, trans.Data[:HashLen], &info)
		if info.Flag&AppFlagRun == 0 {
			return 0, errors.New("could not run the app")
		}

	case OpsUpdateAppLife:
		var info UpdateInfo
		n := runtime.Decode(trans.Data, &info)
		if n != len(trans.Data) {
			return 0, errors.New("error len of transaction.data")
		}
	case OpsRegisterMiner:
		if len(stream) >= 300 {
			return 0, errors.New("transaction length over 300")
		}
		var guerdon uint64
		getDataFormDB(chain, dbStat{}, []byte{StatGuerdon}, &guerdon)
		if trans.Cost < 3*guerdon {
			return 0, fmt.Errorf("no enough cost,hope:%d,have:%d", 3*guerdon, trans.Cost)
		}
		info := RegMiner{}
		runtime.Decode(trans.Data, &info)
		if info.Chain != 0 {
			if info.Chain > chain {
				if info.Chain/2 != chain {
					return 0, errors.New("error dst chain")
				}
				return uint32(len(stream)), nil
			} else if chain > info.Chain {
				if chain/2 != info.Chain {
					return 0, errors.New("error dst chain")
				}
				return uint32(len(stream)), nil
			}
		}
		id := GetLastBlockIndex(chain)
		if info.Index <= id+25 || info.Index > id+depositCycle {
			return 0, errors.New("error index")
		}
	// case OpsDisableAdmin:
	default:
		return 0, errors.New("not support ops")
	}
	return uint32(len(stream)), nil
}
