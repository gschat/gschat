package gschat

import (
	"sync"

	"github.com/gsrpc/gorpc"

	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
)

// gschat proxy bridge
type _Bridge struct {
	gslogger.Log                // mixin log APIs
	sync.Mutex                  // mutex
	client       gsproxy.Client // bridge device
	proxy        *_IMProxy      // proxy belongs to
	imserver     gsproxy.Server // server
	username     string         // login username
}

func (proxy *_IMProxy) createBridge(context gsproxy.Context, client gsproxy.Client) {

	proxy.Lock()
	defer proxy.Unlock()

	bridge, ok := proxy.bridges[client.Device().String()]

	if ok {
		go bridge.close()
	}

	bridge = &_Bridge{
		Log:    gslogger.Get("bridge"),
		client: client,
		proxy:  proxy,
	}

	client.AddService(MakeIMAuth(uint16(ServiceTypeAuth), bridge))

	if bridge.client.Device().OS == gorpc.OSTypeIOS {
		anps, ok := bridge.proxy.getANPS(client.Device().String())

		if ok {
			bridge.client.Bind(uint16(ServiceTypeAPNS), anps)
		} else {
			bridge.W("not found valid anps server for %s", client.Device().String())
		}
	}

	proxy.bridges[client.Device().String()] = bridge
}

func (bridge *_Bridge) close() {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.unbind()
}

func (bridge *_Bridge) Login(username string, properties []*Property) ([]*Property, error) {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.I("%s login with username %s", bridge.client.Device(), username)

	auth, ok := bridge.proxy.getAuth(bridge.client.Device())

	if !ok {
		// reject all income login request
		if bridge.proxy.rejectAll {

			return nil, NewUserNotFound()
		}

		properties = nil

	} else {

		var err error

		properties, err = BindIMAuth(0, auth).Login(username, properties)

		if err != nil {
			return nil, err
		}
	}

	imserver, ok := bridge.proxy.getIM(username)

	if !ok {
		bridge.W("im server not found for %s ", username)
		return nil, gorpc.NewRemoteException()
	}

	bridge.I("bind server %s for %s ", bridge.proxy.serverName(imserver), username)

	err := BindIManager(uint16(ServiceTypeIM), imserver).Bind(username, bridge.client.Device())

	if err != nil {
		bridge.E("%s bind im server %s error\n%s", username, bridge.proxy.serverName(imserver), err)
		return nil, gorpc.NewRemoteException()
	}

	bridge.I("bind server %s for %s -- success", bridge.proxy.serverName(imserver), username)

	bridge.client.Bind(uint16(ServiceTypeIM), imserver)

	bridge.username = username

	bridge.imserver = imserver

	bridge.I("%s login with username %s -- success", bridge.client.Device(), username)

	return properties, nil
}

func (bridge *_Bridge) unbind() error {

	if bridge.imserver != nil {

		err := BindIManager(uint16(ServiceTypeIM), bridge.imserver).Unbind(bridge.username, bridge.client.Device())

		if err != nil {
			bridge.E("unbind user %s from device %s error \n%s", bridge.username, bridge.client.Device())
		}
	}

	return nil
}

func (bridge *_Bridge) Logoff(property []*Property) error {

	bridge.Lock()
	defer bridge.Unlock()

	if bridge.imserver == nil {
		bridge.W("%s already logoff", bridge.client)
		return nil
	}

	err := bridge.unbind()

	if err != nil {
		bridge.E("unbind %s with %s err\n%s", bridge.username, bridge.client, err)
	}

	bridge.imserver = nil

	auth, ok := bridge.proxy.getAuth(bridge.client.Device())

	if !ok {

		return err
	}

	// append username property
	property = append(property, &Property{Key: "username", Value: bridge.username})

	// call real auth service
	err = BindIMAuth(0, auth).Logoff(property)

	if err != nil {

		bridge.imserver = nil

		bridge.E("%s logoff error\n%s", bridge.username, err)
		return err
	}

	return nil

}
