package mq

import (
	"testing"

	"github.com/boltdb/bolt"
	"github.com/gschat/gsim"

	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsos/fs"
)

const (
	storageFile = "./storage.1"

	blocksize = 1024 * 1024

	blocks = 1024

	cachedsize = 128

	name = "test"
)

var storage Storage

var db *bolt.DB

func init() {
	if fs.Exists(storageFile) {
		fs.RemoveAll(storageFile)
	}

	gslogger.NewFlags(gslogger.ERROR | gslogger.INFO)

	db, _ = bolt.Open("storage.db", 0600, nil)
}

func TestCreateStorage(t *testing.T) {
	_, err := newStorage(storageFile, blocksize, blocks, 128)

	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenStorage(t *testing.T) {

	storage, err := openStorage(storageFile, 0)

	if err != nil {
		t.Fatal(err)
	}

	st := storage.(*_Storage)

	if st.blocks != blocks {
		t.Fatalf("check blocks(%d) error,expect %d", st.blocks, blocks)
	}

	if st.blocksize != blocksize {
		t.Fatalf("check blocksize(%d) error,expect %d", st.blocksize, blocksize)
	}
}

func TestStorageAllocFree(t *testing.T) {

	storage, err := openStorage(storageFile, 0)

	if err != nil {
		t.Fatal(err)
	}

	var fifos []FIFO

	for i := 0; i < blocks; i++ {
		fifo, err := storage.Alloc(name)

		if err != nil {
			t.Fatalf("alloc blocks(%d) error %s", i, err)
		}

		fifos = append(fifos, fifo)
	}

	_, err = storage.Alloc(name)

	if err != ErrStorageResource {
		t.Fatal("check alloc overflowerror")
	}

	for _, fifo := range fifos {
		storage.Free(fifo)
	}
}

func TestFlushMessage(t *testing.T) {

	storage, err := openStorage(storageFile, 128)

	if err != nil {
		t.Fatal(err)
	}

	fifo, err := storage.Alloc(name)

	for i := 0; i < 32; i++ {
		msg := gsim.NewMessage()

		msg.Content = make([]byte, 1024*32)

		fifo.Push(msg)

		err = storage.RunOnce(db)

		if err != nil {
			t.Fatal(err)
		}
	}

	if fifo.MaxID() != 32 {
		t.Fatal("increase seq id error")
	}

	gslogger.Get("mq.fifo").D("message range [%d,%d)", fifo.MinID(), fifo.MaxID())

	stream := fifo.NewStream(0, 2)

	defer stream.Close()

	for i := fifo.MinID(); i < fifo.MaxID(); i++ {

		storage.RunOnce(db)

		msg := <-stream.Chan

		if msg.SeqID != i {
			t.Fatalf("stream read expect(%d) got(%d)", i, msg.SeqID)
		}
	}
}

func BenchmarkAllocFree(t *testing.B) {
	storage, err := openStorage(storageFile, 128)

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < t.N; i++ {
		block, err := storage.Alloc(name)

		if err != nil {
			t.Fatalf("alloc blocks(%d) error %s", i, err)
		}

		err = storage.Free(block)

		if err != nil {
			t.Fatalf("freee blocks(%d) error %s", i, err)
		}
	}
}

func BenchmarkPush(t *testing.B) {

	t.StopTimer()

	storage, err := openStorage(storageFile, 128)

	if err != nil {
		t.Fatal(err)
	}

	fifo, err := storage.Alloc("yayanyang")

	t.StartTimer()

	for i := 0; i < t.N; i++ {

		msg := gsim.NewMessage()

		msg.Content = make([]byte, 1024*32)

		fifo.Push(msg)

		if i%128 == 0 {
			err = storage.RunOnce(db)

			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
