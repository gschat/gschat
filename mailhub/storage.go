package mailhub

import (
	"github.com/gschat/gschat"
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
