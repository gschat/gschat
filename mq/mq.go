package mq

import (
	"bytes"
	"path/filepath"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// ...
const (
	MetaTable = "storage"
)

// MQ gsim persistent message queue
type MQ struct {
	sync.Mutex                      // locker
	gslogger.Log                    // Mixin log APIs
	db           *bolt.DB           // bolt metadata db
	path         string             // save path
	storages     map[string]Storage // loade storages
	freeStorages []Storage          // current storage
	cachedsize   uint32             // cached size
	blocksize    uint32             // storage blocksize
	blocks       uint32             // blocks of storage file
}

// NewMQ create new MQ
func NewMQ(path string) (mq *MQ, err error) {
	mq = &MQ{
		Log:        gslogger.Get("MQ"),
		path:       path,
		storages:   make(map[string]Storage),
		blocks:     gsconfig.Uint32("imnode.mq.blocks", 1024),
		blocksize:  gsconfig.Uint32("imnode.mq.blockszie", 1024*1024),
		cachedsize: gsconfig.Uint32("imnode.mq.cachedsize", 128),
	}

	mq.db, err = bolt.Open(filepath.Join(path, "storage.db"), 0600, nil)

	go func() {
		for _ = range time.Tick(gsconfig.Seconds("imnode.mq.flush_timeout", 2)) {

			mq.flushAll()
		}
	}()

	return
}

func (mq *MQ) flushAll() {
	mq.Lock()
	defer mq.Unlock()

	for _, storage := range mq.storages {
		storage.RunOnce(mq.db)
	}
}

func (mq *MQ) newStorageFileName() string {
	return filepath.Join(mq.path, time.Now().Format("20060102150405.999999999")+".storage")
}

func (mq *MQ) getStorage() (Storage, error) {

	if len(mq.freeStorages) == 0 {
		storage, err := newStorage(mq.newStorageFileName(), mq.blocksize, mq.blocks, mq.cachedsize)

		if err != nil {
			return nil, err
		}

		mq.storages[storage.Path()] = storage

		return storage, nil
	}

	storage := mq.freeStorages[len(mq.freeStorages)-1]

	mq.freeStorages = mq.freeStorages[:len(mq.freeStorages)-1]

	return storage, nil
}

func (mq *MQ) freeStorage(storage Storage) {
	mq.freeStorages = append(mq.freeStorages, storage)
}

// Open open user FIFO
func (mq *MQ) Open(username string) (FIFO, error) {
	mq.Lock()
	defer mq.Unlock()

	var meta *QMeta

	err := mq.db.Update(func(tx *bolt.Tx) error {

		bucket, err := tx.CreateBucketIfNotExists([]byte(MetaTable))

		if err != nil {
			return gserrors.Newf(err, "create bucket storage error")
		}

		buff := bucket.Get([]byte(username))

		if buff != nil {
			meta, err = ReadQMeta(bytes.NewBuffer(buff))
		}

		return err
	})

	if err != nil {
		return nil, err
	}

	if meta != nil {
		storage, ok := mq.storages[meta.FilePath]
		if !ok {
			storage, err = openStorage(meta.FilePath, mq.cachedsize)

			if err != nil {
				return nil, err
			}

			if !storage.IsFull() {
				mq.freeStorage(storage)
			}

			mq.storages[meta.FilePath] = storage
		}

		return storage.Open(meta), nil
	}

	storage, err := mq.getStorage()

	if err != nil {
		return nil, err
	}

	fifo, err := storage.Alloc(username)

	if !storage.IsFull() {
		mq.freeStorage(storage)
	}

	return fifo, err
}
