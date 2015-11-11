package main

import (
	_ "net/http/pprof"
	"runtime"
	"strings"
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mailhub"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gsdiscovery"
	"github.com/gsdocker/gsdiscovery/zk"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsdocker/gsrunner"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/tcp"
)

var tunnels = make(map[string]tcp.Client)

var context gsagent.Context

var system mailhub.MailHub

func bindUserResolver(runner gsrunner.Runner, namedService *gorpc.NamedService) (gschat.UserResolver, error) {
	tunnel, err := context.Connect(namedService.NodeName, namedService.NodeName)

	if err != nil {
		return nil, err
	}

	return gschat.BindUserResolver(uint16(gschat.ServiceUserResolver), tunnel.Pipeline()), nil
}

func zkstart(runner gsrunner.Runner, zkservers []string) {
	discovery, err := zk.New(zkservers)

	if err != nil {
		gserrors.Panic(err)
	}

	proxyWatcher, err := discovery.Watch(gschat.NameOfGateway)

	if err != nil {
		gserrors.Panic(err)
	}

	userResolverWatcher, err := discovery.Watch(gschat.NameOfUserResolver)

	if err != nil {
		gserrors.Panic(err)
	}

	go func() {
		for {
			select {
			case event := <-proxyWatcher.Chan():
				switch event.State {
				case gsdiscovery.EvtCreated:

					for _, service := range event.Services {
						tunnel, err := context.Connect(service.NodeName, service.NodeName)

						if err != nil {
							runner.E("connect to %s(%s) error :%s", gschat.NameOfGateway, service.NodeName, err)
							continue
						}

						tunnels[service.NodeName] = tunnel

					}
				case gsdiscovery.EvtDeleted:

					for _, service := range event.Services {

						if tunnel, ok := tunnels[service.NodeName]; ok {
							tunnel.Close()
							delete(tunnels, service.NodeName)
						}
					}

				}
			case event := <-userResolverWatcher.Chan():
				switch event.State {
				case gsdiscovery.EvtCreated:
					for _, service := range event.Services {
						resolver, err := bindUserResolver(runner, service)

						if err != nil {
							runner.E("connect to %s(%s) error :%s", gschat.NameOfUserResolver, service.NodeName, err)
							continue
						}

						system.AddUserResolver(service, resolver)
					}
				case gsdiscovery.EvtUpdated:
					for _, service := range event.Services {
						system.RemoveUserResolver(service)
					}

					for _, service := range event.Updates {
						resolver, err := bindUserResolver(runner, service)

						if err != nil {
							runner.E("connect to %s(%s) error :%s", gschat.NameOfUserResolver, service.NodeName, err)
							continue
						}

						system.AddUserResolver(service, resolver)
					}

				case gsdiscovery.EvtDeleted:
					for _, service := range event.Services {
						system.RemoveUserResolver(service)
					}
				}
			}
		}
	}()
}

func run(runner gsrunner.Runner) {

	node := gsconfig.String("gschat.mailhub.node", "")

	zkservers := strings.Split(gsconfig.String("gschat.mailhub.zk", ""), "|")

	proxies := strings.Split(gsconfig.String("gschat.mailhub.proxies", ""), "|")

	runner.I("node name :%s", node)
	runner.I("zkservers(%d) :%v", len(zkservers), zkservers)
	runner.I("proxies :%v", proxies)

	eventLoop := gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	system = mailhub.New(node, nil)

	context = gsagent.BuildAgent(system).Build(node, eventLoop)

	if gsconfig.String("gschat.mailhub.zk", "") != "" {
		zkstart(runner, zkservers)
	} else {
		for _, proxy := range proxies {
			tunnel, err := context.Connect(proxy, proxy)

			if err != nil {
				runner.E("connect to proxy(%s) error :%s", err)
			}

			tunnels[proxy] = tunnel
		}
	}

	for _ = range time.Tick(20 * time.Second) {
		runner.I("\n%s", gorpc.PrintProfile())
	}
}

func main() {
	runner := gsrunner.New("gschat-mailhub")

	runner.FlagString(
		"proxies", "gschat.mailhub.proxies", "localhost:15827", "gschat proxy services list",
	).FlagString(
		"node", "gschat.mailhub.node", "localhost:15111", "gschat proxy service back side listen address",
	).FlagString(
		"zk", "gschat.mailhub.zk", "", "the zookeeper server list",
	)

	runner.Run(run)
}
