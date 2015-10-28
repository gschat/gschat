package is

import (
	"bytes"
	"sync"
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/tsdb"
	"github.com/gsdocker/gsagent"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// im agent received Q
type _IMAgentQ struct {
	sync.Mutex                   // locker
	gslogger.Log                 // mxin Log APIs
	name         string          // username
	client       gschat.IMClient // agent
	context      gsagent.Context //
	closeflag    chan bool       // close flag
	dataset      tsdb.DataSet    // send stream
	datasource   tsdb.DataSource // message queue
	device       *gorpc.Device   // device
}

func newAgentQ(name string, datasource tsdb.DataSource, context gsagent.Context) *_IMAgentQ {

	agentQ := &_IMAgentQ{
		Log:        gslogger.Get("im-agent-q"),
		name:       name,
		closeflag:  make(chan bool),
		context:    context,
		datasource: datasource,
		device:     context.ID(),
		client:     gschat.BindIMClient(uint16(gschat.ServiceTypeClient), context),
	}

	go agentQ.heartBeatLoop()

	return agentQ
}

func (agentQ *_IMAgentQ) heartBeatLoop() {

	tick := time.NewTicker(gsconfig.Seconds("gsim.heartbeat.timeout", 5))

	defer tick.Stop()

	// heartbeat

	for agentQ.dataset == nil {

		if version, ok := agentQ.datasource.CurrentVersion(agentQ.name); ok {
			agentQ.heartBeat(uint32(version))
		}

		select {
		case <-agentQ.closeflag:
			return
		case <-tick.C:
			if version, ok := agentQ.datasource.CurrentVersion(agentQ.name); ok {
				agentQ.heartBeat(uint32(version))
			}
		}
	}
}

func (agentQ *_IMAgentQ) sendLoop(dataset tsdb.DataSet) {

	agentQ.D("user %s login with %s create receive stream(%d)", agentQ.name, agentQ.device, dataset.MiniVersion())

	for {
		select {
		case <-agentQ.closeflag:

			agentQ.D("user %s login with %s receive stream(%d) -- closed", agentQ.name, agentQ.device, dataset.MiniVersion())

			dataset.Close()

			return
		case dbValue, ok := <-dataset.Stream():

			msg, err := gschat.ReadMail(bytes.NewBuffer(dbValue.Content))

			if err != nil {
				agentQ.D("[%s:%s] receive stream(%d), unmarshal mail -- failed\n%s", agentQ.name, agentQ.device, dataset.MiniVersion(), err)
				continue
			}

			if !ok {

				agentQ.D("user %s login with %s receive stream(%d) -- closed", agentQ.name, agentQ.device, dataset.MiniVersion())

				return
			}

			agentQ.D("%p [%s:%s] receive stream(%d), push mail(%d)", agentQ, agentQ.name, agentQ.device, dataset.MiniVersion(), msg.SQID)

			err = agentQ.client.Push(msg)

			if err != nil {
				agentQ.D("[%s:%s] receive stream(%d), push mail(%d) -- failed\n%s", agentQ.name, agentQ.device, dataset.MiniVersion(), msg.SQID, err)
				continue
			}

			agentQ.D("[%s:%s] receive stream(%d), push mail(%d) -- success", agentQ.name, agentQ.device, dataset.MiniVersion(), msg.SQID)
		}
	}
}

// send heart beat to agentQ
func (agentQ *_IMAgentQ) heartBeat(id uint32) {

	// send received queue heartbeat if id  > recvid id

	go func() {
		err := agentQ.client.Notify(id)

		if err != nil {
			agentQ.E("notify client error :%s", err)
		}

	}()
}

func (agentQ *_IMAgentQ) close() {

	close(agentQ.closeflag)

	agentQ.context.Close()
}

func (agentQ *_IMAgentQ) pull(id uint32) error {

	agentQ.Lock()
	defer agentQ.Unlock()

	if agentQ.dataset == nil {

		var err error

		agentQ.dataset, err = agentQ.datasource.Query(agentQ.name, uint64(id))

		if err != nil {
			agentQ.E("execute tsdb query(%s:%d) error:\n%s", agentQ.name, id, err)
			return err
		}

		go agentQ.sendLoop(agentQ.dataset)

	} else if agentQ.dataset.MiniVersion() > uint64(id) {

		agentQ.dataset.Close()

		var err error

		agentQ.dataset, err = agentQ.datasource.Query(agentQ.name, uint64(id))

		if err != nil {
			agentQ.E("execute tsdb query(%s:%d) error:\n%s", agentQ.name, id, err)
			return err
		}

		go agentQ.sendLoop(agentQ.dataset)
	}

	return nil
}
