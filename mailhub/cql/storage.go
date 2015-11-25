//Package cql the cassandra storage backend
package cql

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gschat/gocql"
	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mailhub"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

var cqlOfKeyspace = `
    CREATE KEYSPACE IF NOT EXISTS gschat WITH replication = {
    'class' : 'SimpleStrategy','replication_factor' : 1 }
`

var cqlOfSQID = `
    CREATE TABLE gschat.SQID(username varchar,id int,PRIMARY KEY (username))
`

var cqlOfMail = `
    CREATE TABLE gschat.MAIL(key varchar, mail BLOB,PRIMARY KEY (key))

`

var cqlOfQuerySQID = `
    SELECT id FROM gschat.SQID WHERE username = ?
`

var cqlOfUpdateSQID = `
    UPDATE gschat.SQID SET id = ? WHERE username = ?
`

var cqlOfInsertMail = `
    INSERT INTO gschat.MAIL(key,mail) VALUES(?,?)
`

var cqlOfQueryMail = `
    SELECT mail FROM gschat.MAIL WHERE key = ?
`

type _Cached struct {
	id      uint32 // SQID
	content []byte // cached mail content
}

type _Storage struct {
	gslogger.Log                          // mixin gslogger APIs
	sync.RWMutex                          // mixin read/write mutex
	config          *gocql.ClusterConfig  // cql cluster config
	session         *gocql.Session        // cql session
	cachedID        map[string]*uint32    // cached ids
	cachedMail      map[string][]*_Cached // cached mail
	split           int                   // flush split
	cqlOfKeyspace   string                // cql of create new keyspace
	cqlOfSQID       string                // cql of create SQID table
	cqlOfMail       string                // cql of create Mail table
	cqlOfQuerySQID  string                // cql of query SQID
	cqlOfUpdateSQID string                // cql of query SQID
	cqlOfInsertMail string                // cql of insert new mail
	cqlOfQueryMail  string                // cql of query mail by key
}

// New create cql storage backend
func New(clusters ...string) (mailhub.Storage, error) {

	storage := &_Storage{
		Log:             gslogger.Get("cql-storage"),
		config:          gocql.NewCluster(clusters...),
		cachedID:        make(map[string]*uint32),
		cachedMail:      make(map[string][]*_Cached),
		split:           gsconfig.Int("gschat.mailhub.cql.split", 1024),
		cqlOfKeyspace:   gsconfig.String("gschat.mailhub.cql.keyspaceCQL", cqlOfKeyspace),
		cqlOfSQID:       gsconfig.String("gschat.mailhub.cql.SQIDCQL", cqlOfSQID),
		cqlOfMail:       gsconfig.String("gschat.mailhub.cql.MailCQL", cqlOfMail),
		cqlOfQuerySQID:  gsconfig.String("gschat.mailhub.cql.QuerySQIDCQL", cqlOfQuerySQID),
		cqlOfUpdateSQID: gsconfig.String("gschat.mailhub.cql.UpdateSQIDCQL", cqlOfUpdateSQID),
		cqlOfInsertMail: gsconfig.String("gschat.mailhub.cql.InsertMailCQL", cqlOfInsertMail),
		cqlOfQueryMail:  gsconfig.String("gschat.mailhub.cql.QueryMailCQL", cqlOfQueryMail),
	}

	storage.config.ProtoVersion = gsconfig.Int("gschat.mailhub.cql.ProtoVersion", 4)
	storage.config.CQLVersion = gsconfig.String("gschat.mailhub.cql.CQLVersion", "3.3.1")
	storage.config.Timeout = gsconfig.Seconds("gschat.mailhub.cql.Timeout", 60)
	storage.config.Port = gsconfig.Int("gschat.mailhub.cassandra-port", 9042)

	if gsconfig.Bool("gschat.mailhub.cql.Compress", true) {
		storage.config.Compressor = &gocql.SnappyCompressor{}
	}

	if err := storage.createKeySpace(); err != nil {
		return nil, err
	}
	if err := storage.createTables(); err != nil {
		return nil, err
	}

	storage.config.Keyspace = gsconfig.String("gschat.mailhub.cql.Keyspace", "gschat")

	var err error

	storage.session, err = storage.config.CreateSession()

	if err != nil {
		return nil, gserrors.Newf(err, "create new cql session error")
	}

	go storage.flush()

	return storage, err
}

func (storage *_Storage) flush() {

	ticker := time.NewTicker(gsconfig.Seconds("gschat.mailhub.cql.flush.duration", 1))

	defer ticker.Stop()

	for _ = range ticker.C {
		storage.doflush()
	}

}

