package server

import (
	"sync"
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mq"
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
	closeflag    chan bool       // close flag
	stream       *mq.Stream      // send stream
	fifo         mq.FIFO         // message queue
	device       *gorpc.Device   // device
}

func newAgentQ(name string, fifo mq.FIFO, agent *_IMAgent) *_IMAgentQ {

	agentQ := &_IMAgentQ{
		Log:       gslogger.Get("im-agent-q"),
		name:      name,
		client:    gschat.BindIMClient(uint16(gschat.ServiceTypeClient), agent),
		closeflag: make(chan bool),
		fifo:      fifo,
		device:    agent.ID(),
	}

	return agentQ
}

func (agentQ *_IMAgentQ) heartBeatLoop() {

	// heartbeat

	for agentQ.stream == nil {

		tick := time.Tick(gsconfig.Seconds("gsim.heartbeat.timeout", 5))

		agentQ.heartBeat(agentQ.fifo.MaxID())

		select {
		case <-agentQ.closeflag:
			return
		case <-tick:
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

			agentQ.D("[%s:%s] receive stream(%d), push mail(%d)", agentQ.name, agentQ.device, stream.Offset, msg.SQID)

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

	go agentQ.client.Notify(id)
}

func (agentQ *_IMAgentQ) close() {

	close(agentQ.closeflag)
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