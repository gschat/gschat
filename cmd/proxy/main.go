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

	gsproxy.BuildProxy(
		improxy,
	).DHKeyResolver(
		improxy.NewDHKeyResolver(),
	).AddrF(
		gsconfig.String("gschat.proxy.laddr", ""),
	).AddrB(
		gsconfig.String("gschat.proxy.node", ""),
	).Build("gschat-proxy", eventLoop)

	if gsconfig.String("gschat.zk", "") != "" {
		zkservers := strings.Split(gsconfig.String("gschat.zk", ""), "|")
		discovery, err := zk.New(zkservers)

		if err != nil {
			gserrors.Panicf(err, "create new discovery service error")
		}

		discovery.WatchPath(gsconfig.String("gschat.zk.nodes", ""))

		err = discovery.UpdateRegistry(gsconfig.String("gschat.zk.regsitry", ""))

		if err != nil {
			gserrors.Panicf(err, "discovery update gsrpc registry error")
		}

		node := gsconfig.String("gschat.proxy.node", "")

		err = discovery.Register(&gorpc.NamedService{
			Name:       gschat.NameOfGateway,
			DispatchID: gorpc.ServiceID(gschat.NameOfGateway),
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
		"node", "gschat.proxy.node", ":15672", "gschat proxy service back side listen address",
	).FlagString(
		"zk", "gschat.zk", "10.0.0.213:2181", "the zookeeper server list",
	).FlagString(
		"zk-registry", "gschat.zk.regsitry", "/gschat/registry", "the zookeeper server list",
	).FlagString(
		"zk-nodes", "gschat.zk.nodes", "/gschat/nodes", "the zookeeper server list",
	)

	runner.Run(run)
}
