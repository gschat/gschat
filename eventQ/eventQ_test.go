package eventQ

import (
	"testing"

	"github.com/gsrpc/gorpc"
)

func TestConn(t *testing.T) {
	Q, err := New("amqp://user1:www.gridy.com@10.0.0.103:5672")

	if err != nil {
		t.Fatal(err)
	}

	defer Q.Close()

	Q.DeviceOffline(gorpc.NewDevice())
}

func BenchmarkPush(t *testing.B) {
	t.StopTimer()
	Q, err := New("amqp://user1:www.gridy.com@10.0.0.103:5672")

	if err != nil {
		t.Fatal(err)
	}

	defer Q.Close()

	device := gorpc.NewDevice()

	device.ID = "channel test"

	t.StartTimer()

	for i := 0; i < t.N; i++ {
		Q.DeviceOnline(device)
	}
}
