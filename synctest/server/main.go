package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
	"github.com/gsrpc/gorpc/tcp"
)

var laddr = flag.String("l", ":13512", "server ip address")

var message = flag.String("m", "hello world ....", "mail content")

var log = gslogger.Get("app")

type _MockServer struct {
	client gschat.SyncClient
}

func newMockServer(pipeline gorpc.Pipeline) *_MockServer {
	return &_MockServer{
		client: gschat.BindSyncClient(0, pipeline),
	}
}

func (mock *_MockServer) Sync(callSite *gorpc.CallSite, num uint32) (err error) {

	log.I("Sync ....")

	go func() {
		for i := uint32(0); i < num; i++ {

			mail := gschat.NewMail()

			mail.Content = *message

			mock.client.Push(nil, mail)
		}
	}()

	return nil
}

func main() {

	flag.Parse()

	gslogger.NewFlags(gslogger.ERROR)

	eventLoop := gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	tcp.NewServer(
		gorpc.BuildPipeline(eventLoop).Handler(
			"state-handler",
			func() gorpc.Handler {
				return handler.NewStateHandler(func(pipeline gorpc.Pipeline, state gorpc.State) {
					if state == gorpc.StateConnected {

						log.I("connected ...")

						mock := newMockServer(pipeline)

						pipeline.AddService(gschat.MakeSyncServer(0, mock))

					}

				})
			},
		),
	).Listen(*laddr)
}
