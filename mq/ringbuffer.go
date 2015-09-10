package mq

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sync"

	"github.com/gsdocker/gslogger"

	"github.com/gschat/gschat"
)

// Stream Push stream
type Stream struct {
	fifo   FIFO                // bound fifo
	cursor uint32              // stream cursor
	pushQ  chan *gschat.Mail   // pushQ
	Chan   <-chan *gschat.Mail // output stream
	Offset uint32              // the stream start offset
}

// Close .
func (stream *Stream) Close() {
	stream.fifo.CloseStream(stream)
}

// FIFO .
type FIFO interface {
	// get fifo metadata
	Meta() *QMeta
	// Max received sequence id
	MaxID() uint32
	// Min received sequence id
	MinID() uint32
	// Push message, return received id
	Push(msg *gschat.Mail)
	// create new message stream
	NewStream(startID uint32, cachedsize int) *Stream
	// close stream
	CloseStream(stream *Stream)
}

type _RingBuffer struct {
	gslogger.Log                    // Mixin Log
	sync.Mutex                      // Mixin locker
	QMeta                           // mixin metadata
	pushQ        chan *gschat.Mail  // push message Q
	ringbuff     []*gschat.Mail     // memory cached message
	ringHeader   int                // ringbuff cached header indexer
	ringTail     int                // ringbuff cached tail indexer
	indexer      map[uint32]int     // ringbuff message indexer
	streams      map[string]*Stream // streams
}

func (storage *_Storage) newRingBuffer(name string, id uint32, cachedsize uint32) *_RingBuffer {

	meta := &QMeta{
		Name:     name,
		FilePath: storage.filepath,
		ID:       id,
		Capacity: storage.blocksize / 2,
	}

	cachedMeta := NewQCachedMeta()

	cachedMeta.Offset = storage.headersize + id*storage.blocksize

	meta.DuoCached[0] = cachedMeta

	cachedMeta = NewQCachedMeta()

	cachedMeta.Offset = storage.headersize + id*storage.blocksize + meta.Capacity

	meta.DuoCached[1] = cachedMeta

	return storage.openRingBuffer(meta, cachedsize)
}

func (storage *_Storage) openRingBuffer(meta *QMeta, cachedsize uint32) *_RingBuffer {

	ringbuff := &_RingBuffer{
		Log:      gslogger.Get("mq.fifo"),
		QMeta:    *meta,
		pushQ:    make(chan *gschat.Mail, cachedsize),
		indexer:  make(map[uint32]int),
		ringbuff: make([]*gschat.Mail, cachedsize),
		streams:  make(map[string]*Stream),
	}

	return ringbuff
}

func (ringbuff *_RingBuffer) flush(file *os.File, headersize uint32) {

	ringbuff.Lock()
	defer ringbuff.Unlock()

	for {
		select {
		case msg := <-ringbuff.pushQ:
			ringbuff.flushMessage(file, headersize, msg)
			continue
		default:
		}

		// TODO: process read routine

		for _, stream := range ringbuff.streams {
			ringbuff.flushStream(file, stream)
		}

		return
	}
}

func (ringbuff *_RingBuffer) flushStream(file *os.File, stream *Stream) {
	meta := ringbuff.Meta()

	cached := meta.DuoCached[meta.CurrentCached]
	//
	precached := meta.DuoCached[(meta.CurrentCached+1)%2]
	//

	cursor := uint32(0)

	prevcursor := uint32(0)

	if stream.cursor < precached.MinSID {
		stream.cursor = precached.MinSID
	}

	for ; stream.cursor < cached.MaxSID; stream.cursor++ {

		slot, ok := ringbuff.indexer[stream.cursor]
		if ok {
			ringbuff.V("found cached message :%d", stream.cursor)
			select {
			case stream.pushQ <- ringbuff.ringbuff[slot]:
			default:
				return
			}

			continue
		}

		var msg *gschat.Mail

		if stream.cursor < precached.MaxSID {
			for {

				msg, prevcursor = ringbuff.readMessage(file, precached, prevcursor)

				if msg == nil {
					return
				}

				ringbuff.cachedMessage(msg)

				if msg.SQID == stream.cursor {
					ringbuff.V("read message(%p:%d) -- success", msg, msg.SQID)
					break
				}
			}
		} else {
			for {

				msg, cursor = ringbuff.readMessage(file, cached, cursor)

				if msg == nil {
					return
				}

				ringbuff.cachedMessage(msg)

				if msg.SQID == stream.cursor {
					break
				}
			}
		}

		select {
		case stream.pushQ <- msg:
		default:
			return
		}
	}
}

