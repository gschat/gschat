package mq

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsos/fs"
)

// Errors
var (
	ErrStorageExists = errors.New("storage already exists")

	ErrStorageNotExists = errors.New("storage not exists")

	ErrStorageBroken = errors.New("storage broken")

	ErrStorageResource = errors.New("storage alloc failed")
)

// Storage MQ storage object
type Storage interface {
	// Alloc new block, return block start offset
	Alloc(name string) (FIFO, error)

	Open(meta *QMeta) FIFO

	Free(FIFO) error

	IsFull() bool

	RunOnce(*bolt.DB) error

	Path() string
}

type _Storage struct {
	sync.Mutex                  // storage locker
	gslogger.Log                // mixin log APIs
	filepath     string         // storage filepath
	headersize   uint32         // header size
	bytesize     int64          // storage support max byte size
	blocksize    uint32         // storage support max blocksize
	blocks       uint32         // storage blocks
	nextblock    uint32         // next not used block indexer
	blocklist    []uint32       // storage block list
	cachedsize   uint32         // memory cached size
	ringbuffers  []*_RingBuffer // allocated ringbuffer
}

// create a new storage space
func newStorage(filepath string, blocksize, blocks, cachedsize uint32) (Storage, error) {

	gserrors.Assert(blocksize%2 == 0, "even number expect")

	if fs.Exists(filepath) {
		return nil, gserrors.Newf(ErrStorageExists, "storage file already exists :%s", filepath)
	}

	storage := &_Storage{
		Log:         gslogger.Get("storage"),
		filepath:    filepath,
		bytesize:    int64(blocksize) * int64(blocks),
		blocks:      blocks,
		blocksize:   blocksize,
		blocklist:   make([]uint32, blocks),
		cachedsize:  cachedsize,
		ringbuffers: make([]*_RingBuffer, blocks),
	}

	if storage.bytesize%1024 != 0 {
		storage.bytesize = (storage.bytesize/1024 + 1) * 1024
	}

	storage.D("creating storage\n\tfile :%s\n\tbytesize :%d, blocks :%d, blocksize :%d, listheader :%d",
		storage.filepath,
		storage.bytesize,
		storage.blocks,
		storage.blocksize,
		storage.nextblock,
	)

	// write storage header
	file, err := os.Create(storage.filepath)

	defer file.Close()

	if err != nil {
		return nil, err
	}

	header := make([]byte, 22+4*len(storage.blocklist))

	binary.BigEndian.PutUint16(header, uint16(len(header)))

	binary.BigEndian.PutUint64(header[2:], uint64(storage.bytesize))

	binary.BigEndian.PutUint32(header[10:], uint32(storage.blocksize))

	binary.BigEndian.PutUint32(header[14:], uint32(storage.blocks))

	binary.BigEndian.PutUint32(header[18:], uint32(storage.nextblock))

	for i := uint32(0); i < uint32(len(storage.blocklist)); i++ {

		storage.blocklist[i] = i + 1

		binary.BigEndian.PutUint32(header[22+i*4:], uint32(storage.blocklist[i]))
	}

	_, err = file.Write(header)

	storage.headersize = uint32(len(header))

	if err != nil {
		return nil, gserrors.Newf(err, "create new storage file error")
	}

	placehold := make([]byte, 1024)

	for i := int64(0); i < storage.bytesize/int64(1024); i++ {
		_, err = file.Write(placehold)

		if err != nil {
			return nil, gserrors.Newf(err, "write placehold data error")
		}
	}

	storage.D("create storage -- scuccess\n\tfile :%s\n\tbytesize :%d, blocks :%d, blocksize :%d, listheader :%d",
		storage.filepath,
		storage.bytesize,
		storage.blocks,
		storage.blocksize,
		storage.nextblock,
	)

	return storage, nil
}

