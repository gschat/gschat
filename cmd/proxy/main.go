// package gschat-proxy the gscaht proxy server
package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gschat/gschat/gw"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy"
	"github.com/gsrpc/gorpc"
)

var applog = gslogger.Get("profile")

var config = flag.String("config", "", "set the config file to load")

var frontend = flag.String("f", ":13516", "set gschat-proxy frontend listen address")

var backend = flag.String("b", ":15827", "set gschat-proxy backend listen address")

var log = flag.String("log", "", "write log file")

var level = flag.String("log-level", "", "write log file")

func main() {
	defer func() {
		if e := recover(); e != nil {
			applog.E("catch unhandle error:\n%s", e)
		}

		gslogger.Join()
	}()

	go func() {
		applog.E("%s", http.ListenAndServe("localhost:7000", nil))
	}()

	flag.Parse()

	gsconfig.Update("gsproxy.frontend.laddr", *frontend)
	gsconfig.Update("gsproxy.backend.laddr", *backend)

	gsconfig.Update("gschat.proxy.log", *log)

	gsconfig.Update("gschat.proxy.log.level", *level)

	// load config file
	if *config != "" {
		if err := gsconfig.LoadJSON(*config); err != nil {
			panic(gserrors.Newf(err, "load config file error :%s", *config))
		}
	}

	*log = gsconfig.String("gschat.proxy.log", "")

	*level = gsconfig.String("gschat.proxy.log.level", "")

	// open log file
	if *log != "" {

		fullpath, _ := filepath.Abs(*log)

		dir := filepath.Dir(fullpath)

		name := filepath.Base(fullpath)

		gslogger.SetLogDir(dir)

		gslogger.NewSink(gslogger.NewFilelog("gschat-proxy", name, gsconfig.Int64("gschat.proxy.log.maxsize", 0)))
	}

	// set log level
	if *level != "" {
		gslogger.NewFlags(gslogger.ParseLevel(*level))
	}

	improxy := gw.New()

	var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	applog.I("start gschat-proxy ...")

	gsproxy.BuildProxy(improxy).Build("gschat-proxy", eventLoop)

	applog.I("start gschat-proxy -- success")

	for _ = range time.Tick(20 * time.Second) {
		applog.I("\n%s", gorpc.PrintProfile())
	}
}
