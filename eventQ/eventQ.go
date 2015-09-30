package eventQ

import (
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// Q event queue facade
type Q interface {
	DeviceOnline(device *gorpc.Device)
	DeviceOffline(device *gorpc.Device)
	UserOnline(username string, device *gorpc.Device)
	UserOffline(username string, device *gorpc.Device)
}

type _Service struct {
	gslogger.Log // eventQ
}
