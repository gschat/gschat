package server

import (
	"github.com/gschat/gschat"
	"github.com/gsdocker/gsagent"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// IM agent device agent
type _IMAgent struct {
	gslogger.Log                     // Mixin log APIs
	gsagent.Context                  // agent context
	server          *_IMServer       // server
	dispatcher      gorpc.Dispatcher // serivce dispatcher
	device          *gorpc.Device    // device
	agentQ          *_IMAgentQ       // bound agent Q
}

func (server *_IMServer) AddAgent(context gsagent.Context) (gsagent.Agent, error) {

	server.Lock()
	defer server.Unlock()

	if agent, ok := server.agents[context.ID().String()]; ok {
		agent.close()
	}

	user, ok := server.binders[context.ID().String()]

	if !ok {
		return nil, gserrors.Newf(gschat.NewUserNotFound(), "device(%s) not bound to user", context.ID())
	}

	agent := &_IMAgent{
		Log:    gslogger.Get("im-agent"),
		server: server,
		device: context.ID(),
	}

	agent.agentQ = user.createAgentQ(context)

	agent.Context = context

	agent.dispatcher = gschat.MakeIMServer(uint16(gschat.ServiceTypeIM), agent)

	return agent, nil
}

func (agent *_IMAgent) close() {
	agent.agentQ.close()
}

func (agent *_IMAgent) Register(context gsagent.Context) error {
	return nil
}

func (agent *_IMAgent) Unregister(context gsagent.Context) {

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
	return agent.agentQ.pull(offset)
}

func (agent *_IMAgent) Dispatch(call *gorpc.Request) (callReturn *gorpc.Response, err error) {
	return agent.dispatcher.Dispatch(call)
}
