package gw

import (
	"sync"

	"github.com/gsrpc/gorpc"

	"github.com/gschat/gschat"
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
	token        []byte         // token string
}

func (proxy *_IMProxy) createBridge(context gsproxy.Context, client gsproxy.Client) (err error) {

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

	client.AddService(gschat.MakeIMAuth(uint16(gschat.ServiceTypeAuth), bridge))
	client.AddService(gschat.MakeIMPush(uint16(gschat.ServiceTypePush), bridge))

	proxy.bridges[client.Device().String()] = bridge

	proxy.Q.DeviceOnline(client.Device())

	return
}

func (bridge *_Bridge) close() {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.unbind()
}

func (bridge *_Bridge) Login(username string, properties []*gschat.Property) ([]*gschat.Property, error) {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.I("%s login with username %s", bridge.client.Device(), username)

	auth, ok := bridge.proxy.getAuth(bridge.client.Device())

	if !ok {
		// reject all income login request
		if bridge.proxy.rejectAll {

			return nil, gschat.NewUserNotFound()
		}

		properties = nil

	} else {

		var err error

		properties, err = gschat.BindIMAuth(0, auth).Login(username, properties)

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

	err := gschat.BindIManager(uint16(gschat.ServiceTypeIM), imserver).Bind(username, bridge.client.Device())

	if err != nil {
		bridge.E("%s bind im server %s error\n%s", username, bridge.proxy.serverName(imserver), err)
		return nil, gorpc.NewRemoteException()
	}

	bridge.I("bind server %s for %s -- success", bridge.proxy.serverName(imserver), username)

	bridge.client.Bind(uint16(gschat.ServiceTypeIM), imserver)

	bridge.username = username

	bridge.imserver = imserver

	bridge.proxy.Q.UserOnline(username, bridge.client.Device())

	bridge.I("%s login with username %s -- success", bridge.client.Device(), username)

	return properties, nil
}

func (bridge *_Bridge) unbind() error {

	if bridge.imserver != nil {

		err := gschat.BindIManager(uint16(gschat.ServiceTypeIM), bridge.imserver).Unbind(bridge.username, bridge.client.Device())

		if err != nil {
			bridge.E("unbind user %s from device %s error \n%s", bridge.username, bridge.client.Device())
		}
	}

	if bridge.username != "" {
		bridge.proxy.Q.UserOffline(bridge.username, bridge.client.Device())
	} else {
		bridge.proxy.Q.DeviceOffline(bridge.client.Device())
	}

	return nil
}

func (bridge *_Bridge) Register(pushToken []byte) error {

	bridge.proxy.Q.PushBind(bridge.client.Device(), pushToken)

	bridge.token = pushToken

	return nil
}

func (bridge *_Bridge) Unregister() error {

	bridge.proxy.Q.PushUnbind(bridge.client.Device(), bridge.token)

	return nil
}

func (bridge *_Bridge) Logoff(property []*gschat.Property) error {

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
	property = append(property, &gschat.Property{Key: "username", Value: bridge.username})

	// call real auth service
	err = gschat.BindIMAuth(0, auth).Logoff(property)

	if err != nil {

		bridge.imserver = nil

		bridge.E("%s logoff error\n%s", bridge.username, err)
		return err
	}

	return nil

}
