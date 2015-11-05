package mailhub

import (
	"sync/atomic"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
)

type _Sync struct {
	gslogger.Log
	client  gschat.Client // sync client
	mailbox *_MailBox     // mailbox belongs to
	offset  uint32        // start offset
	count   uint32        // sync count
	closed  uint32        // close flag
}

func (mailbox *_MailBox) newSync(client gschat.Client, offset uint32, count uint32) *_Sync {
	sync := &_Sync{
		Log:    gslogger.Get("mailbox"),
		client: client,
		offset: offset,
		count:  count,
	}

	go sync.loop()

	return sync
}

func (sync *_Sync) loop() {

	max := sync.offset + sync.count

	for i := sync.offset; i < max; i++ {

		if !atomic.CompareAndSwapUint32(&sync.closed, 0, 0) {
			return
		}

		mail, err := sync.mailbox.mail(i)

		if err != nil {
			sync.W("%s query mail %d error :%s", sync.mailbox.username, i, err)
			continue
		}

		sync.client.Push(nil, mail)
	}
}

func (sync *_Sync) Close() {
	atomic.CompareAndSwapUint32(&sync.closed, 0, 1)
}
