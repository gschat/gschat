package gschat

import (
	"fmt"
	"sync"

	"github.com/gschat/gschat/hashring"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
)

type _IMProxy struct {
	sync.RWMutex                                  // mutex
	gslogger.Log                                  // Mixin Log APIs
	name         string                           // proxy name
	servers      map[gsproxy.Server]*NamedService // register servers
	bridges      map[string]*_Bridge              // bridges
	imservers    *hashring.HashRing               // imserver hash ring
	anps         *hashring.HashRing               // ios push server
	auth         *hashring.HashRing               // ios push server
	rejectAll    bool                             // set the default auth behavor if no ath server found
}

// NewIMProxy create new im proxy
func NewIMProxy() gsproxy.Proxy {
	return &_IMProxy{
		Log:       gslogger.Get("improxy"),
		servers:   make(map[gsproxy.Server]*NamedService),
		imservers: hashring.New(),
		anps:      hashring.New(),
		auth:      hashring.New(),
		rejectAll: gsconfig.Bool("gschat.auth.default.rejectall", false),
	}
}

func (proxy *_IMProxy) OpenProxy(context gsproxy.Context) error {
	proxy.D("open improxy %s", context.Name())
	proxy.name = context.Name()

	return nil
}

func (proxy *_IMProxy) CloseProxy(context gsproxy.Context) {
	proxy.D("close improxy", context.Name())
}

func (proxy *_IMProxy) CreateServer(server gsproxy.Server) error {

	proxy.Lock()
	defer proxy.Unlock()

	proxy.servers[server] = nil

	go proxy.bindServer(server)

	return nil
}

func (proxy *_IMProxy) bindServer(server gsproxy.Server) {
	proxy.D("try bind server [%p] ...", server)

	service := BindService(uint16(ServiceTypeUnknown), server)

	namedService, err := service.Name()

	if err != nil {
		proxy.E("query server(%p) provide service type error\n%s", server, err)
		proxy.CloseServer(server)
		return
	}

	proxy.dobind(server, namedService)

}

func (proxy *_IMProxy) dobind(server gsproxy.Server, namedService *NamedService) {
	proxy.Lock()
	defer proxy.Unlock()

	if _, ok := proxy.servers[server]; !ok {
		proxy.W("server(%p) already unbound", server)
		return
	}

	var ring *hashring.HashRing

	switch namedService.Type {
	case ServiceTypeIM:
		ring = proxy.imservers
	case ServiceTypeANPS:
		ring = proxy.anps
	case ServiceTypeAuth:
		ring = proxy.auth
	default:
		proxy.E("server(%p) provide unknown service type :%s", server, namedService.Type)
		go proxy.CloseServer(server)
		return
	}

	proxy.servers[server] = namedService

	for i := uint32(0); i < namedService.VNodes; i++ {
		ring.Put(fmt.Sprintf("%s:%d", namedService.Name, i), server)
	}

	proxy.D("bind service [%p] with name [%s] -- success", server, namedService)
}

func (proxy *_IMProxy) CloseServer(server gsproxy.Server) {
	proxy.Lock()
	defer proxy.Unlock()

	namedService, ok := proxy.servers[server]

	if !ok || namedService == nil {
		proxy.W("server(%p) already unbound", server)
		return
	}

	var ring *hashring.HashRing

	switch namedService.Type {
	case ServiceTypeIM:
		ring = proxy.imservers
	case ServiceTypeANPS:
		ring = proxy.anps
	case ServiceTypeAuth:
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

func (proxy *_IMProxy) getAuth(device gsproxy.Device) (gsproxy.Server, bool) {

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

func (proxy *_IMProxy) CreateDevice(device gsproxy.Device) error {

	proxy.createBridge(device)

	return nil
}

func (proxy *_IMProxy) CloseDevice(device gsproxy.Device) {

	proxy.Lock()
	defer proxy.Unlock()

	bridge, ok := proxy.bridges[device.String()]
	if !ok {
		return
	}

	delete(proxy.bridges, device.String())

	go bridge.close()
}