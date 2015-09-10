package main

import (
	"flag"
	"fmt"
	"math/big"
	"time"

	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/net"
)

var ip = flag.String("s", "127.0.0.1:13512", "set the server ip address")

var clients = flag.Int("c", 1000, "set the simulator count")

func createDevice(name string) (gorpc.Sink, *net.TCPClient) {
	G, _ := new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ := new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)

	clientSink := gorpc.NewSink(name, time.Second*5, 1024, 10)

	device := gorpc.NewDevice()

	device.ID = "im-client-test" + name

	device.OSVersion = "1.0"

	return clientSink, net.NewTCPClient(
		name,
		*ip,
		gorpc.BuildPipeline().Handler(
			"dh-client",
			func() gorpc.Handler {
				return net.NewCryptoClient(device, G, P)
			},
		).Handler(
			"heartbeat-client",
			func() gorpc.Handler {
				return net.NewHeartbeatHandler(time.Second * 10)
			},
		).Handler(
			"sink-client",
			func() gorpc.Handler {
				return clientSink
			},
		),
	).Connect(time.Second * 1)
}

func main() {

	flag.Parse()

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	for i := 0; i < *clients; i++ {
		createDevice(fmt.Sprintf("simulator(%d)", i))
	}

	<-make(chan bool)
}
