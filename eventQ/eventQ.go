package eventQ

import (
	"encoding/json"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/streadway/amqp"
)

// Q event queue facade
type Q interface {
	Close()
	DeviceOnline(device *gorpc.Device)
	DeviceOffline(device *gorpc.Device)
	UserOnline(username string, device *gorpc.Device)
	UserOffline(username string, device *gorpc.Device)
	PushBind(device *gorpc.Device, token []byte)
	PushUnbind(device *gorpc.Device, token []byte)
}

type _Service struct {
	gslogger.Log                  // eventQ
	conn         *amqp.Connection // amqp connection
	channel      *amqp.Channel    // amqp channel
	channelName  string           // channel name
	routeKey     string           // amq route key
}

// New .
func New(raddr string) (Q, error) {
	service := &_Service{
		Log:         gslogger.Get("eventQ"),
		channelName: gsconfig.String("eventQ.clientStatus.Q", "clientStatusChangedQueue"),
	}

	conn, err := amqp.Dial(raddr)

	if err != nil {
		return nil, gserrors.Newf(err, "dial amq server[%s] error", raddr)
	}

	service.conn = conn

	channel, err := conn.Channel()

	if err != nil {
		return nil, gserrors.Newf(err, "get amq server[%s] channel error", raddr)
	}

	q, err := channel.QueueDeclare(
		service.channelName,
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	service.routeKey = q.Name

	if err != nil {
		return nil, gserrors.Newf(err, "failed to declare an exchange on amq server[%s] error", raddr)
	}

	service.channel = channel

	return service, nil
}

func (service *_Service) Close() {
	service.channel.Close()
	service.conn.Close()
}

func (service *_Service) DeviceOnline(device *gorpc.Device) {
	status := &DeviceStatus{
		Device: device,
		Online: true,
	}

	content, err := json.Marshal(status)

	if err != nil {
		service.E("marshal device online error\n%s", err)
		return
	}

	service.send(content)
}

func (service *_Service) send(content []byte) {

	err := service.channel.Publish(
		"",               // exchange
		service.routeKey, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(content),
		})

	if err != nil {
		service.E("send message to amq error\n%s", err)
		return
	}
}

func (service *_Service) DeviceOffline(device *gorpc.Device) {

	status := &DeviceStatus{
		Device: device,
		Online: false,
	}

	content, err := json.Marshal(status)

	if err != nil {
		service.E("marshal device offline status error\n%s", err)
		return
	}

	service.send(content)
}

func (service *_Service) UserOnline(username string, device *gorpc.Device) {
	status := &UserStatus{
		Name:   username,
		Device: device,
		Online: true,
	}

	content, err := json.Marshal(status)

	if err != nil {
		service.E("marshal user online status error\n%s", err)
		return
	}

	service.send(content)
}

func (service *_Service) UserOffline(username string, device *gorpc.Device) {
	status := &UserStatus{
		Name:   username,
		Device: device,
		Online: false,
	}

	content, err := json.Marshal(status)

	if err != nil {
		service.E("marshal user offline status error\n%s", err)
		return
	}

	service.send(content)
}

func (service *_Service) PushBind(device *gorpc.Device, token []byte) {
	status := &PushToken{
		Device: device,
		Token:  string(token),
		Bind:   true,
	}

	content, err := json.Marshal(status)

	if err != nil {
		service.E("marshal push bind error\n%s", err)
		return
	}

	service.send(content)
}

func (service *_Service) PushUnbind(device *gorpc.Device, token []byte) {
	status := &PushToken{
		Device: device,
		Token:  string(token),
		Bind:   false,
	}

	content, err := json.Marshal(status)

	if err != nil {
		service.E("marshal push unbind error\n%s", err)
		return
	}

	service.send(content)
}
