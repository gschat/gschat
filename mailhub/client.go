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
	wrapper.api = gschat.BindClient(uint16(gschat.ServiceClient), agent)

	// register mailhub interface
	agent.AddService(gschat.MakeMailHub(uint16(gschat.ServiceMailHub), wrapper))

	go wrapper.notifyLoop()

	return wrapper
}

func (client *_Client) Close() {
	close(client.closed)
}

func (client *_Client) notifyLoop() {

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

func (client *_Client) Sync(callSite *gorpc.CallSite, offset uint32, count uint32) (retval uint32, err error) {

	if atomic.CompareAndSwapUint32(&client.syncFlag, 0, 1) {
		client.sync, err = client.mailbox.sync(client.api, offset, count)

		if err != nil {
			return 0, err
		}

		return client.sync.count, nil
	}

	client.W("device %s with login use %s duplicate call sync", client.agent.ID(), client.mailbox.username)

	return 0, gschat.NewResourceBusy()
}

func (client *_Client) Fin(callSite *gorpc.CallSite, offset uint32) (err error) {

	if atomic.CompareAndSwapUint32(&client.syncFlag, 1, 0) {
		client.sync.Close()
		return nil
	}

	return nil
}
