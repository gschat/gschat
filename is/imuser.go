package is

import (
	"bytes"
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/tsdb"
	"github.com/gsdocker/gsagent"
)

type _IMUser struct {
	name       string          // usernmae
	datasource tsdb.DataSource // user mq
}

func (server *_IMServer) newUser(name string) (*_IMUser, error) {

	return &_IMUser{
		name:       name,
		datasource: server.datasource,
	}, nil
}

func (user *_IMUser) put(mail *gschat.Mail) (retval uint64, err error) {

	now := time.Now()

	mail.TS = uint64(now.Unix())*1000000000 + uint64(now.Nanosecond())

	var buff bytes.Buffer

	err = gschat.WriteMail(&buff, mail)

	if err != nil {
		return 0, err
	}

	return mail.TS, user.datasource.Update(user.name, buff.Bytes())
}

func (user *_IMUser) createAgentQ(context gsagent.Context) *_IMAgentQ {
	return newAgentQ(user.name, user.datasource, context)
}
