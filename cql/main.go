package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/gschat/gschat"
	"github.com/gschat/gschat/mailhub"
	"github.com/gschat/gschat/mailhub/cql"
	"github.com/gsdocker/gslogger"
)

var clusters = flag.String("raddrs", "10.0.0.210", "cassandra cluster")

var users = flag.Int("users", 1024, "users")

var writers = flag.Int("writers", 10, "concurrency writers")

var content = flag.String("msg", "1111111111111111111", "write content")

var counter = flag.Int("num", 1024*10, "write messages of one user")

var sleep = flag.Duration("s", 100*time.Millisecond, "write sleep duration")

var applog = gslogger.Get("cql-insert")

func main() {
	flag.Parse()

	storage, err := cql.New(strings.Split(*clusters, "|")...)

	if err != nil {
		panic(err)
	}

	exit := make(chan bool, *writers)

	for i := 0; i < *writers; i++ {
		go write(exit, i, storage)
	}

	for i := 0; i < *writers; i++ {
		<-exit
	}
}

func write(exit chan bool, id int, storage mailhub.Storage) {
	c := *users / *writers

	mail := gschat.NewMail()

	mail.Sender = "cql-test"
	mail.Type = gschat.MailTypeSingle
	mail.Content = *content

	timer := time.NewTimer(*sleep)

	flag := 0

	for i := c * id; i < c*(id+1); i++ {

		mail.Receiver = fmt.Sprintf("username(%d)", id)

		for j := 0; j < *counter; j++ {

			flag++

			_, err := storage.Save(mail.Receiver, mail)

			if err != nil {
				applog.E("save mail(%s) error :%s", err)
			}

			if flag == 1024 {

				flag = 0

				timer.Reset(*sleep)

				<-timer.C
			}

		}
	}

	exit <- true
}
