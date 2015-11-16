package mailhub

import (
	"sync/atomic"
	"time"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsrpc/gorpc"
)

type _Client struct {
	gslogger.Log               // mixin gslogger
	api          gschat.Client // client rpc interface
	mailbox      *_MailBox     // mailbox the client own
	agent        gsagent.Agent // client'a agent object
	sendID       uint32        // the send sequence id
	syncFlag     uint32        // the sync flag
	sync         *_Sync        // current sync session
	closed       chan bool     // closed channel
}

func (mailbox *_MailBox) newClient(agent gsagent.Agent) *_Client {
	wrapper := &_Client{
		Log:     gslogger.Get("mailbox"),
		agent:   agent,
		mailbox: mailbox,
		closed:  make(chan bool),
	}

	// create client proxy interface
	wrapper.api = gschat.BindClient(gorpc.ServiceID(gschat.NameOfClient), agent)

	// register mailhub interface
	agent.AddService(gschat.MakeMailHub(gorpc.ServiceID(gschat.NameOfMailHub), wrapper))

	wrapper.I("create new client %s for %s", agent.ID(), mailbox.username)

	go wrapper.notifyLoop()

	return wrapper
}

func (client *_Client) Device() *gorpc.Device {
	return client.agent.ID()
}

func (client *_Client) Close() {
	close(client.closed)
	client.agent.Close()

	if atomic.CompareAndSwapUint32(&client.syncFlag, 1, 0) {
		client.sync.Close()
		client.sync = nil
	}
}

func (client *_Client) notifyLoop() {

	id, err := client.mailbox.receivedID()

	if err != nil {
		client.E("user %s(%s) notify groutine query received id error :%s", client.mailbox.username, client.agent.ID(), err)
	} else {
		client.I("send %s's message notify to device %s", client.mailbox.username, client.agent.ID())
		client.notifyClient(id)
	}

	ticker := time.NewTicker(gsconfig.Seconds("gschat.mailhub.notify.duration", 60))

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			id, err := client.mailbox.receivedID()

			if err != nil {
				client.E("user %s(%s) notify groutine query received id error :%s", client.mailbox.username, client.agent.ID(), err)
				continue
			}

			client.notifyClient(id)

			client.I("send %s's message notify to device %s", client.mailbox.username, client.agent.ID())

		case <-client.closed:
			client.V("stop notify goroutine for device %s login with %s", client.agent.ID(), client.mailbox.username)
			return
		}
	}
}

func (client *_Client) notifyClient(id uint32) {

	if !atomic.CompareAndSwapUint32(&client.syncFlag, 0, 0) {
		client.V("skip notify user %s(%s) received SEQ id", client.mailbox.username, client.agent.ID())
	}

	err := client.api.Notify(nil, id)

	if err != nil {
		client.E("user %s(%s) notify groutine query received id error :%s", client.mailbox.username, client.agent.ID(), err)
	}
}

func (client *_Client) PutSync(callSite *gorpc.CallSite) (retval uint32, err error) {
	return client.sendID, nil
}

func (client *_Client) Put(callSite *gorpc.CallSite, mail *gschat.Mail) (retval uint64, err error) {
	return client.mailbox.mailhub.dispatchMail(client.agent.ID(), mail)
}

func (client *_Client) Sync(callSite *gorpc.CallSite, offset uint32, count uint32) (sync *gschat.Sync, err error) {

	if atomic.CompareAndSwapUint32(&client.syncFlag, 0, 1) {
		client.sync, err = client.mailbox.sync(client.api, offset, count)

		if err != nil {
			return nil, err
		}

		return &gschat.Sync{
			Offset: client.sync.offset,
			Count:  client.sync.count,
		}, nil
	}

	client.W("device %s with login use %s duplicate call sync", client.agent.ID(), client.mailbox.username)

	return nil, gschat.NewResourceBusy()
}

func (client *_Client) Fin(callSite *gorpc.CallSite, offset uint32) (err error) {

	if atomic.CompareAndSwapUint32(&client.syncFlag, 1, 0) {
		client.sync.Close()
		client.sync = nil
		return nil
	}

	return nil
}

func (client *_Client) Fetch(callSite *gorpc.CallSite, mailIDs []uint32) (retval []*gschat.Mail, err error) {
	return nil, nil
}