func (storage *_Storage) doflush() {
	storage.D("flush cached mail ...")

	storage.Lock()
	cached := storage.cachedMail
	storage.cachedMail = make(map[string][]*_Cached)
	storage.Unlock()

	if len(cached) == 0 {
		storage.D("flush cached mail -- finish")
		return
	}

	max := 0

	for _, usercached := range cached {
		max += len(usercached)
	}

	storage.D("flush cached mail(%d) ...", max)

	updateIDs := make(map[string]uint32)

	batch := storage.session.NewBatch(gocql.LoggedBatch)

	counter := 0

	for name, usercached := range cached {

		maxID := uint32(0)

		for _, mail := range usercached {

			if mail.id > maxID {
				updateIDs[name] = mail.id
				maxID = mail.id
			}

			batch.Query(storage.cqlOfInsertMail, fmt.Sprintf("%s%d", name, mail.id), mail.content)

			counter++

			if counter%storage.split == 0 {

				for name, id := range updateIDs {
					batch.Query(storage.cqlOfUpdateSQID, id, name)
				}

				if err := storage.session.ExecuteBatch(batch); err != nil {
					storage.E("batch execute error :%s", err)
				} else {
					storage.D("flush cached %d -- finish", counter)
				}

				updateIDs = make(map[string]uint32)

				batch = storage.session.NewBatch(gocql.LoggedBatch)

			}
		}
	}

	if counter%storage.split > 0 {

		for name, id := range updateIDs {
			batch.Query(storage.cqlOfUpdateSQID, id, name)
		}

		if err := storage.session.ExecuteBatch(batch); err != nil {
			storage.E("batch execute error :%s", err)
		} else {
			storage.D("flush cached %d -- finish", counter)
		}
	}

	storage.D("flush cached mail -- finish")
}

func (storage *_Storage) createKeySpace() error {

	cluster := storage.config

	cluster.Keyspace = "system"

	var err error

	session, err := cluster.CreateSession()

	if err != nil {
		return gserrors.Newf(err, "create sessione rror")
	}

	defer session.Close()

	if err := session.Query(storage.cqlOfKeyspace).Exec(); err != nil {
		return gserrors.Newf(err, "create keyspace error\n%s", storage.cqlOfKeyspace)
	}

	return nil
}

func (storage *_Storage) createTables() error {

	session, err := storage.config.CreateSession()

	if err != nil {
		return err
	}

	defer session.Close()

	// create SQID table
	if err := session.Query(storage.cqlOfSQID).Exec(); err != nil {
		gserrors.Newf(err, "create SQID_TABLE error")
	}
	// create mail table
	if err := session.Query(storage.cqlOfMail).Exec(); err != nil {
		gserrors.Newf(err, "create SQID_TABLE error")
	}

	return nil
}

func (storage *_Storage) Save(username string, mail *gschat.Mail) (uint32, error) {

	id, err := storage.seqID(username)

	if err != nil {
		return 0, err
	}

	mail.SQID = atomic.AddUint32(id, 1)

	var buff bytes.Buffer

	err = gschat.WriteMail(&buff, mail)

	if err != nil {
		return 0, gserrors.Newf(err, "[cql storage] marshal mail error")
	}

	cached := &_Cached{
		id:      mail.SQID,
		content: buff.Bytes(),
	}
	storage.Lock()
	storage.cachedMail[username] = append(storage.cachedMail[username], cached)
	storage.Unlock()

	return mail.SQID, nil
}
func (storage *_Storage) Query(username string, id uint32) (*gschat.Mail, error) {

	var content []byte

	if err := storage.session.Query(
		storage.cqlOfQueryMail,
		fmt.Sprintf("%s%d", username, id)).Scan(&content); err != nil {

		return nil, err
	}

	mail, err := gschat.ReadMail(bytes.NewBuffer(content))

	if err != nil {
		return nil, gserrors.Newf(err, "[cql storage] unmarshal mail(%s:%d) error", username, id)
	}

	return mail, nil
}

func (storage *_Storage) seqID(username string) (*uint32, error) {
	storage.RLock()
	id, ok := storage.cachedID[username]
	storage.RUnlock()

	if !ok {

		storage.V("query %s SQID", username)

		var remoteID uint32
		if err := storage.session.Query(storage.cqlOfQuerySQID, username).Scan(&remoteID); err != nil {

			if err != gocql.ErrNotFound {
				return id, gserrors.Newf(err, "[cql storage] query %s SQID error", username)
			}

			remoteID = 0
		}

		storage.V("query %s SQID -- success", username)

		storage.Lock()
		id, ok = storage.cachedID[username]

		if !ok {
			id = &remoteID

			storage.cachedID[username] = id
		}

		storage.Unlock()

	}

	return id, nil
}

func (storage *_Storage) SEQID(username string) (uint32, error) {

	id, err := storage.seqID(username)

	if err != nil {
		return 0, err
	}

	return atomic.LoadUint32(id), nil
}
