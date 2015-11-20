package mailhub

import (
	"sync"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gslogger"
)

// Storage the mail storage
type Storage interface {
	// Save save the mail into offline received mailbox
	// if success return the mail received SEQ id
	Save(username string, mail *gschat.Mail) (uint32, error)

	// Query query received mails by username and received id
	Query(username string, id uint32) (*gschat.Mail, error)

	// query the user's mailbox last message id
	SEQID(username string) (uint32, error)
}

type _UserCached struct {
	gslogger.Log                // mixin gslogger APIs
	sync.RWMutex                // mixin read/write mutex
	indexer      map[uint32]int // mail
	ringbuff     []*gschat.Mail // ring buffer
	header       int            // ring header
	tail         int            // ring tail
}

func newUserCached(size int) *_UserCached {
	return &_UserCached{
		Log:      gslogger.Get("cached-storage"),
		indexer:  make(map[uint32]int),
		ringbuff: make([]*gschat.Mail, size),
	}
}

func (userCached *_UserCached) Get(id uint32) (*gschat.Mail, bool) {

	userCached.RLock()
	if indexer, ok := userCached.indexer[id]; ok {
		userCached.RUnlock()
		return userCached.ringbuff[indexer], true
	}
	userCached.RUnlock()
	return nil, false
}

func (userCached *_UserCached) Update(val *gschat.Mail) {

	userCached.Lock()

	old := userCached.ringbuff[userCached.tail]

	if old != nil {
		delete(userCached.indexer, old.SQID)
	}

	userCached.ringbuff[userCached.tail] = val
	userCached.indexer[val.SQID] = userCached.tail

	userCached.tail++

	if userCached.tail == len(userCached.ringbuff) {
		userCached.tail = 0
	}

	if userCached.tail == userCached.header {
		userCached.header++

		if userCached.header == len(userCached.ringbuff) {
			userCached.header = 0
		}
	}

	userCached.Unlock()
}

// The memory userCached implement Storage interface
type _MemoryCached struct {
	sync.RWMutex                         // sequence id
	cached       map[string]*_UserCached //user cached
	seqIDs       map[string]uint32       // user cached sqIDs
	storage      Storage                 // storage implement
}

func newMemoryCached(storage Storage) *_MemoryCached {
	return &_MemoryCached{
		cached:  make(map[string]*_UserCached),
		seqIDs:  make(map[string]uint32),
		storage: storage,
	}
}

func (memoryCached *_MemoryCached) Save(username string, mail *gschat.Mail) (id uint32, err error) {

	if memoryCached.storage != nil {

		id, err = memoryCached.storage.Save(username, mail)
		if err != nil {
			return
		}

	} else {
		memoryCached.Lock()
		id = memoryCached.seqIDs[username]

		id++

		memoryCached.seqIDs[username] = id
		memoryCached.Unlock()
	}

	mail.SQID = id

	memoryCached.RLock()
	usercached, ok := memoryCached.cached[username]
	memoryCached.RUnlock()

	if !ok {
		usercached = newUserCached(gsconfig.Int("gschat.mailhub.memorycached", 1024))

		memoryCached.Lock()
		memoryCached.cached[username] = usercached
		memoryCached.Unlock()
	}

	usercached.Update(mail)

	return
}

func (memoryCached *_MemoryCached) queryCached(username string, id uint32) (*gschat.Mail, bool) {
	memoryCached.RLock()
	usercached, ok := memoryCached.cached[username]
	memoryCached.RUnlock()

	if ok {
		return usercached.Get(id)
	}

	return nil, false
}

func (memoryCached *_MemoryCached) Query(username string, id uint32) (*gschat.Mail, error) {
	if mail, ok := memoryCached.queryCached(username, id); ok {

		return mail, nil
	}

	if memoryCached.storage != nil {
		mail, err := memoryCached.storage.Query(username, id)

		if err != nil {
			return mail, err
		}

		memoryCached.RLock()
		usercached, ok := memoryCached.cached[username]
		memoryCached.RUnlock()

		if !ok {
			usercached = newUserCached(gsconfig.Int("gschat.mailhub.memorycached", 1024))
			memoryCached.Lock()
			memoryCached.cached[username] = usercached
			memoryCached.Unlock()
		}

		usercached.Update(mail)

		return mail, nil
	}

	return nil, gschat.NewResourceNotFound()
}

func (memoryCached *_MemoryCached) SEQID(username string) (uint32, error) {

	if memoryCached.storage != nil {

		return memoryCached.storage.SEQID(username)
	}

	memoryCached.RLock()
	id, ok := memoryCached.seqIDs[username]
	memoryCached.RUnlock()

	if ok {

		return id, nil
	}

	id = uint32(1)

	memoryCached.Lock()
	memoryCached.seqIDs[username] = id
	memoryCached.Unlock()

	return id, nil
}
