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
	device       gsproxy.Device // bridge device
	proxy        *_IMProxy      // proxy belongs to
	imserver     gsproxy.Server // server
	username     string         // login username
}

func (proxy *_IMProxy) createBridge(device gsproxy.Device) {

	bridge := &_Bridge{
		Log:    gslogger.Get("bridge"),
		device: device,
		proxy:  proxy,
	}

	device.Register(MakeIMAuth(uint16(ServiceTypeAuth), bridge))

	if bridge.device.ID().OS == gorpc.OSTypeIOS {
		anps, ok := bridge.proxy.getANPS(device.String())

		if ok {
			bridge.device.Bind(uint16(ServiceTypeANPS), anps)
		} else {
			bridge.W("not found valid anps server for %s", device.String())
		}
	}
}

func (bridge *_Bridge) close() {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.unbind()
}

func (bridge *_Bridge) Login(username string, properties []*Property) ([]*Property, error) {
	bridge.Lock()
	defer bridge.Unlock()

	auth, ok := bridge.proxy.getAuth(bridge.device)

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

	err := BindIManager(uint16(ServiceTypeIM), imserver).Bind(username, bridge.device.ID())

	if err != nil {
		bridge.E("%s bind im server %s error\n%s", username, bridge.proxy.serverName(imserver), err)
		return nil, gorpc.NewRemoteException()
	}

	bridge.V("bind server %s for %s -- success", bridge.proxy.serverName(imserver), username)

	bridge.device.Bind(uint16(ServiceTypeIM), imserver)

	bridge.username = username

	bridge.imserver = imserver

	return properties, nil
}

func (bridge *_Bridge) unbind() error {
	if bridge.imserver != nil {
		return BindIManager(uint16(ServiceTypeIM), bridge.imserver).Unbind(bridge.username, bridge.device.ID())
	}

	return nil
}

func (bridge *_Bridge) Logoff(property []*Property) error {

	bridge.Lock()
	defer bridge.Unlock()

	if bridge.imserver == nil {
		bridge.W("%s already logoff", bridge.device)
		return nil
	}

	err := bridge.unbind()

	if err != nil {
		bridge.E("unbind %s with %s err\n%s", bridge.username, bridge.device, err)
	}

	bridge.imserver = nil

	auth, ok := bridge.proxy.getAuth(bridge.device)

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
