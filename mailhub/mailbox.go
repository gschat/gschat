package mailhub

import (
	"github.com/gschat/gschat"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsrpc/gorpc"
)

// user's mailbox
type _MailBox struct {
	gslogger.Log                     // mixin gslogger
	mailhub      *_MailHub           // mailhub belongs to
	username     string              // mailbox belongs to
	clients      map[string]*_Client // register agents
	recvID       uint32              // receive sequence id
}

func (mailhub *_MailHub) newMailBox(username string) *_MailBox {
	return &_MailBox{
		Log:      gslogger.Get("mailbox"),
		mailhub:  mailhub,
		username: username,
		clients:  make(map[string]*_Client),
	}
}

func (mailbox *_MailBox) addAgent(newagent gsagent.Agent) {

	var agents []gsagent.Agent

	for _, client := range mailbox.clients {
		agents = append(agents, client.agent)
	}

	if old, ok := mailbox.clients[newagent.ID().String()]; ok {
		old.Close()
	}

	mailbox.clients[newagent.ID().String()] = mailbox.newClient(newagent)

	go func() {
		for _, agent := range agents {
			err := gschat.BindClient(uint16(gschat.ServiceClient), agent).DeviceStateChanged(nil, newagent.ID(), true)

			if err != nil {
				mailbox.E("call %s DeviceStateChanged error :%s", agent.ID(), err)
			}
		}
	}()
}

func (mailbox *_MailBox) removeAgent(device *gorpc.Device) {

	if client, ok := mailbox.clients[device.String()]; ok {
		client.Close()
		delete(mailbox.clients, device.String())
	}

	var agents []gsagent.Agent

	for _, client := range mailbox.clients {
		agents = append(agents, client.agent)
	}

	go func() {
		for _, agent := range agents {
			err := gschat.BindClient(uint16(gschat.ServiceClient), agent).DeviceStateChanged(nil, device, false)

			if err != nil {
				mailbox.E("call %s DeviceStateChanged error :%s", agent.ID(), err)
			}
		}
	}()
}

func (mailbox *_MailBox) mail(id uint32) (*gschat.Mail, error) {
	return mailbox.mailhub.storage.Query(mailbox.username, id)
}

func (mailbox *_MailBox) notify(id uint32) {
	for _, client := range mailbox.clients {
		client.notifyClient(id)
	}
}

func (mailbox *_MailBox) receivedID() (uint32, error) {
	return mailbox.mailhub.storage.SEQID(mailbox.username)
}

func (mailbox *_MailBox) sync(client gschat.Client, offset uint32, count uint32) (*_Sync, error) {

	maxid, err := mailbox.mailhub.storage.SEQID(mailbox.username)

	if err != nil {
		return nil, err
	}

	if offset > maxid {
		return nil, gschat.NewResourceNotFound()
	}

	maxnum := gsconfig.Uint32("gschat.mailhub.sync.maxnum", 1024)

	if count > maxnum {
		count = maxnum
	}

	return mailbox.newSync(client, offset, count), nil
}
