package gw

import (
	"fmt"
	"sync"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/eventQ"
	"github.com/gschat/gschat/hashring"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
)

type _IMProxy struct {
	sync.RWMutex                                         // mutex
	gslogger.Log                                         // Mixin Log APIs
	name         string                                  // proxy name
	servers      map[gsproxy.Server]*gschat.NamedService // register servers
	bridges      map[string]*_Bridge                     // bridges
	imservers    *hashring.HashRing                      // imserver hash ring
	anps         *hashring.HashRing                      // ios push server
	auth         *hashring.HashRing                      // ios push server
	rejectAll    bool                                    // set the default auth behavor if no ath server found
	Q            eventQ.Q                                // eventQ
}

// NewIMProxy create new im proxy
func NewIMProxy(Q eventQ.Q) gsproxy.Proxy {
	return &_IMProxy{
		Log:       gslogger.Get("improxy"),
		servers:   make(map[gsproxy.Server]*gschat.NamedService),
		bridges:   make(map[string]*_Bridge),
		imservers: hashring.New(),
		anps:      hashring.New(),
		auth:      hashring.New(),
		rejectAll: gsconfig.Bool("gschat.auth.default.rejectall", false),
		Q:         Q,
	}
}

func (proxy *_IMProxy) Register(context gsproxy.Context) error {
	proxy.D("open improxy %s", context.String())
	proxy.name = context.String()

	return nil
}

func (proxy *_IMProxy) Unregister(context gsproxy.Context) {
	proxy.D("close improxy", context.String())
}

func (proxy *_IMProxy) AddServer(context gsproxy.Context, server gsproxy.Server) error {

	proxy.Lock()
	defer proxy.Unlock()

	proxy.servers[server] = nil

	go proxy.bindServer(context, server)

	return nil
}

func (proxy *_IMProxy) bindServer(context gsproxy.Context, server gsproxy.Server) {
	proxy.D("try bind server [%p] ...", server)

	service := gschat.BindService(uint16(gschat.ServiceTypeUnknown), server)

	namedService, err := service.Name()

	if err != nil {
		proxy.E("query server(%p) provide service type error\n%s", server, err)
		proxy.RemoveServer(context, server)
		return
	}

	proxy.dobind(context, server, namedService)

}

func (proxy *_IMProxy) dobind(context gsproxy.Context, server gsproxy.Server, namedService *gschat.NamedService) {
	proxy.Lock()
	defer proxy.Unlock()

	if _, ok := proxy.servers[server]; !ok {
		proxy.W("server(%p) already bound", server)
		return
	}

	var ring *hashring.HashRing

	switch namedService.Type {
	case gschat.ServiceTypeIM:
		ring = proxy.imservers
	case gschat.ServiceTypePush:
		ring = proxy.anps
	case gschat.ServiceTypeAuth:
		ring = proxy.auth
	default:
		proxy.E("server(%p) provide unknown service type :%s", server, namedService.Type)
		go proxy.RemoveServer(context, server)
		return
	}

	proxy.servers[server] = namedService

	for i := uint32(0); i < namedService.VNodes; i++ {
		ring.Put(fmt.Sprintf("%s:%d", namedService.Name, i), server)
	}

	proxy.D("bind service [%p] with name [%s] -- success", server, namedService)
}

func (proxy *_IMProxy) RemoveServer(context gsproxy.Context, server gsproxy.Server) {
	proxy.Lock()
	defer proxy.Unlock()

	namedService, ok := proxy.servers[server]

	if !ok || namedService == nil {
		proxy.W("server(%p) already unbound", server)
		return
	}

	var ring *hashring.HashRing

	switch namedService.Type {
	case gschat.ServiceTypeIM:
		ring = proxy.imservers
	case gschat.ServiceTypePush:
		ring = proxy.anps
	case gschat.ServiceTypeAuth:
		ring = proxy.auth
	default:
		proxy.E("server(%p) provide unknown service type :%s", server, namedService.Type)
		return
	}

	delete(proxy.servers, server)

	for i := uint32(0); i < namedService.VNodes; i++ {
		ring.Remove(fmt.Sprintf("%s:%d", namedService.Name, i))
	}
}

func (proxy *_IMProxy) getAuth(device *gorpc.Device) (gsproxy.Server, bool) {

	val, ok := proxy.auth.Get(device.String())

	if !ok {
		return nil, false
	}

	return val.(gsproxy.Server), true
}

func (proxy *_IMProxy) getIM(name string) (gsproxy.Server, bool) {
	val, ok := proxy.imservers.Get(name)

	if !ok {
		return nil, false
	}

	return val.(gsproxy.Server), true
}

func (proxy *_IMProxy) getANPS(name string) (gsproxy.Server, bool) {

	val, ok := proxy.imservers.Get(name)

	if !ok {
		return nil, false
	}

	return val.(gsproxy.Server), true
}

func (proxy *_IMProxy) serverName(server gsproxy.Server) string {
	proxy.RLock()
	defer proxy.RUnlock()

	if namedService, ok := proxy.servers[server]; ok && namedService != nil {
		return namedService.String()
	}

	return "unknown"
}

func (proxy *_IMProxy) AddClient(context gsproxy.Context, client gsproxy.Client) error {

	proxy.createBridge(context, client)

	return nil
}

func (proxy *_IMProxy) RemoveClient(context gsproxy.Context, client gsproxy.Client) {

	proxy.I("remove client :%s", client.Device())

	proxy.Lock()
	defer proxy.Unlock()

	bridge, ok := proxy.bridges[client.Device().String()]
	if !ok {
		proxy.W("client :%s not found", client.Device())
		return
	}

	delete(proxy.bridges, client.Device().String())

	go bridge.close()
}
