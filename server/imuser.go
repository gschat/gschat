package server

import (
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mq"
	"github.com/gsdocker/gsagent"
)

type _IMUser struct {
	name string  // usernmae
	fifo mq.FIFO // user mq
}

func (server *_IMServer) newUser(name string) (*_IMUser, error) {

	fifo, err := server.fsqueue.Open(name)

	if err != nil {
		return nil, err
	}

	return &_IMUser{
		name: name,
		fifo: fifo,
	}, nil
}

func (user *_IMUser) put(mail *gschat.Mail) (retval uint64, err error) {

	now := time.Now()

	mail.TS = uint64(now.Unix())*1000000000 + uint64(now.Nanosecond())

	user.fifo.Push(mail)

	return mail.TS, nil
}

func (user *_IMUser) createAgentQ(context gsagent.Context) *_IMAgentQ {
	return newAgentQ(user.name, user.fifo, context)
}
