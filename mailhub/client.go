package mailhub

import (
	"github.com/gschat/gschat"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsrpc/gorpc"
)

type _Client struct {
	api      gschat.Client // client rpc interface
	mailbox  *_MailBox     // mailbox the client own
	agent    gsagent.Agent // client'a agent object
	sendID   uint32        // the send sequence id
	syncFlag uint32        // the sync flag
}

func (mailbox *_MailBox) newClient(agent gsagent.Agent) *_Client {
	wrapper := &_Client{
		agent:   agent,
		mailbox: mailbox,
	}

	// create client proxy interface
	wrapper.api = gschat.BindClient(uint16(gschat.ServiceClient), agent)

	// register mailhub interface
	agent.AddService(gschat.MakeMailHub(uint16(gschat.ServiceMailHub), wrapper))

	return wrapper
}

func (client *_Client) PutSync(callSite *gorpc.CallSite) (retval uint32, err error) {
	return client.sendID, nil
}

func (client *_Client) Put(callSite *gorpc.CallSite, mail *gschat.Mail) (retval uint64, err error) {
	return 0, nil
}

func (client *_Client) Sync(callSite *gorpc.CallSite, offset uint32, count uint32) (retval uint32, err error) {
	return 0, nil
}
