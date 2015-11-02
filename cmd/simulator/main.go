package main

import (
	"flag"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
)

var applog = gslogger.Get("gschat-simulator")
var raddr = flag.String("raddr", "localhost:13516", "set gschat-proxy listen address")
var clients = flag.Int("clients", 100, "set the simulate client number")
var prefix = flag.String("name", "simulator", "set the simulator login username's prefix")
var heartbeat = flag.Int("heartbeat", 5, "set heartbeat send timeout")
var level = flag.String("log-level", "", "set heartbeat send timeout")

var g = flag.String("G", "6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", "set the dh key'G")
var p = flag.String("P", "13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", "set the dh key'P")

func resolve(device *gorpc.Device) (*handler.DHKey, error) {

	_G, ok := new(big.Int).SetString(*g, 0)

	if !ok {
		return nil, gserrors.Newf(nil, "invalid G :%s", *g)
	}

	_P, ok := new(big.Int).SetString(*p, 0)

	if !ok {
		return nil, gserrors.Newf(nil, "invalid P :%s", *p)
	}

	return handler.NewDHKey(_G, _P), nil
}

func main() {
	flag.Parse()

	if *level != "" {
		gslogger.NewFlags(gslogger.ParseLevel(*level))
	}

	var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	for i := 0; i < *clients; i++ {
		newClient(eventLoop, fmt.Sprintf("%s:%d", *prefix, i), *raddr, time.Second*time.Duration(*heartbeat), handler.DHKeyResolve(resolve))
	}

	for _ = range time.Tick(20 * time.Second) {
		applog.I("\n%s", gorpc.PrintProfile())
	}

}
