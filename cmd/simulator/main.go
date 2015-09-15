package main

import (
	"flag"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
	"github.com/gsrpc/gorpc/tcp"
)

var ip = flag.String("s", "127.0.0.1:13516", "set the server ip address")

var clients = flag.Int("c", 1000, "set the simulator count")

var name = flag.String("n", "simulator", "set simulator name")

var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

var clientBuilder *tcp.ClientBuilder

func createDevice(name string) {
	G, _ := new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ := new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)

	clientBuilder = tcp.BuildClient(
		gorpc.BuildPipeline(eventLoop).Handler(
			"profile",
			gorpc.ProfileHandler,
		).Handler(
			"dh-client",
			func() gorpc.Handler {
				return handler.NewCryptoClient(gorpc.NewDevice(), G, P)
			},
		).Handler(
			"heatbeat-client",
			func() gorpc.Handler {
				return handler.NewHeartbeatHandler(5 * time.Second)
			},
		),
	).Remote(*ip).Reconnect(time.Second)

	clientBuilder.Connect(name)
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	// gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	// rand.Seed(time.Now().Unix())

	for i := 0; i < *clients; i++ {

		// <-time.After(time.Millisecond * time.Duration(rand.Intn(10)))
		createDevice(fmt.Sprintf("%s(%d)", *name, i))
	}

	<-make(chan bool)
}
