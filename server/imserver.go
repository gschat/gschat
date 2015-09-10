package server

import (
	"path/filepath"
	"sync"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mq"
	"github.com/gsdocker/gsactor"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// implement gsactor.AgentSystem
type _IMServer struct {
	sync.RWMutex                            // Mixin read write mutex
	gslogger.Log                            // Mixin logger
	context      gsactor.AgentSystemContext // agent system context
	namedService gschat.NamedService        // services
	users        map[string]*_IMUser        // register users
	devices      map[string]*_IMAgent       // device agents
	binders      map[*_IMAgent]*_IMAgentQ   // agent and user binder
	fsqueue      *mq.MQ                     //
}

// NewIMServer create new im server
func NewIMServer(vnodes uint32) gsactor.AgentSystem {

	currentDir, _ := filepath.Abs("./")

	fsqueue, err := mq.NewMQ(gsconfig.String("imserver.mq.dir", currentDir))

	if err != nil {
		gserrors.Panicf(err, "create new MQ error")
	}

	return &_IMServer{
		Log: gslogger.Get("imserver"),
		namedService: gschat.NamedService{
			VNodes: vnodes,
			Type:   gschat.ServiceTypeIM,
		},
		users:   make(map[string]*_IMUser),
		devices: make(map[string]*_IMAgent),
		binders: make(map[*_IMAgent]*_IMAgentQ),
		fsqueue: fsqueue,
	}
}

func (server *_IMServer) Open(context gsactor.AgentSystemContext) error {

	server.D("open")

	server.context = context

	server.namedService.Name = context.Name()

	context.Register(gschat.MakeService(uint16(gschat.ServiceTypeUnknown), server))

	context.Register(gschat.MakeIManager(uint16(gschat.ServiceTypeIM), server))

	return nil
}

func (server *_IMServer) Bind(username string, device *gorpc.Device) (err error) {

	server.Lock()
	defer server.Unlock()

	user, ok := server.users[username]

	if !ok {

		server.D("create user %s", username)

		user, err = server.newUser(username)

		if err != nil {
			server.D("create user %s -- failed\n%s", username, err)
			return err
		}

		server.D("create user %s -- success", username)

		server.users[username] = user
	}

	agent, ok := server.devices[device.String()]

	if !ok {
		agent = server.newAgent(device)

		server.devices[device.String()] = agent
	}

	agentQ := user.bind(agent)

	server.binders[agent] = agentQ

	return nil
}

func (server *_IMServer) Unbind(username string, device *gorpc.Device) error {

	server.Lock()
	defer server.Unlock()

	user, ok := server.users[username]

	if !ok {
		return nil
	}

	agent, ok := server.devices[device.String()]

	if !ok {
		return nil
	}

	user.unbind(agent)

	delete(server.binders, agent)

	return nil
}

func (server *_IMServer) agentQ(agent *_IMAgent) (agentQ *_IMAgentQ, ok bool) {
	server.RLock()
	defer server.RUnlock()

	agentQ, ok = server.binders[agent]

	return
}

func (server *_IMServer) user(username string) (user *_IMUser, ok bool) {
	server.RLock()
	defer server.RUnlock()

	user, ok = server.users[username]

	return
}

func (server *_IMServer) Name() (*gschat.NamedService, error) {
	return &server.namedService, nil
}

func (server *_IMServer) Close() {
	server.D("closed")
}

func (server *_IMServer) removeDevice(key string, agent *_IMAgent) {
	server.Lock()
	defer server.Unlock()

	if old, ok := server.devices[key]; ok && old == agent {
		delete(server.devices, key)
	}
}
