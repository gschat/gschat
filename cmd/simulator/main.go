package main

import (
	"flag"
	"fmt"
	"math/big"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
	"github.com/gsrpc/gorpc/tcp"
)

var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

type _MockClient struct {
	gslogger.Log
	name     string
	imserver gschat.IMServer
	id       uint32
}

func newMockClient(name string) *_MockClient {
	return &_MockClient{
		Log:  gslogger.Get("mock-client"),
		name: name,
	}
}

func (mock *_MockClient) Push(mail *gschat.Mail) (err error) {

	mock.id = mail.SQID

	return nil
}

func (mock *_MockClient) Notify(seqid uint32) error {

	mock.I("%s notify %d", mock.name, seqid)

	im := mock.imserver

	if im != nil {
		err := im.Pull(mock.id)

		if err != nil {
			mock.E("im pull error :%s", err)
		}
	}

	return nil
}

func (mock *_MockClient) Connected(pipeline gorpc.Pipeline) {
	pipeline.AddService(gschat.MakeIMClient(uint16(gschat.ServiceTypeClient), mock))

	auth := gschat.BindIMAuth(uint16(gschat.ServiceTypeAuth), pipeline)

	_, err := auth.Login(mock.name, nil)

	if err != nil {
		mock.E("login(%s) error :%s", mock.name, err)
		pipeline.Close()
		return
	}

	im := gschat.BindIMServer(uint16(gschat.ServiceTypeIM), pipeline)

	mock.imserver = im

	go func() {

		wheel := pipeline.EventLoop().TimeWheel()

		ticker := wheel.NewTicker(5 * time.Second)

		defer ticker.Stop()

		for _ = range ticker.C {

			mail := gschat.NewMail()

			mail.Sender = mock.name

			mail.Receiver = mock.name

			mail.Type = gschat.MailTypeSingle

			mail.Content = "hello world"

			mock.I("put mail")
			_, err = im.Put(mail)
			mock.I("put mail -- success")

			if err != nil {
				mock.E("send message error :%s", err)
				return
			}

		}
	}()
}

func (mock *_MockClient) Disconnected(pipeline gorpc.Pipeline) {
	mock.imserver = nil
}

var ip = flag.String("s", "127.0.0.1:13516", "set the server ip address")

var clients = flag.Int("c", 1000, "set the simulator count")

var name = flag.String("n", "simulator", "set simulator name")

var clientBuilder *tcp.ClientBuilder

func createSimulator(name string) {
	G, _ := new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ := new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)

	device := gorpc.NewDevice()

	device.ID = name

	client := newMockClient(name)

	clientBuilder = tcp.BuildClient(
		gorpc.BuildPipeline(eventLoop).Handler(
			"profile",
			gorpc.ProfileHandler,
		).Handler(
			"dh-client",
			func() gorpc.Handler {
				return handler.NewCryptoClient(device, G, P)
			},
		).Handler(
			"heatbeat-client",
			func() gorpc.Handler {
				return handler.NewHeartbeatHandler(5 * time.Second)
			},
		),
	).EvtNewPipeline(
		tcp.EvtNewPipeline(client.Connected),
	).EvtClosePipeline(
		tcp.EvtClosePipeline(client.Disconnected),
	).Remote(*ip).Reconnect(5 * time.Second)

	clientBuilder.Connect(name)
}

func main() {

	log := gslogger.Get("profile")

	go func() {
		log.E("%s", http.ListenAndServe("localhost:7000", nil))
	}()

	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	// rand.Seed(time.Now().Unix())

	for i := 0; i < *clients; i++ {

		// <-time.After(time.Millisecond * time.Duration(rand.Intn(10)))
		createSimulator(fmt.Sprintf("%s(%d)", *name, i))
	}

	for _ = range time.Tick(20 * time.Second) {
		log.I("\n%s", gorpc.PrintProfile())
	}
}
