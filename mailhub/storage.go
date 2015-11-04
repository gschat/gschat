package mailhub

import (
	"github.com/gschat/gschat"
)

// Storage the mail storage
type Storage interface {
	// Save save the mail into offline received mailbox
	// if success return the mail received SEQ id
	Save(username string, mail *gschat.Mail) (uint32, error)

	// Query query received mails by username and start received SEQ id
	// the length parameter indicate expect received mails
	Query(username string, startID uint32, length int) ([]*gschat.Mail, error)

	// query the user's mailbox last message id
	SEQID(username string) (uint32, error)
}
