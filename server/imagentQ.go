package server

import (
	"sync"
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mq"
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
	stream       *mq.Stream      // send stream
	fifo         mq.FIFO         // message queue
	device       *gorpc.Device   // device
}

func newAgentQ(name string, fifo mq.FIFO, context gsagent.Context) *_IMAgentQ {

	agentQ := &_IMAgentQ{
		Log:       gslogger.Get("im-agent-q"),
		name:      name,
		closeflag: make(chan bool),
		context:   context,
		fifo:      fifo,
		device:    context.ID(),
		client:    gschat.BindIMClient(uint16(gschat.ServiceTypeClient), context),
	}

	go agentQ.heartBeatLoop()

	return agentQ
}

func (agentQ *_IMAgentQ) heartBeatLoop() {

	tick := time.NewTicker(gsconfig.Seconds("gsim.heartbeat.timeout", 5))

	defer tick.Stop()

	// heartbeat

	for agentQ.stream == nil {

		agentQ.heartBeat(agentQ.fifo.MaxID())

		select {
		case <-agentQ.closeflag:
			return
		case <-tick.C:
			agentQ.heartBeat(agentQ.fifo.MaxID())
		}
	}
}

func (agentQ *_IMAgentQ) sendLoop(stream *mq.Stream) {

	agentQ.D("user %s login with %s create receive stream(%d)", agentQ.name, agentQ.device, stream.Offset)

	for {
		select {
		case <-agentQ.closeflag:

			agentQ.D("user %s login with %s receive stream(%d) -- closed", agentQ.name, agentQ.device, stream.Offset)

			stream.Close()

			return
		case msg, ok := <-stream.Chan:

			if !ok {

				agentQ.D("user %s login with %s receive stream(%d) -- closed", agentQ.name, agentQ.device, stream.Offset)

				return
			}

			agentQ.D("%p [%s:%s] receive stream(%d), push mail(%d)", agentQ, agentQ.name, agentQ.device, stream.Offset, msg.SQID)

			err := agentQ.client.Push(msg)

			if err != nil {
				agentQ.D("[%s:%s] receive stream(%d), push mail(%d) -- failed\n%s", agentQ.name, agentQ.device, stream.Offset, msg.SQID, err)
				continue
			}

			agentQ.D("[%s:%s] receive stream(%d), push mail(%d) -- success", agentQ.name, agentQ.device, stream.Offset, msg.SQID)
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

	if agentQ.stream == nil {

		agentQ.stream = agentQ.fifo.NewStream(id, gsconfig.Int("gsim.recvq.cached", 5))

		go agentQ.sendLoop(agentQ.stream)

	} else if agentQ.stream.Offset > id {

		agentQ.stream.Close()

		agentQ.stream = agentQ.fifo.NewStream(id, gsconfig.Int("gsim.recvq.cached", 5))
		go agentQ.sendLoop(agentQ.stream)
	}

	return nil
}
