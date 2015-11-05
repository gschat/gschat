package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gschat/gschat/mailhub"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsproxy/gsagent"
	"github.com/gsrpc/gorpc"
)

var applog = gslogger.Get("profile")

var config = flag.String("config", "", "set the config file to load")

var proxies = flag.String("p", "localhost:13516", "set gschat-proxy nodes")

var log = flag.String("log", "", "write log file")

var level = flag.String("log-level", "", "write log file")

var nodename = flag.String("n", "mailhub", "set the gschat-mailhub's node name")

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

	gsconfig.Update("gschat.mailhub.log", *log)

	gsconfig.Update("gschat.mailhub.log.level", *level)

	gsconfig.Update("gschat.mailhub.nodename", *nodename)

	gsconfig.Update("gschat.mailhub.proxies", *proxies)

	// load config file
	if *config != "" {
		if err := gsconfig.LoadJSON(*config); err != nil {
			panic(gserrors.Newf(err, "load config file error :%s", *config))
		}
	}

	*log = gsconfig.String("gschat.mailhub.log", "")

	*level = gsconfig.String("gschat.mailhub.log.level", "")

	*nodename = gsconfig.String("gschat.mailhub.nodename", "mailhub")

	*proxies = gsconfig.String("gschat.mailhub.proxies", "localhost:13516")

	// open log file
	if *log != "" {

		fullpath, _ := filepath.Abs(*log)

		dir := filepath.Dir(fullpath)

		name := filepath.Base(fullpath)

		gslogger.SetLogDir(dir)

		gslogger.NewSink(gslogger.NewFilelog("gschat-mailhub", name, gsconfig.Int64("gschat.mailhub.log.maxsize", 0)))
	}

	// set log level
	if *level != "" {
		gslogger.NewFlags(gslogger.ParseLevel(*level))
	}

	mailhub := mailhub.New(*nodename, nil)

	var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

	applog.I("start gschat-mailhub ...")

	context := gsagent.BuildAgent(mailhub).Build(*nodename, eventLoop)

	for _, proxy := range strings.Split(*proxies, "|") {
		context.Connect(proxy, proxy)
	}

	applog.I("start gschat-proxy -- success")

	for _ = range time.Tick(20 * time.Second) {
		applog.I("\n%s", gorpc.PrintProfile())
	}
}
