// package gschat-proxy the gscaht proxy server
package main

import (
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/gschat/gschat/gw"
	"github.com/gsdocker/gsproxy"
	"github.com/gsdocker/gsrunner"
	"github.com/gsrpc/gorpc"
)

func run(runner gsrunner.Runner) {

	eventLoop := gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	improxy := gw.New()

	gsproxy.BuildProxy(improxy).DHKeyResolver(improxy.NewDHKeyResolver()).Build("gschat-proxy", eventLoop)

	for _ = range time.Tick(20 * time.Second) {
		runner.I("\n%s", gorpc.PrintProfile())
	}
}

func main() {
	runner := gsrunner.New("gschat-proxy")

	runner.FlagString(
		"laddr", "gschat.proxy.laddr", ":13516", "gschat proxy service front side listen address",
	).FlagString(
		"node", "gschat.proxy.node", ":15827", "gschat proxy service back side listen address",
	)

	runner.Run(run)
}
