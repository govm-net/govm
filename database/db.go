package database

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
	"path"
	"sync"
)

const (
	dataFN        = "data.db"
	flagFN        = "flag.db"
	tbFlagID      = "_tbname_flag_id_"
	tbIDFlag      = "_tbname_id_flag_"
	tbBuckets     = "_tbname_buckets_"
	closeFlagFile = "closed.flag"
)

// TDbMgr 数据库管理器
type TDbMgr struct {
	mu        sync.Mutex
	flagDb    *bolt.DB
	historyDb *bolt.DB
	dataDb    *bolt.DB
	flag      []byte
	dir       string
	id        uint64
}

// OpenMgr open db manager
func OpenMgr(dir string) *TDbMgr {
	out := new(TDbMgr)
	out.dir = dir
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, 0600)
		log.Println("create dir:", dir, err)
	}
	out.dataDb, err = bolt.Open(path.Join(dir, dataFN), 0600, nil)
	if err != nil {
		log.Println("fail to open file:", dir, dataFN, err)
		return nil
	}
	out.flagDb, err = bolt.Open(path.Join(dir, flagFN), 0600, nil)
	if err != nil {
		out.dataDb.Close()
		log.Println("fail to open file:", dir, flagFN, err)
		return nil
	}
	err = out.flagDb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(tbFlagID))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(tbIDFlag))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		out.dataDb.Close()
		out.flagDb.Close()
		log.Println("fail to open file:", dir, flagFN, err)
		return nil
	}

	fn := path.Join(out.dir, closeFlagFile)
	defer os.Remove(fn)
	if _, err := os.Stat(fn); os.IsNotExist(err) {
		flag := out.GetLastFlag()
		if len(flag) == 0 {
			return out
		}
		go out.Rollback(flag)
	}

	return out
}

// Close 关闭管理器
func (m *TDbMgr) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dataDb != nil {
		m.dataDb.Close()
		m.dataDb = nil
	}
	if m.flagDb != nil {
		m.flagDb.Close()
		m.flagDb = nil
	}
	if m.historyDb != nil {
		m.historyDb.Close()
		m.historyDb = nil
	}
	if len(m.flag) == 0 {
		fn := path.Join(m.dir, closeFlagFile)
		f, err := os.Create(fn)
		if err != nil {
			return
		}
		defer f.Close()
		f.Write([]byte("closed"))
	}
}

func (m *TDbMgr) getHistoryFileName(index uint64) string {
	fn := fmt.Sprintf("history_%d.db", index)
	return path.Join(m.dir, fn)
}

func itoa(in uint64) []byte {
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, in)
	return out
}

// OpenFlag 开启标志，标志用于记录操作，支持批量操作的回滚
func (m *TDbMgr) OpenFlag(flag []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flag != nil {
		return errors.New("exist flag")
	}
	if len(flag) <= 8 {
		return fmt.Errorf("error flag length(>8):%x", flag)
	}
	err := m.flagDb.Update(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(tbFlagID))
		v := b1.Get(flag)
		if v != nil {
			log.Printf("try to open old flag:%x\n", flag)
			return fmt.Errorf("try to open old flag:%x", flag)
		}
		b2 := tx.Bucket([]byte(tbIDFlag))
		// id从1开始，0位保存最后一个id值
		v = b2.Get(itoa(0))
		if v != nil {
			m.id = binary.BigEndian.Uint64(v)
		} else {
			m.id = 0
		}
		m.id++
		b2.Put(itoa(m.id), flag)
		b2.Put(itoa(0), itoa(m.id))
		b1.Put(flag, itoa(m.id))
		return nil
	})
	if err != nil {
		log.Println("fail to set flag:", err)
		return err
	}
	hn := m.getHistoryFileName(m.id)
	os.Remove(hn)
	m.historyDb, err = bolt.Open(hn, 0600, nil)
	err = m.historyDb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(tbBuckets))
		if err != nil {
			log.Printf("fail to create bucket,%s\n", tbBuckets)
			return err
		}
		return err
	})
	if err != nil {
		return err
	}
	m.flag = flag
	// log.Printf("success open flag:%x\n", flag)
	return nil
}

// GetLastFlag 获取最后一个标志
func (m *TDbMgr) GetLastFlag() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flag != nil {
		return m.flag
	}
	var out []byte
	m.flagDb.View(func(tx *bolt.Tx) error {
		b2 := tx.Bucket([]byte(tbIDFlag))
		v := b2.Get(itoa(0))
		v = b2.Get(v)
		if v != nil {
			out = make([]byte, len(v))
			copy(out, v)
		}
		return nil
	})
	return out
}

