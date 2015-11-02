package main

import (
	"time"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
	"github.com/gsrpc/gorpc/tcp"
)

type _Client struct {
	gslogger.Log               // mixin log
	name         string        // client user name
	device       *gorpc.Device // client device name
}

func newClient(eventLoop gorpc.EventLoop, name string, raddr string, heartbeat time.Duration, dhkeyResolver handler.DHKeyResolver) (*_Client, error) {
	client := &_Client{
		Log:  gslogger.Get("simulator"),
		name: name,
	}

	client.device = gorpc.NewDevice()

	client.device.ID = name
	client.device.AppKey = "gschat.simulator"

	dhkey, err := dhkeyResolver.Resolve(client.device)

	if err != nil {
		return nil, err
	}

	tcp.BuildClient(
		gorpc.BuildPipeline(eventLoop).Handler(
			"profile",
			gorpc.ProfileHandler,
		).Handler(
			"dh",
			func() gorpc.Handler {
				return handler.NewCryptoClient(client.device, dhkey)
			},
		).Handler(
			"state",
			func() gorpc.Handler {
				return handler.NewStateHandler(func(pipeline gorpc.Pipeline, state gorpc.State) {
					go client.StateChanged(pipeline, state)
				})
			},
		).Handler(
			"heatbeat-client",
			func() gorpc.Handler {
				return handler.NewHeartbeatHandler(heartbeat)
			},
		),
	).Remote(raddr).Reconnect(5 * time.Second).Connect(name)

	return client, nil
}

func (client *_Client) StateChanged(pipeline gorpc.Pipeline, state gorpc.State) {

	if state == gorpc.StateConnected {

		pipeline.AddService(gschat.MakeClient(uint16(gschat.ServiceClient), client))

		client.I("user %s login ...", client.name)

		auth := gschat.BindAuth(uint16(gschat.ServiceAuth), pipeline)

		_, err := auth.Login(nil, client.name, nil)

		if err != nil {
			go pipeline.Inactive()
			client.I("user %s login failed\n%s", client.name, err)
			return
		}

		client.I("user %s login success", client.name)
	} else {
		client.I("client %s connection state changed %s", client.name, state)
		pipeline.RemoveService(gschat.MakeClient(uint16(gschat.ServiceClient), client))
	}
}

func (client *_Client) Push(callSite *gorpc.CallSite, mail *gschat.Mail) (err error) {
	return nil
}

func (client *_Client) Notify(callSite *gorpc.CallSite, SQID uint32) (err error) {
	return nil
}

func (client *_Client) DeviceStateChanged(callSite *gorpc.CallSite, device *gorpc.Device, online bool) (err error) {
	return nil
}
