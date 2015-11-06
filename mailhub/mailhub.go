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
		storage:       newMemoryCached(storage),
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

func (mailhub *_MailHub) BindUser(callSite *gorpc.CallSite, userid string, device *gorpc.Device) error {
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

	mailhub.I("bind user %s with device %s", userid, device)

	return nil
}

func (mailhub *_MailHub) UnbindUser(callSite *gorpc.CallSite, userid string, device *gorpc.Device) (err error) {

	mailhub.userMutex.Lock()
	defer mailhub.userMutex.Unlock()

	mailhub.I("ubind user %s with device %s", userid, device)

	agentID := device.String()

	mailbox, ok := mailhub.users[userid]

	if ok {
		mailbox.removeAgent(device)
	}

	if target, ok := mailhub.agents[agentID]; ok && target == mailbox {
		delete(mailhub.agents, agentID)

		mailhub.I("ubind user %s with device %s -- success", userid, device)

	}

	return nil
}

func (mailhub *_MailHub) user(username string) (*_MailBox, bool) {

	mailhub.userMutex.RLock()
	defer mailhub.userMutex.RUnlock()

	mailbox, ok := mailhub.users[username]

	return mailbox, ok
}

func (mailhub *_MailHub) queryGroup(id string) ([]string, bool) {
	return nil, false
}

func (mailhub *_MailHub) queryBlockRule(id string) []*gschat.BlockRule {
	return nil
}

func (mailhub *_MailHub) dispatchMail(device *gorpc.Device, mail *gschat.Mail) (uint64, error) {

	mail.ID = <-mailhub.snowflake.Gen

	mailhub.D("dispatch mail(%d:%s -> %s)", mail.ID, mail.Sender, mail.Receiver)

	if mail.Type == gschat.MailTypeMulti {
		group, ok := mailhub.queryGroup(mail.Receiver)

		if !ok {
			return 0, gschat.NewUserNotFound()
		}

		for _, username := range group {
			mailhub.receivedMail(username, mail)
		}
	} else if mail.Type == gschat.MailTypeSingle {
		mailhub.receivedMail(mail.Receiver, mail)
	} else {
		mailhub.W("unsupport mail type '%s' from %s", mail.Type, device)
	}

	return 0, nil
}

func (mailhub *_MailHub) receivedMail(username string, mail *gschat.Mail) {

	blockRules := mailhub.queryBlockRule(username)

	for _, rule := range blockRules {
		if rule.Target == mail.Sender {

			mailhub.I("block receive mail(%d:%s -> %s)", mail.ID, mail.Sender, mail.Receiver)

			return
		}
	}

	id, err := mailhub.storage.Save(username, mail)

	if err != nil {
		mailhub.W("save user %s mail error :%s", username, err)
		return
	}

	mailhub.D("received mail(%d:%s -> %s)", mail.ID, mail.Sender, mail.Receiver)

	if mailbox, ok := mailhub.user(username); ok {
		mailhub.D("notify mail(%d:%s -> %s) received", mail.ID, mail.Sender, mail.Receiver)
		mailbox.notify(id)
	} else {
		mailhub.D("user %s offline , drop mail received notify", username)
	}
}
