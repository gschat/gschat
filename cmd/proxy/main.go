package main

import (
	"flag"
	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/gschat/gschat/eventQ"
	"github.com/gschat/gschat/gw"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
)

var listen = flag.String("l", ":13516", "set gschat-proxy ip address")

var tunnel = flag.String("t", ":15827", "set gschat-proxy tunnel listen ip address")

var pprof = flag.String("pprof", ":5000", "set the server ip address")

var qconfig = flag.String("Q", "amqp://user1:www.gridy.com@10.0.0.103:5672", "eventQ config")

func main() {

	flag.Parse()

	var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	var proxy gsproxy.Context

	defer func() {
		if e := recover(); e != nil {
			gslogger.Get("gschat-allinone").E("catch exception\n%s", e)
		}

		if proxy != nil {
			proxy.Close()
		}
	}()

	log := gslogger.Get("profile")

	go func() {
		log.E("%s", http.ListenAndServe(*pprof, nil))
	}()

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	Q, err := eventQ.New(*qconfig)

	if err != nil {
		panic(err)
	}

	proxy = gsproxy.BuildProxy(gw.NewIMProxy(Q)).AddrB(*tunnel).AddrF(*listen).Heartbeat(60*time.Second).Build("im-test-proxy", eventLoop)

	if err != nil {
		panic(err)
	}

	for _ = range time.Tick(20 * time.Second) {
		log.I("\n%s", gorpc.PrintProfile())
	}
}
