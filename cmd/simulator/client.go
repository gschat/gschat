package main

import (
	"sync"
	"time"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
	"github.com/gsrpc/gorpc/tcp"
)

type _Client struct {
	sync.Mutex
	gslogger.Log                // mixin log
	name         string         // client user name
	device       *gorpc.Device  // client device name
	mailhub      gschat.MailHub // mailhub
	pipeline     gorpc.Pipeline // connection
	seqID        uint32         // received id
	syncNum      uint32         // current sync expect num
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

		client.mailhub = gschat.BindMailHub(uint16(gschat.ServiceMailHub), pipeline)

		mail := gschat.NewMail()

		mail.Sender = client.name
		mail.Receiver = client.name
		mail.Type = gschat.MailTypeSingle

		if _, err := client.mailhub.Put(nil, mail); err != nil {
			go pipeline.Inactive()
			client.I("user %s send message error\n%s", client.name, err)
			return
		}

		client.pipeline = pipeline

	} else {
		client.I("client %s connection state changed %s", client.name, state)
		pipeline.RemoveService(gschat.MakeClient(uint16(gschat.ServiceClient), client))
	}
}

func (client *_Client) Push(callSite *gorpc.CallSite, mail *gschat.Mail) (err error) {

	client.Lock()
	defer client.Unlock()

	client.syncNum--

	client.D("recv message :%d , expect num %d", mail.SQID, client.syncNum)

	if client.seqID < mail.SQID {
		client.seqID = mail.SQID
	}

	if client.syncNum == 0 {
		client.D("sync finish ...")

		client.mailhub.Fin(callSite, mail.SQID)

		time.AfterFunc(5*time.Second, func() {
			if _, err := client.mailhub.Put(nil, mail); err != nil {
				go client.pipeline.Inactive()
				client.I("user %s send message error\n%s", client.name, err)
			}
		})

	}

	return nil
}

func (client *_Client) Notify(callSite *gorpc.CallSite, SQID uint32) (err error) {

	client.D("client recv notify SQID %d", SQID)

	if SQID > client.seqID {

		client.D("sync message")

		client.Lock()
		defer client.Unlock()

		client.syncNum, err = client.mailhub.Sync(callSite, client.seqID+1, SQID-client.seqID)

		if err != nil {
			client.E("call mailhub@Sync error :%s", err)
		}
	}

	return nil
}

func (client *_Client) DeviceStateChanged(callSite *gorpc.CallSite, device *gorpc.Device, online bool) (err error) {
	return nil
}
