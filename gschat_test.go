package gschat

import (
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gschat/gschat/server"
	"github.com/gsdocker/gsagent"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
	"github.com/gsrpc/gorpc/tcp"
)

var agentsysm gsagent.SystemContext

var proxysysm gsproxy.Context
var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

var log = gslogger.Get("test")

func init() {

	filepath.Walk("./", func(newpath string, info os.FileInfo, err error) error {

		if info.IsDir() && newpath != "./" {

			return filepath.SkipDir
		}

		if filepath.Ext(info.Name()) == ".storage" {

			os.Remove(newpath)
		}

		if filepath.Ext(info.Name()) == ".db" {
			os.Remove(newpath)
		}

		return err
	})

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.INFO)

	proxysysm = gsproxy.BuildProxy(NewIMProxy()).Build("im-test-proxy", eventLoop)

	<-time.After(time.Second)

	agentsysm = gsagent.New("im-test-server", server.NewIMServer(10), eventLoop, 5)

	agentsysm.Connect("im-test-proxy", "127.0.0.1:15827", 1024, 5*time.Second)

}

func createDevice(name string) (tcp.Client, error) {
	G, _ := new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ := new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)

	device := gorpc.NewDevice()

	device.ID = name

	clientBuilder := tcp.BuildClient(
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
	)

	return clientBuilder.Connect(name)
}

type _MockClient struct {
	Mails chan *Mail
}

func newMockClient() *_MockClient {
	return &_MockClient{
		Mails: make(chan *Mail),
	}
}

func (mock *_MockClient) Push(mail *Mail) (err error) {

	mock.Mails <- mail

	return nil
}

func (mock *_MockClient) Notify(SQID uint32) (err error) {
	return nil
}

func TestIM(t *testing.T) {

	client, _ := createDevice("im-test-client")

	defer client.Close()

	mockClient := newMockClient()

	client.Pipeline().AddService(MakeIMClient(uint16(ServiceTypeClient), mockClient))

	auth := BindIMAuth(uint16(ServiceTypeAuth), client.Pipeline())

	token, err := auth.Login("gschat@gmail.com", nil)

	if err != nil {
		t.Fatal(err)
	}

	im := BindIMServer(uint16(ServiceTypeIM), client.Pipeline())

	mail := NewMail()

	mail.Sender = "gschat@gmail.com"

	mail.Receiver = "gschat@gmail.com"

	mail.Type = MailTypeSingle

	mail.Content = "hello world"

	_, err = im.Put(mail)

	if err != nil {
		t.Fatal(err)
	}

	err = im.Pull(0)

	if err != nil {
		t.Fatal(err)
	}

	if (<-mockClient.Mails).Content != mail.Content {
		t.Fatal("check mail content error")
	}

	err = auth.Logoff(token)

	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkLoginLogoff(t *testing.B) {

	t.StopTimer()

	client, _ := createDevice("im-test-client")

	defer client.Close()

	mockClient := newMockClient()

	client.Pipeline().AddService(MakeIMClient(uint16(ServiceTypeClient), mockClient))

	auth := BindIMAuth(uint16(ServiceTypeAuth), client.Pipeline())

	t.StartTimer()

	for i := 0; i < t.N; i++ {
		token, err := auth.Login("gschat@gmail.com", nil)

		if err != nil {
			t.Fatal(err)
		}

		err = auth.Logoff(token)

		if err != nil {
			t.Fatal(err)
		}

	}
}
