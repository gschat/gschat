package gschat

import (
	"github.com/gsdocker/gsactor"
	"github.com/gsrpc/gorpc"
)

// IM client device agent
type _IMAgent struct {
	gorpc.Dispatcher                      // Mixin dispatcher
	context          gsactor.AgentContext // agent context
}

func (server *_IMServer) Agent(context gsactor.AgentContext) (gsactor.Agent, error) {

	agent := &_IMAgent{
		context: context,
	}

	return agent, nil
}

func (client *_IMAgent) Closed() {
}

func (client *_IMAgent) Put(data *Data) (retval uint64, err error) {
	return 0, nil
}

func (client *_IMAgent) Pull(ts uint64) (err error) {
	return nil
}
