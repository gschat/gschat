package main

import (
	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/server"
	"github.com/gsdocker/gsagent"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
)

func main() {

	var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	var proxy gsproxy.Context

	var agents gsagent.SystemContext

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

	log := gslogger.Get("profile")

	go func() {
		log.E("%s", http.ListenAndServe("localhost:6060", nil))
	}()

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	var err error

	proxy = gsproxy.BuildProxy(gschat.NewIMProxy()).AddrF(":13516").Build("im-test-proxy", eventLoop)

	if err != nil {
		panic(err)
	}

	<-time.After(time.Second)

	agentsystem := gsagent.New("im-test-server", server.NewIMServer(10), eventLoop, 5)

	agentsystem.Connect("im-test-proxy", "127.0.0.1:15827", 1024, 5*time.Second)

	for _ = range time.Tick(20 * time.Second) {
		log.I("\n%s", gorpc.PrintProfile())
	}
}
