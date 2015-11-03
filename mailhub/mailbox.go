package mailhub

import (
	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsrpc/gorpc"
)

// user's mailbox
type _MailBox struct {
	gslogger.Log                     // mixin gslogger
	username     string              // mailbox belongs to
	clients      map[string]*_Client // register agents
}

func (mailhub *_MailHub) newMailBox(username string) *_MailBox {
	return &_MailBox{
		Log:      gslogger.Get("mailbox"),
		username: username,
		clients:  make(map[string]*_Client),
	}
}

func (mailbox *_MailBox) addAgent(newagent gsagent.Agent) {

	var agents []gsagent.Agent

	for _, client := range mailbox.clients {
		agents = append(agents, client.agent)
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
	delete(mailbox.clients, device.String())

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
