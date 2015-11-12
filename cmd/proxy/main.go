// package gschat-proxy the gscaht proxy server
package main

import (
	_ "net/http/pprof"
	"runtime"
	"strings"
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/gw"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gsdiscovery/zk"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gsproxy"
	"github.com/gsdocker/gsrunner"
	"github.com/gsrpc/gorpc"
)

func run(runner gsrunner.Runner) {

	eventLoop := gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	improxy := gw.New()

	gsproxy.BuildProxy(improxy).DHKeyResolver(improxy.NewDHKeyResolver()).Build("gschat-proxy", eventLoop)

	if gsconfig.String("gschat.mailhub.zk", "") != "" {
		zkservers := strings.Split(gsconfig.String("gschat.mailhub.zk", ""), "|")
		discovery, err := zk.New(zkservers)

		if err != nil {
			gserrors.Panicf(err, "create new discovery service error")
		}

		node := gsconfig.String("gschat.proxy.node", "")

		err = discovery.Register(&gorpc.NamedService{
			Name:       gschat.NameOfGateway,
			DispatchID: uint16(gschat.ServiceGateway),
			VNodes:     gsconfig.Uint32("gschat.proxy.vnodes", 4),
			NodeName:   node,
		})

		if err != nil {
			gserrors.Panicf(err, "register service(%s) %s error", gschat.NameOfGateway, node)
		}
	}

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
	).FlagString(
		"zk", "gschat.mailhub.zk", "10.0.0.213:2181", "the zookeeper server list",
	)

	runner.Run(run)
}
