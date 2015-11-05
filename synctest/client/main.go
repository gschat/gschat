package main

import (
	"flag"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
	"github.com/gsrpc/gorpc/tcp"
)

var raddr = flag.String("r", "localhost:13512", "server ip address")
var num = flag.Int("n", 10000, "sync mails")

var log = gslogger.Get("app")

type _MockClient struct {
	closed chan bool
	num    int32
	start  time.Time
}

func newMockClient(num int32) *_MockClient {
	return &_MockClient{
		closed: make(chan bool),
		num:    num,
	}
}

func (mock *_MockClient) Start() {
	mock.start = time.Now()
}

func (mock *_MockClient) Push(callSite *gorpc.CallSite, mail *gschat.Mail) (err error) {

	if atomic.AddInt32(&mock.num, -1) == 0 {

		log.I("sync %d mails use time :%s", *num, time.Now().Sub(mock.start))

		mock.closed <- true
	}

	return nil
}

func (mock *_MockClient) Join() {
	<-mock.closed
}

func main() {

	flag.Parse()

	gslogger.NewFlags(gslogger.ERROR | gslogger.INFO)

	defer func() {
		if e := recover(); e != nil {
			log.E("catch unhandle exception :%s", e)
		}

		gslogger.Join()
	}()

	mock := newMockClient(int32(*num))

	eventLoop := gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	_, err := tcp.BuildClient(gorpc.BuildPipeline(eventLoop).Handler("state",
		func() gorpc.Handler {
			return handler.NewStateHandler(func(pipeline gorpc.Pipeline, state gorpc.State) {

				if state == gorpc.StateConnected {

					pipeline.AddService(gschat.MakeSyncClient(0, mock))

					server := gschat.BindSyncServer(0, pipeline)

					mock.Start()

					go server.Sync(nil, uint32(*num))
				}

			})
		},
	)).Remote(*raddr).Connect("test")

	if err != nil {
		panic(err)
	}

	mock.Join()
}