func openStorage(filepath string, cachedsize uint32) (Storage, error) {
	if !fs.Exists(filepath) {
		return nil, gserrors.Newf(ErrStorageNotExists, "storage file not exists :%s", filepath)
	}

	storage := &_Storage{
		Log:        gslogger.Get("storage"),
		filepath:   filepath,
		cachedsize: cachedsize,
	}

	// read header
	// write storage header
	file, err := os.Open(storage.filepath)
	defer file.Close()

	if err != nil {
		return nil, gserrors.Newf(err, "open storage file error :%s", storage.filepath)
	}

	buff := make([]byte, 2)

	_, err = file.Read(buff)

	if err != nil {
		return nil, gserrors.Newf(err, "read storage header error")
	}

	length := binary.BigEndian.Uint16(buff)

	if length < 22 {
		return nil, gserrors.Newf(ErrStorageBroken, "storage file broken")
	}

	header := make([]byte, length-2)

	// read header
	_, err = file.Read(header)

	if err != nil {
		return nil, gserrors.Newf(err, "read storage header error")
	}

	storage.bytesize = int64(binary.BigEndian.Uint64(header[0:]))

	storage.blocksize = (binary.BigEndian.Uint32(header[8:]))

	storage.blocks = (binary.BigEndian.Uint32(header[12:]))

	storage.nextblock = (binary.BigEndian.Uint32(header[16:]))

	storage.blocklist = make([]uint32, uint32(length-22)/4)

	storage.ringbuffers = make([]*_RingBuffer, storage.blocks)

	storage.D("open storage\n\tfile :%s\n\tbytesize :%d, blocks :%d, blocksize :%d, listheader :%d,blocklist :%d",
		storage.filepath,
		storage.bytesize,
		storage.blocks,
		storage.blocksize,
		storage.nextblock,
		len(storage.blocklist),
	)

	for i := 0; i < len(storage.blocklist); i++ {
		storage.blocklist[i] = (binary.BigEndian.Uint32(header[20+i*4:]))
	}

	storage.headersize = uint32(length)

	fi, err := file.Stat()

	if err != nil {
		return nil, gserrors.Newf(err, "read storage header error")
	}

	if (storage.bytesize + int64(length)) != fi.Size() {
		return nil, gserrors.Newf(ErrStorageBroken, "storage filesize(%d) dismatch header.bytesize(%d)", fi.Size()-int64(length), storage.bytesize)
	}

	return storage, nil
}

func (storage *_Storage) Path() string {
	return storage.filepath
}

func (storage *_Storage) Alloc(name string) (FIFO, error) {

	storage.Lock()
	defer storage.Unlock()

	if storage.nextblock == storage.blocks {
		return nil, ErrStorageResource
	}

	file, err := os.OpenFile(storage.filepath, os.O_WRONLY, 0666)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	nextblock := storage.nextblock

	storage.D("before alloc nextblock(%d)", nextblock)

	storage.nextblock, storage.blocklist[nextblock] = storage.blocklist[nextblock], math.MaxUint32

	storage.D("after alloc nextblock(%d)", storage.nextblock)

	buff := [4]byte{}

	binary.BigEndian.PutUint32(buff[:], uint32(storage.nextblock))

	file.WriteAt(buff[:], 18)

	binary.BigEndian.PutUint32(buff[:], uint32(storage.blocklist[nextblock]))

	file.WriteAt(buff[:], int64(22+4*nextblock))

	ringbuffer := storage.newRingBuffer(name, nextblock, storage.cachedsize)

	storage.ringbuffers[nextblock] = ringbuffer

	return ringbuffer, nil
}

func (storage *_Storage) IsFull() bool {
	storage.Lock()
	defer storage.Unlock()

	return storage.nextblock == storage.blocks
}

func (storage *_Storage) Open(meta *QMeta) FIFO {

	storage.Lock()
	defer storage.Unlock()

	fifo := storage.openRingBuffer(meta, storage.cachedsize)

	storage.ringbuffers[meta.ID] = fifo

	return fifo
}

func (storage *_Storage) Free(fifo FIFO) error {

	meta := fifo.Meta()

	storage.Lock()
	defer storage.Unlock()

	file, err := os.OpenFile(storage.filepath, os.O_WRONLY, 0666)

	if err != nil {
		return err
	}

	defer file.Close()

	storage.blocklist[meta.ID], storage.nextblock = storage.nextblock, meta.ID

	buff := [4]byte{}

	binary.BigEndian.PutUint32(buff[:], uint32(meta.ID))

	file.WriteAt(buff[:], 18)

	binary.BigEndian.PutUint32(buff[:], uint32(storage.blocklist[meta.ID]))

	file.WriteAt(buff[:], int64(18+4*meta.ID))

	storage.ringbuffers[meta.ID] = nil

	return nil
}

func (storage *_Storage) RunOnce(db *bolt.DB) error {

	storage.Lock()
	defer storage.Unlock()

	file, err := os.OpenFile(storage.filepath, os.O_RDWR, 0666)

	if err != nil {
		return err
	}

	defer file.Close()

	for _, ringbuff := range storage.ringbuffers {

		if ringbuff == nil {
			continue
		}

		ringbuff.flush(file, storage.headersize)

		var buff bytes.Buffer

		err := WriteQMeta(&buff, ringbuff.Meta())

		if err != nil {
			return err
		}

		err = db.Update(func(tx *bolt.Tx) error {

			bucket, err := tx.CreateBucketIfNotExists([]byte(MetaTable))
			if err != nil {
				return gserrors.Newf(err, "create bucket storage error")
			}

			return bucket.Put([]byte(ringbuff.Name), buff.Bytes())
		})

		if err != nil {
			return err
		}
	}

	return nil
}