// Commit 提交，将数据写入磁盘，标志清除
func (m *TDbMgr) Commit(flag []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if bytes.Compare(m.flag, flag) != 0 {
		return errors.New("difference flag")
	}
	if m.historyDb != nil {
		m.historyDb.Close()
		m.historyDb = nil
	}
	m.flag = nil
	// log.Printf("success Commit flag:%x\n", flag)
	if m.id > 20000 {
		hn := m.getHistoryFileName(m.id - 20000)
		os.Remove(hn)
		log.Println("[db]remove history:", hn)
	}
	return nil
}

// Cancel 取消提交，将数据回滚
func (m *TDbMgr) Cancel(flag []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if bytes.Compare(m.flag, flag) != 0 {
		return errors.New("difference flag")
	}
	if len(m.flag) == 0 {
		return nil
	}
	if m.historyDb != nil {
		m.historyDb.Close()
		m.historyDb = nil
	}
	m.flag = nil
	log.Printf("Cancel, dir:%s, flag:%x\n", m.dir, flag)
	return m.rollback(flag)
}

func (m *TDbMgr) getValueFromHistoryDb(id uint64, tbName, key []byte) []byte {
	hn := m.getHistoryFileName(id)
	hdb, err := bolt.Open(hn, 0600, nil)
	if err != nil {
		log.Println("fail to open history file:", hn)
		return nil
	}
	defer hdb.Close()

	var out []byte

	err = hdb.View(func(tx *bolt.Tx) error {
		//存储所有tbName的bucket
		b := tx.Bucket(tbName)
		if b == nil {
			log.Println("fail to open bucket:", tbBuckets)
			return errors.New("fail to open bucket")
		}
		v := b.Get(key)
		if v == nil {
			return nil
		}
		out = make([]byte, len(v))
		copy(out, v)
		return nil
	})

	return out
}

func (m *TDbMgr) setValueWithID(id uint64, tbName, key, value []byte) error {
	return m.dataDb.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(tbName)
		if err != nil {
			log.Printf("fail to create bucket,%x\n", tbName)
			return err
		}
		if value == nil {
			b.Delete(key)
		} else {
			err = b.Put(key, value)
			if err != nil {
				log.Println("fail to put:", err)
				return err
			}
		}
		b, err = tx.CreateBucketIfNotExists(getStatTbName(tbName))
		if err != nil {
			log.Printf("fail to create bucket,%x\n", tbName)
			return err
		}
		b.Put(key, itoa(id))

		return nil
	})
}

func getStatTbName(name []byte) []byte {
	out := []byte("_prefix_")
	out = append(out, name...)
	return out
}

func (m *TDbMgr) rollback(flag []byte) error {
	err := m.flagDb.View(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(tbFlagID))
		v := b1.Get(flag)
		if v == nil {
			return errors.New("not exist flag")
		}
		m.id = binary.BigEndian.Uint64(v)

		b2 := tx.Bucket([]byte(tbIDFlag))
		v2 := b2.Get(itoa(0))
		if bytes.Compare(v, v2) != 0 {
			return errors.New("not last flag")
		}
		return nil
	})
	if err != nil {
		return err
	}

	hn := m.getHistoryFileName(m.id)
	hdb, err := bolt.Open(hn, 0600, nil)
	if err != nil {
		log.Println("fail to open history file:", hn)
		return err
	}
	defer os.Remove(hn)
	defer hdb.Close()

	/*	遍历文件中的tbName，
		然后遍历每个table下的key，
		获取它上一次的值的位置id_i
		然后再从id_i的文件中得到value
		将value写到当前的dataDb中
	*/
	err = hdb.View(func(tx *bolt.Tx) error {
		//存储所有tbName的bucket
		b := tx.Bucket([]byte(tbBuckets))
		if b == nil {
			log.Println("fail to open bucket:", tbBuckets)
			return errors.New("fail to open bucket")
		}

		//遍历每个tbName
		b.ForEach(func(tbName, v []byte) error {
			//得到tbName对应的bucker
			tbBuck := tx.Bucket(getStatTbName(tbName))
			if tbBuck == nil {
				return nil
			}
			//遍历里面的每个记录，
			tbBuck.ForEach(func(key, hid []byte) error {
				oldID := binary.BigEndian.Uint64(hid)
				if oldID == 0 {
					//delete value
					m.setValueWithID(oldID, tbName, key, nil)
					return nil
				}
				v := m.getValueFromHistoryDb(oldID, tbName, key)
				//set value
				m.setValueWithID(oldID, tbName, key, v)
				return nil
			})

			return nil
		})

		return nil
	})
	if err != nil {
		log.Println("fail to rollback history value:", hn)
		// return err
	}

	m.flagDb.Update(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte(tbFlagID))
		v := b1.Get(flag)
		b1.Delete(flag)
		b2 := tx.Bucket([]byte(tbIDFlag))
		b2.Delete(v)
		b2.Put(itoa(0), itoa(m.id-1))
		return nil
	})
	return nil
}

