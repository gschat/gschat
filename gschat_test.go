package gschat

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gschat/gschat/server"
	"github.com/gsdocker/gsactor"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/net"
)

var agentsysm gsactor.AgentSystemContext

var proxysysm gsproxy.Context

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

	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.DEBUG | gslogger.INFO)

	var err error

	proxysysm, err = gsproxy.BuildProxy(NewIMProxy).Run("im-test-proxy")

	if err != nil {
		panic(err)
	}

	<-time.After(time.Second)

	agentsysm, err = gsactor.BuildAgentSystem(func() gsactor.AgentSystem {
		return server.NewIMServer(10)
	}).Run("im-test-server")

	if err != nil {
		panic(err)
	}

}

func createDevice(name string) (gorpc.Sink, *net.TCPClient) {
	G, _ := new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ := new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)

	clientSink := gorpc.NewSink(name, time.Second*5, 1024, 10)

	device := gorpc.NewDevice()

	device.ID = "im-client-test"

	device.OSVersion = "1.0"

	return clientSink, net.NewTCPClient(
		"127.0.0.1:13512",
		gorpc.BuildPipeline().Handler(
			"dh-client",
			func() gorpc.Handler {
				return net.NewCryptoClient(device, G, P)
			},
		).Handler(
			"sink-client",
			func() gorpc.Handler {
				return clientSink
			},
		),
	).Connect(time.Second * 1)
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

	sink, client := createDevice("im-test-client")

	defer client.Close()

	mockClient := newMockClient()

	sink.Register(MakeIMClient(uint16(ServiceTypeClient), mockClient))

	auth := BindIMAuth(uint16(ServiceTypeAuth), sink)

	token, err := auth.Login("gschat@gmail.com", nil)

	if err != nil {
		t.Fatal(err)
	}

	im := BindIMServer(uint16(ServiceTypeIM), sink)

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

	sink, client := createDevice("im-test-client")

	defer client.Close()

	sink.Register(MakeIMClient(uint16(ServiceTypeClient), &_MockClient{}))

	auth := BindIMAuth(uint16(ServiceTypeAuth), sink)

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
