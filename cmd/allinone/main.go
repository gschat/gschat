package main

import (
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/server"
	"github.com/gsdocker/gsactor"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
)

func main() {

	var proxy gsproxy.Context

	var agents gsactor.AgentSystemContext

	defer func() {
		if e := recover(); e != nil {
			gslogger.Get("gschat-allinone").E("catch exception\n%s", e)
		}

		if proxy != nil {
			proxy.Close()
		}

		if agents != nil {
			agents.Close()
		}
	}()

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	var err error

	proxy, err = gsproxy.BuildProxy(gschat.NewIMProxy).AddrF(":13516").Run("im-test-proxy")

	if err != nil {
		panic(err)
	}

	<-time.After(time.Second)

	agents, err = gsactor.BuildAgentSystem(func() gsactor.AgentSystem {
		return server.NewIMServer(10)
	}).Run("im-test-server")

	if err != nil {
		panic(err)
	}

	<-make(chan bool)
}