// Rollback 将指定标志之后的所有操作回滚，要求当前没有开启标志
func (m *TDbMgr) Rollback(flag []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flag != nil {
		return errors.New("flag opend")
	}
	log.Printf("Rollback, dir:%s, flag:%x\n", m.dir, flag)
	return m.rollback(flag)
}

// SetWithFlag 写入数据，标志仅仅是一个标志，方便数据回滚
// 每个flag都有对应的historyDb文件，用于记录tbName.key的前一个标签记录位置
// 同时记录本标签最终设置的值，方便回滚
func (m *TDbMgr) SetWithFlag(flag, tbName, key, value []byte) error {
	//log.Printf("[db]tb:%s ,key: %x ,value: %x\n", tbName, key, value)
	m.mu.Lock()
	defer m.mu.Unlock()
	if bytes.Compare(m.flag, flag) != 0 {
		return errors.New("error flag")
	}
	var oldID uint64
	m.dataDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(getStatTbName(tbName))
		if b == nil {
			return nil
		}
		v := b.Get(key)
		if v == nil {
			return nil
		}
		oldID = binary.BigEndian.Uint64(v)
		return nil
	})

	err := m.historyDb.Update(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte(tbBuckets))
		if b == nil {
			log.Println("fail to open bucket:", tbBuckets)
			return errors.New("fail to open bucket")
		}
		err = b.Put(tbName, []byte{1})
		if err != nil {
			log.Println("fail to write history tbname:", err)
			return err
		}

		if oldID != m.id {
			b, err = tx.CreateBucketIfNotExists(getStatTbName(tbName))
			if err != nil {
				log.Printf("fail to create history stat bucket,%x\n", tbName)
				return err
			}
			b.Put(key, itoa(oldID))
		}

		b, err = tx.CreateBucketIfNotExists(tbName)
		if err != nil {
			log.Printf("fail to create history bucket,%x\n", tbName)
			return err
		}
		err = b.Put(key, value)
		return err
	})
	if err != nil {
		log.Println("fail to write history info:", err)
		return err
	}
	m.dataDb.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(tbName)
		if err != nil {
			log.Printf("fail to create bucket,%x\n", tbName)
			return err
		}
		err = b.Put(key, value)
		if err != nil {
			log.Println("fail to put:", err)
			return err
		}
		b, err = tx.CreateBucketIfNotExists(getStatTbName(tbName))
		if err != nil {
			log.Printf("fail to create bucket,%x\n", tbName)
			return err
		}
		b.Put(key, itoa(m.id))
		return err
	})
	return nil
}

// Set 存储数据，不携带标签，不会被回滚,tbName的put一值不用flag，否则可能导致数据混乱
func (m *TDbMgr) Set(tbName, key, value []byte) error {
	return m.dataDb.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(tbName)
		if err != nil {
			log.Printf("fail to create bucket,%x\n", tbName)
			return err
		}
		if len(value) == 0 {
			return b.Delete(key)
		}
		err = b.Put(key, value)
		if err != nil {
			log.Println("fail to put:", err)
			return err
		}
		return err
	})
}

// Get 获取数据
func (m *TDbMgr) Get(tbName, key []byte) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []byte
	m.dataDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(tbName)
		if b == nil {
			return nil
		}
		v := b.Get(key)
		if v != nil {
			out = make([]byte, len(v))
			copy(out, v)
		}
		return nil
	})

	return out
}

// Exist 数据是否存在
func (m *TDbMgr) Exist(tbName, key []byte) (bExist bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bExist = false
	m.dataDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(tbName)
		if b == nil {
			return nil
		}
		v := b.Get(key)
		if v != nil {
			bExist = true
		}
		return nil
	})

	return
}
