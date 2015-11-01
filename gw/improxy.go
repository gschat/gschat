package improxy

import (
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
}

// New create new im proxy instance
func New() gsproxy.Proxy {
	return &_IMProxy{
		Log: gslogger.Get("improxy"),
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
	improxy.I("bind new tranproxy services")

	return nil
}

func (improxy *_IMProxy) UnbindServices(context gsproxy.Context, server gsproxy.Server) {

}

func (improxy *_IMProxy) AddClient(context gsproxy.Context, client gsproxy.Client) error {
	return nil
}

func (improxy *_IMProxy) RemoveClient(context gsproxy.Context, client gsproxy.Client) {

}
