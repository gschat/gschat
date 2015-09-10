package server

import (
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mq"
)

type _IMUser struct {
	name   string                   // usernmae
	fifo   mq.FIFO                  // user mq
	agents map[*_IMAgent]*_IMAgentQ // bind agents
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

func (user *_IMUser) bind(agent *_IMAgent) *_IMAgentQ {

	agentQ, ok := user.agents[agent]

	if !ok {
		agentQ = newAgentQ(user.name, user.fifo, agent)
	}

	return agentQ
}

func (user *_IMUser) unbind(agent *_IMAgent) {
	delete(user.agents, agent)
}