func (ringbuff *_RingBuffer) cachedMessage(msg *gschat.Mail) {
	old := ringbuff.ringbuff[ringbuff.ringTail]

	if old != nil {
		delete(ringbuff.indexer, old.SQID)
	}

	ringbuff.ringbuff[ringbuff.ringTail] = msg
	ringbuff.indexer[msg.SQID] = ringbuff.ringTail

	ringbuff.ringTail++

	if ringbuff.ringTail == len(ringbuff.ringbuff) {
		ringbuff.ringTail = 0
	}

	if ringbuff.ringTail == ringbuff.ringHeader {
		ringbuff.ringHeader++

		if ringbuff.ringHeader == len(ringbuff.ringbuff) {
			ringbuff.ringHeader = 0
		}
	}
}

func (ringbuff *_RingBuffer) readMessage(file *os.File, cached *QCachedMeta, cursor uint32) (*gschat.Mail, uint32) {

	if cursor >= cached.Cursor {
		return nil, cursor
	}

	sizebuff := [2]byte{}

	_, err := file.ReadAt(sizebuff[:], int64(cached.Offset+cursor))

	if err != nil {
		ringbuff.E("read gschat.Mail error:%s", err)
		return nil, cursor
	}

	size := binary.BigEndian.Uint16(sizebuff[:])

	buff := make([]byte, size-2)

	_, err = file.ReadAt(buff, int64(cached.Offset+cursor+2))

	if err != nil {
		ringbuff.E("read gschat.Mail error:%s", err)
		return nil, cursor
	}

	msg, err := gschat.ReadMail(bytes.NewBuffer(buff))

	if err != nil {
		ringbuff.E("unmarshal gschat.Mail error:%s", err)
		return nil, cursor
	}

	cursor += uint32(size)

	return msg, cursor
}

func (ringbuff *_RingBuffer) flushMessage(file *os.File, headersize uint32, msg *gschat.Mail) {

	meta := ringbuff.Meta()

	cached := meta.DuoCached[meta.CurrentCached]

	var buff bytes.Buffer

	msg.SQID = cached.MaxSID

	ringbuff.D("flush message(%d)", msg.SQID)

	err := gschat.WriteMail(&buff, msg)

	if err != nil {
		ringbuff.E("marshal gschat.Mail error:%s", err)
		return
	}

	size := uint16(buff.Len() + 2)

	if uint32(size) > meta.Capacity {
		ringbuff.W("drop too long message (%d > %d)", size, meta.Capacity)
		return
	}

	sizebuff := [2]byte{}

	binary.BigEndian.PutUint16(sizebuff[:], size)

	if cached.Cursor+uint32(size) > meta.Capacity {
		// replace cache buff
		meta.CurrentCached = (meta.CurrentCached + 1) % 2

		nextcached := meta.DuoCached[meta.CurrentCached]

		nextcached.Cursor = 0

		nextcached.MaxSID = cached.MaxSID

		nextcached.MinSID = cached.MaxSID

		ringbuff.V("swap duocached :(%p)[%d,%d) -> (%p)[%d,%d)", cached, cached.MinSID, cached.MaxSID, nextcached, nextcached.MinSID, nextcached.MaxSID)

		cached = nextcached
	}

	_, err = file.WriteAt(sizebuff[:], int64(cached.Offset+cached.Cursor))

	if err != nil {
		ringbuff.E("write gschat.Mail error:%s", err)
		return
	}

	_, err = file.WriteAt(buff.Bytes(), int64(cached.Offset+cached.Cursor+2))

	if err != nil {
		ringbuff.E("write gschat.Mail error:%s", err)
		return
	}
	// now write metadata
	cached.Cursor += uint32(size)
	cached.MaxSID++

	ringbuff.D("flush message(%d) -- success", msg.SQID)
}

// Meta .
func (ringbuff *_RingBuffer) Meta() *QMeta {
	return &ringbuff.QMeta
}

func (ringbuff *_RingBuffer) MaxID() uint32 {
	return ringbuff.DuoCached[ringbuff.CurrentCached].MaxSID
}

func (ringbuff *_RingBuffer) MinID() uint32 {
	return ringbuff.DuoCached[(ringbuff.CurrentCached+1)%2].MinSID
}

func (ringbuff *_RingBuffer) Push(msg *gschat.Mail) {

	ringbuff.pushQ <- msg
}

func (ringbuff *_RingBuffer) NewStream(startID uint32, cachesize int) *Stream {

	ringbuff.Lock()
	defer ringbuff.Unlock()

	stream := &Stream{
		fifo:   ringbuff,
		cursor: startID,
		Offset: startID,
		pushQ:  make(chan *gschat.Mail, cachesize),
	}

	stream.Chan = stream.pushQ

	ringbuff.streams[fmt.Sprintf("%p", stream)] = stream

	return stream
}

func (ringbuff *_RingBuffer) CloseStream(stream *Stream) {

	ringbuff.Lock()
	defer ringbuff.Unlock()

	delete(ringbuff.streams, fmt.Sprintf("%p", stream))

	close(stream.pushQ)
}
