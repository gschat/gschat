package mailhub

import (
	"sync"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsrpc/gorpc"
)

type _MailHub struct {
	gslogger.Log                        // mixin gslogger
	sync.RWMutex                        // mixin read/write locker
	namedServices []*gorpc.NamedService // transproxy services
	users         map[string]*_MailBox  // user bound mailboxes
	agents        map[string]*_MailBox  // agent bound mailbox
}

// New create new mail hub
func New(name string) gsagent.System {
	return &_MailHub{
		Log:    gslogger.Get("mailhub"),
		users:  make(map[string]*_MailBox),
		agents: make(map[string]*_MailBox),
		namedServices: []*gorpc.NamedService{
			&gorpc.NamedService{
				Name:       gschat.NameOfMailHub,
				DispatchID: uint16(gschat.ServiceMailHub),
				VNodes:     gsconfig.Uint32("gschat.mailhub.vnodes", 4),
				NodeName:   name,
			},
		},
	}
}

func (mailhub *_MailHub) Register(context gsagent.Context) error {
	return nil
}

func (mailhub *_MailHub) Unregister(context gsagent.Context) {
}

func (mailhub *_MailHub) BindAgent(agent gsagent.Agent) error {
	mailhub.RLock()
	defer mailhub.RUnlock()

	if mailbox, ok := mailhub.agents[agent.ID().String()]; ok {
		mailbox.addAgent(agent)
		return nil
	}

	return gserrors.Newf(gschat.NewResourceNotFound(), "bind agent error :binder of user ==> device not found")
}

func (mailhub *_MailHub) UnbindAgent(agent gsagent.Agent) {

}

func (mailhub *_MailHub) AgentServices() []*gorpc.NamedService {
	return mailhub.namedServices
}

func (mailhub *_MailHub) AddTunnel(name string, pipeline gorpc.Pipeline) {
	pipeline.AddService(gschat.MakeUserBinder(uint16(gschat.ServiceUserBinder), mailhub))
}

func (mailhub *_MailHub) RemoveTunnel(name string, pipeline gorpc.Pipeline) {

}

func (mailhub *_MailHub) BindUser(callSite *gorpc.CallSite, userid string, device *gorpc.Device) (err error) {
	mailhub.Lock()
	defer mailhub.Unlock()

	agentID := device.String()

	mailbox, ok := mailhub.users[userid]

	if !ok {
		mailbox = mailhub.newMailBox(userid)
		mailhub.users[userid] = mailbox
	}

	if mailbox, ok := mailhub.agents[agentID]; ok {
		if mailbox.username != userid {
			mailbox.removeAgent(device)
		}
	}

	mailhub.agents[agentID] = mailbox

	return nil
}

func (mailhub *_MailHub) UnbindUser(callSite *gorpc.CallSite, userid string, device *gorpc.Device) (err error) {

	mailhub.Lock()
	defer mailhub.Unlock()

	agentID := device.String()

	mailbox, ok := mailhub.users[userid]

	if ok {
		mailbox.removeAgent(device)
	}

	if target, ok := mailhub.agents[agentID]; ok && target == mailbox {
		delete(mailhub.agents, agentID)
	}

	return nil
}
