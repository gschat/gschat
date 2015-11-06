package gw

import (
	"fmt"
	"sync"

	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/hashring"
)

// the gschat proxy service
type _IMProxy struct {
	gslogger.Log                                          // mixin gslogger APIs
	sync.RWMutex                                          // read/write mutex
	servers      map[gsproxy.Server][]*gorpc.NamedService // bind servers
	services     map[string]*hashring.HashRing            // register services
	clients      map[string]*_IMBridge                    // clients
}

// New create new im proxy instance
func New() gsproxy.Proxy {
	return &_IMProxy{
		Log:      gslogger.Get("improxy"),
		servers:  make(map[gsproxy.Server][]*gorpc.NamedService),
		services: make(map[string]*hashring.HashRing),
		clients:  make(map[string]*_IMBridge),
	}
}

func (improxy *_IMProxy) Register(context gsproxy.Context) error {
	improxy.I("improxy started")
	return nil
}

func (improxy *_IMProxy) Unregister(context gsproxy.Context) {
	improxy.I("improxy closed")
}

func (improxy *_IMProxy) BindServices(context gsproxy.Context, server gsproxy.Server, services []*gorpc.NamedService) error {

	improxy.Lock()
	defer improxy.Unlock()

	improxy.I("bind new tranproxy services for %s", server)

	if _, ok := improxy.servers[server]; ok {
		improxy.unbindServices(context, server)
	}

	improxy.servers[server] = services

	for _, service := range services {

		improxy.I("bind service %s", service.Name)

		ring, ok := improxy.services[service.Name]

		if !ok {
			improxy.D("create new hashring for service %s", service)
			ring = hashring.New()
			improxy.services[service.Name] = ring
		}

		for i := uint32(0); i < service.VNodes; i++ {
			ring.Put(fmt.Sprintf("%s:%d", service.NodeName, i), server)
		}

		improxy.I("bind service %s -- success", service)
	}

	improxy.I("bind new tranproxy services for %s -- success", server)

	return nil
}

// service query server node with service name and shared key
func (improxy *_IMProxy) service(name string, sharedkey string) (gsproxy.Server, bool) {

	improxy.RLock()
	defer improxy.RUnlock()

	if ring, ok := improxy.services[name]; ok {
		if val, ok := ring.Get(sharedkey); ok {
			return val.(gsproxy.Server), true
		}
	}

	return nil, false

}

func (improxy *_IMProxy) unbindServices(context gsproxy.Context, server gsproxy.Server) {
	if services, ok := improxy.servers[server]; ok {
		for _, service := range services {
			improxy.I("unbind service %s", service)

			ring, ok := improxy.services[service.Name]

			if ok {
				for i := uint32(0); i < service.VNodes; i++ {
					ring.Remove(fmt.Sprintf("%s:%d", service.NodeName, i))
				}
			}

			improxy.I("unbind service %s -- success", service)
		}
	}
}

func (improxy *_IMProxy) UnbindServices(context gsproxy.Context, server gsproxy.Server) {
	improxy.Lock()
	defer improxy.Unlock()

	improxy.unbindServices(context, server)

}

func (improxy *_IMProxy) AddClient(context gsproxy.Context, client gsproxy.Client) error {

	improxy.Lock()
	defer improxy.Unlock()

	if old, ok := improxy.clients[client.Device().String()]; ok {
		old.Close()
	}

	improxy.clients[client.Device().String()] = improxy.newBridge(context, client)

	return nil
}

func (improxy *_IMProxy) RemoveClient(context gsproxy.Context, client gsproxy.Client) {
	improxy.Lock()
	defer improxy.Unlock()

	if old, ok := improxy.clients[client.Device().String()]; ok && old.client == client {
		old.Close()
		delete(improxy.clients, client.Device().String())
	}
}
