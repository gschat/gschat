package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/gschat/gschat/is"
	"github.com/gsdocker/gsagent"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

type _Proxies []string

func (i *_Proxies) String() string {
	return fmt.Sprintf("%v", *i)
}

//实现Set接口,Set接口决定了如何解析flag的值
func (i *_Proxies) Set(value string) error {

	*i = append(*i, value)
	return nil
}

var proxies _Proxies

func main() {

	flag.Var(&proxies, "proxy", "set proxy ip")

	var pprof = flag.String("pprof", "localhost:6060", "set the server ip address")

	flag.Parse()

	var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	var agents gsagent.SystemContext

	defer func() {
		if e := recover(); e != nil {
			gslogger.Get("gschat-allinone").E("catch exception\n%s", e)
		}

		if agents != nil {
			agents.Close()
		}
	}()

	log := gslogger.Get("profile")

	go func() {
		log.E("%s", http.ListenAndServe(*pprof, nil))
	}()

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	agentsystem := gsagent.New("im-test-server", is.NewIMServer(10), eventLoop, 5*time.Second)

	if proxies == nil {
		proxies = []string{"localhost:15827"}
	}

	for _, proxy := range proxies {
		agentsystem.Connect("im-test-proxy"+proxy, proxy, 1024, 5*time.Second)
	}

	for _ = range time.Tick(20 * time.Second) {
		log.I("\n%s", gorpc.PrintProfile())
	}
}
