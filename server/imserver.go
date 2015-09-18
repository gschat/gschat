package server

import (
	"path/filepath"
	"sync"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mq"
	"github.com/gsdocker/gsagent"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// implement gsagent.System
type _IMServer struct {
	sync.RWMutex                       // Mixin read write mutex
	gslogger.Log                       // Mixin logger
	context      gsagent.SystemContext // agent system context
	namedService gschat.NamedService   // services
	users        map[string]*_IMUser   // register users
	binders      map[string]*_IMUser   // agent and user binder
	agents       map[string]*_IMAgent  // agents
	fsqueue      *mq.MQ                //
}

// NewIMServer create new im server
func NewIMServer(vnodes uint32) gsagent.System {

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
		binders: make(map[string]*_IMUser),
		agents:  make(map[string]*_IMAgent),
		fsqueue: fsqueue,
	}
}

func (server *_IMServer) Register(context gsagent.SystemContext) error {

	server.D("open")

	server.context = context

	server.namedService.Name = context.Name()

	return nil
}

func (server *_IMServer) Unregister(context gsagent.SystemContext) {
}

func (server *_IMServer) AddTunnel(name string, pipeline gorpc.Pipeline) {

	pipeline.AddService(gschat.MakeService(uint16(gschat.ServiceTypeUnknown), server))

	pipeline.AddService(gschat.MakeIManager(uint16(gschat.ServiceTypeIM), server))
}

func (server *_IMServer) RemoveTunnel(name string, pipeline gorpc.Pipeline) {

}

func (server *_IMServer) Bind(username string, device *gorpc.Device) (err error) {

	server.Lock()
	defer server.Unlock()

	server.I("bind user %s to %s", username, device)

	user, ok := server.users[username]

	if !ok {
		var err error

		user, err = server.newUser(username)

		if err != nil {
			return err
		}

		server.users[username] = user

		server.I("create new user %s", username)
	}

	if _, ok := server.binders[device.String()]; !ok {

		server.binders[device.String()] = user
	}

	server.I("bind user %s to %s -- success", username, device)

	return nil
}

func (server *_IMServer) Unbind(username string, device *gorpc.Device) error {

	server.D("unbind .....................")

	server.Lock()
	defer server.Unlock()

	if user, ok := server.binders[device.String()]; ok && user.name == username {
		delete(server.binders, device.String())
	}

	if client, ok := server.agents[device.String()]; ok {
		client.close()
		delete(server.agents, device.String())
	}

	server.D("unbind ..................... -- success")

	return nil
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
