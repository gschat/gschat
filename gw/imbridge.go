package gw

import (
	"sync"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
)

type _IMBridge struct {
	gslogger.Log                // mixin gslogger
	sync.Mutex                  // login mutex
	improxy      *IMProxy       // improxy object
	client       gsproxy.Client // bound client
	mailhub      gsproxy.Server // bound mailhub
	pushservice  gsproxy.Server // bound pushservice
	username     string         // login username
}

func (improxy *IMProxy) newBridge(context gsproxy.Context, client gsproxy.Client) *_IMBridge {
	bridge := &_IMBridge{
		Log:     gslogger.Get("imbridge"),
		improxy: improxy,
		client:  client,
	}

	client.AddService(gschat.MakeAuth(gorpc.ServiceID(gschat.NameOfAuth), bridge))
	client.AddService(gschat.MakeAuth(gorpc.ServiceID(gschat.NameOfPush), bridge))

	pushservice, ok := bridge.improxy.service(gschat.NameOfPushServiceProvider, bridge.client.Device().String())

	if ok {
		bridge.pushservice = pushservice

		err := gschat.BindPushServiceProvider(gorpc.ServiceID(gschat.NameOfPushServiceProvider), bridge.pushservice).DeviceStatusChanged(nil, bridge.client.Device(), true)

		if err != nil {
			bridge.E("notify pushservice %s device %s offline error \n%s", bridge.pushservice, bridge.client.Device(), err)
		}
	}

	return bridge
}

func (bridge *_IMBridge) Close() {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.I("close client")

	go bridge.close()
}

func (bridge *_IMBridge) close() {
	if bridge.mailhub != nil {

		err := gschat.BindUserBinder(gorpc.ServiceID(gschat.NameOfUserBinder), bridge.mailhub).UnbindUser(nil, bridge.username, bridge.client.Device())

		if err != nil {
			bridge.E("mailhub %s unbind user %s from device %s error \n%s", bridge.mailhub, bridge.username, bridge.client.Device(), err)
		}
	}

	if bridge.pushservice != nil {
		err := gschat.BindPushServiceProvider(gorpc.ServiceID(gschat.NameOfPushServiceProvider), bridge.pushservice).DeviceStatusChanged(nil, bridge.client.Device(), false)

		if err != nil {
			bridge.E("notify pushservice %s device %s offline error \n%s", bridge.pushservice, bridge.client.Device(), err)
		}
	}
}

func (bridge *_IMBridge) Register(callSite *gorpc.CallSite, pushToken []byte) (err error) {

	bridge.Lock()
	defer bridge.Unlock()

	if bridge.pushservice == nil {
		return nil
	}

	err = gschat.BindPushServiceProvider(gorpc.ServiceID(gschat.NameOfPushServiceProvider), bridge.pushservice).DeviceRegister(callSite, bridge.client.Device(), pushToken)

	if err != nil {
		bridge.E("notify pushservice %s new device %s push token error\n%s", bridge.pushservice, bridge.client.Device(), err)
		return gschat.NewResourceNotFound()
	}

	return nil
}

func (bridge *_IMBridge) Unregister(callSite *gorpc.CallSite) (err error) {

	bridge.Lock()
	defer bridge.Unlock()

	if bridge.pushservice == nil {
		return nil
	}

	err = gschat.BindPushServiceProvider(gorpc.ServiceID(gschat.NameOfPushServiceProvider), bridge.pushservice).DeviceUnregister(callSite, bridge.client.Device())

	if err != nil {
		bridge.E("notify pushservice %s device %s unregister push token error\n%s", bridge.pushservice, bridge.client.Device(), err)
		return gschat.NewResourceNotFound()
	}

	return nil
}

func (bridge *_IMBridge) Login(callSite *gorpc.CallSite, username string, properties []*gorpc.KV) (retval []*gorpc.KV, err error) {

	bridge.Lock()
	defer bridge.Unlock()

	bridge.I("user %s login with device :%s", username, bridge.client.Device())

	auth, ok := bridge.improxy.service(gschat.NameOfAuth, bridge.client.Device().String())

	if ok {

		properties, err = gschat.BindAuth(gorpc.ServiceID(gschat.NameOfAuth), auth).Login(callSite, username, properties)

		if err != nil {
			return properties, err
		}
	}

	//bind mailhub

	bridge.mailhub, ok = bridge.improxy.service(gschat.NameOfMailHub, bridge.client.Device().String())

	if !ok {
		bridge.E("there is no valid mailhub service")
		return nil, gschat.NewResourceNotFound()
	}

	err = gschat.BindUserBinder(gorpc.ServiceID(gschat.NameOfUserBinder), bridge.mailhub).BindUser(callSite, username, bridge.client.Device())

	if err != nil {
		bridge.E("mailhub %s bind user %s from device %s error \n%s", bridge.mailhub, bridge.username, bridge.client.Device(), err)

		return nil, err
	}

	bridge.username = username

	bridge.client.TransproxyBind(gorpc.ServiceID(gschat.NameOfMailHub), bridge.mailhub)

	bridge.I("device %s login with username %s -- success", bridge.client.Device(), username)

	return nil, nil
}

func (bridge *_IMBridge) Logoff(callSite *gorpc.CallSite, properties []*gorpc.KV) (err error) {
	bridge.Lock()
	defer bridge.Unlock()

	if bridge.mailhub != nil {

		err := gschat.BindUserBinder(gorpc.ServiceID(gschat.NameOfUserBinder), bridge.mailhub).UnbindUser(nil, bridge.username, bridge.client.Device())

		if err != nil {
			bridge.E("mailhub %s unbind user %s from device %s error \n%s", bridge.mailhub, bridge.username, bridge.client.Device(), err)
		}
	}

	return nil
}
