package mailhub

import (
	"fmt"
	"sync"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/hashring"
	"github.com/gsrpc/gorpc/snowflake"
)

type _MailHub struct {
	gslogger.Log                                 // mixin gslogger
	userMutex     sync.RWMutex                   // mixin read/write locker
	resolverMutex sync.RWMutex                   // mixin read/write locker
	userResolvers *hashring.HashRing             // user resolver services
	groups        map[string][]string            // groups
	blockRules    map[string][]*gschat.BlockRule // block rules
	users         map[string]*_MailBox           // user bound mailboxes
	agents        map[string]*_MailBox           // agent bound mailbox
	namedServices []*gorpc.NamedService          // transproxy services
	storage       Storage                        // offline message storage
	snowflake     *snowflake.SnowFlake           // id generate
}

// New create new mail hub
func New(name string, storage Storage) gsagent.System {
	return &_MailHub{
		Log:           gslogger.Get("mailhub"),
		users:         make(map[string]*_MailBox),
		agents:        make(map[string]*_MailBox),
		groups:        make(map[string][]string),
		blockRules:    make(map[string][]*gschat.BlockRule),
		userResolvers: hashring.New(),
		storage:       storage,
		namedServices: []*gorpc.NamedService{
			&gorpc.NamedService{
				Name:       gschat.NameOfMailHub,
				DispatchID: uint16(gschat.ServiceMailHub),
				VNodes:     gsconfig.Uint32("gschat.mailhub.vnodes", 4),
				NodeName:   name,
			},
		},
		snowflake: snowflake.New(gsconfig.Uint32("gschat.mailhub.id", 0), gsconfig.Uint32("gschat.mailhub.snowflake.cached", 1)),
	}
}

func (mailhub *_MailHub) AddUserResolver(name *gorpc.NamedService, userResolver gschat.UserResolver) {
	mailhub.resolverMutex.Lock()
	defer mailhub.resolverMutex.Unlock()

	resolver := mailhub.newUserResolver(userResolver)

	for i := uint32(0); i < name.VNodes; i++ {

		key := fmt.Sprintf("%s:%d", name.NodeName, i)

		mailhub.userResolvers.Remove(key)

		mailhub.userResolvers.Put(key, resolver)
	}
}

func (mailhub *_MailHub) RemoveUserResolver(name *gorpc.NamedService) {

	mailhub.resolverMutex.Lock()
	defer mailhub.resolverMutex.Unlock()

	for i := uint32(0); i < name.VNodes; i++ {

		key := fmt.Sprintf("%s:%d", name.NodeName, i)

		mailhub.userResolvers.Remove(key)
	}
}

func (mailhub *_MailHub) Register(context gsagent.Context) error {
	return nil
}

func (mailhub *_MailHub) Unregister(context gsagent.Context) {
}

func (mailhub *_MailHub) BindAgent(agent gsagent.Agent) error {
	mailhub.userMutex.RLock()
	defer mailhub.userMutex.RUnlock()

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
	mailhub.userMutex.Lock()
	defer mailhub.userMutex.Unlock()

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

	mailhub.userMutex.Lock()
	defer mailhub.userMutex.Unlock()

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

func (mailhub *_MailHub) queryGroup(id string) ([]string, bool) {
	return nil, false
}

func (mailhub *_MailHub) queryBlockRule(id string) ([]*gschat.BlockRule, bool) {
	return nil, false
}

func (mailhub *_MailHub) dispatchMail(mail *gschat.Mail) (uint64, error) {

	if mail.Type == gschat.MailTypeMulti {
		group, ok := mailhub.queryGroup(mail.Receiver)

		if !ok {
			return 0, gschat.NewUserNotFound()
		}

		mail.ID = <-mailhub.snowflake.Gen

		for _, username := range group {
			mailhub.storage.Save(username, mail)
			//TODO: update mailbox
		}
	}

	return 0, nil
}
