package mailhub

import (
	"sync"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gsconfig"
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
	indexer  map[uint32]int // mail
	ringbuff []*gschat.Mail // ring buffer
	header   int            // ring header
	tail     int            // ring tail
}

func newUserCached(size int) *_UserCached {
	return &_UserCached{
		indexer:  make(map[uint32]int),
		ringbuff: make([]*gschat.Mail, size),
	}
}

func (userCached *_UserCached) Get(id uint32) (*gschat.Mail, bool) {
	if indexer, ok := userCached.indexer[id]; ok {
		return userCached.ringbuff[indexer], true
	}

	return nil, false
}

func (userCached *_UserCached) Update(val *gschat.Mail) {

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
	memoryCached.Lock()
	defer memoryCached.Unlock()

	if memoryCached.storage != nil {
		id, err = memoryCached.storage.Save(username, mail)
		if err != nil {
			return
		}

		memoryCached.seqIDs[username] = id
	} else {
		id = memoryCached.seqIDs[username]

		id++

		memoryCached.seqIDs[username] = id
	}

	mail.SQID = id

	usercached, ok := memoryCached.cached[username]

	if !ok {
		usercached = newUserCached(gsconfig.Int("gschat.mailhub.memorycached", 1024))

		memoryCached.cached[username] = usercached
	}

	usercached.Update(mail)

	return
}

func (memoryCached *_MemoryCached) queryCached(username string, id uint32) (*gschat.Mail, bool) {
	memoryCached.RLock()
	defer memoryCached.RUnlock()

	if usercached, ok := memoryCached.cached[username]; ok {
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

		memoryCached.Lock()
		defer memoryCached.Unlock()

		usercached, ok := memoryCached.cached[username]

		if !ok {
			usercached = newUserCached(gsconfig.Int("gschat.mailhub.memorycached", 1024))

			memoryCached.cached[username] = usercached
		}

		usercached.Update(mail)

		return mail, nil
	}

	return nil, gschat.NewResourceNotFound()
}

func (memoryCached *_MemoryCached) cachedSEQID(username string) (uint32, bool) {
	memoryCached.RLock()
	defer memoryCached.RUnlock()

	if id, ok := memoryCached.seqIDs[username]; ok {
		return id, true
	}

	return 0, false
}

func (memoryCached *_MemoryCached) SEQID(username string) (uint32, error) {
	if id, ok := memoryCached.cachedSEQID(username); ok {
		return id, nil
	}

	if memoryCached.storage != nil {

		id, err := memoryCached.storage.SEQID(username)

		if err != nil {
			return id, err
		}

		memoryCached.Lock()
		defer memoryCached.Unlock()

		memoryCached.seqIDs[username] = id

		return id, nil
	}

	id := uint32(1)

	memoryCached.seqIDs[username] = id

	return id, nil
}
