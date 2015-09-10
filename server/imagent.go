package server

import (
	"github.com/gschat/gschat"
	"github.com/gsdocker/gsactor"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// IM agent device agent
type _IMAgent struct {
	gslogger.Log                          // Mixin log APIs
	gsactor.AgentContext                  // agent context
	server               *_IMServer       // server
	dispatcher           gorpc.Dispatcher // serivce dispatcher
	device               *gorpc.Device    // device
}

func (server *_IMServer) newAgent(device *gorpc.Device) *_IMAgent {
	return &_IMAgent{
		Log:    gslogger.Get("im-agent"),
		server: server,
	}
}

func (server *_IMServer) Agent(context gsactor.AgentContext) (gsactor.Agent, error) {

	server.Lock()
	defer server.Unlock()

	agent, ok := server.devices[context.ID().String()]

	if !ok {
		agent = server.newAgent(context.ID())

		server.devices[context.ID().String()] = agent
	}

	agent.AgentContext = context

	agent.dispatcher = gschat.MakeIMServer(uint16(gschat.ServiceTypeIM), agent)

	return agent, nil
}

func (agent *_IMAgent) ID() *gorpc.Device {
	return agent.device
}

func (agent *_IMAgent) Put(mail *gschat.Mail) (retval uint64, err error) {

	user, ok := agent.server.user(mail.Receiver)

	if !ok {
		agent.W("can't dispatch message from %s to %s -- receiver not found", mail.Sender, mail.Receiver)
		return 0, gschat.NewUserNotFound()
	}

	return user.put(mail)
}

func (agent *_IMAgent) Pull(offset uint32) (err error) {

	agentQ, ok := agent.server.agentQ(agent)

	if !ok {
		return gschat.NewUserNotFound()
	}

	return agentQ.pull(offset)
}

func (agent *_IMAgent) Closed() {
	agent.server.removeDevice(agent.ID().String(), agent)
}

func (agent *_IMAgent) Dispatch(call *gorpc.Request) (callReturn *gorpc.Response, err error) {
	return agent.dispatcher.Dispatch(call)
}
