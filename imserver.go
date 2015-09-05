package gschat

import (
	"github.com/gsdocker/gsactor"
	"github.com/gsdocker/gslogger"
)

// implement gsactor.AgentSystem
type _IMServer struct {
	gslogger.Log                            // Mixin logger
	context      gsactor.AgentSystemContext // agent system context
}

// NewIMServer create new im server
func NewIMServer() gsactor.AgentSystem {
	return &_IMServer{
		Log: gslogger.Get("imserver"),
	}
}

func (server *_IMServer) Open(context gsactor.AgentSystemContext) error {

	server.D("open")

	server.context = context

	return nil
}

func (server *_IMServer) Close() {
	server.D("closed")
}
