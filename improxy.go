package gschat

import "github.com/gsdocker/gsproxy"

type _IMProxy struct {
}

// IMSelector .
type IMSelector interface {
}

// IMProxyBuilder .
type IMProxyBuilder struct {
	selector IMSelector
}

// BuildIMProxy .
func BuildIMProxy() *IMProxyBuilder {
	return &IMProxyBuilder{}
}

// Selector .
func (builder *IMProxyBuilder) Selector(selector IMSelector) *IMProxyBuilder {
	builder.selector = selector
	return builder
}

func (proxy *_IMProxy) OpenProxy(context gsproxy.Context) {

}

func (proxy *_IMProxy) CreateServer(server gsproxy.Server) error {
	return nil
}

func (proxy *_IMProxy) CloseServer(server gsproxy.Server) {
}

func (proxy *_IMProxy) CreateDevice(device gsproxy.Device) error {
	return nil
}

func (proxy *_IMProxy) CloseDevice(device gsproxy.Device) {

}
