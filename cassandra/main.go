package main

import (
	"flag"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gschat/gocql"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

var hosts = flag.String("raddrs", "192.168.88.2|192.168.88.3|192.168.88.4", "cassandra cluster")

var conns = flag.Int("conns", 1000, "concurrent connections")

var rf = flag.Int("rf", 2, "data replication factor")

var duration = flag.Duration("duration", time.Millisecond*10, "update duration")

var applog = gslogger.Get("app")

func createKeySpace(cluster *gocql.ClusterConfig) error {
	cluster.Keyspace = "system"

	cluster.Timeout = 60 * time.Second

	var err error

	session, err := cluster.CreateSession()

	if err != nil {
		return gserrors.Newf(err, "create sessione rror")
	}

	defer session.Close()

	err = session.Query(`DROP KEYSPACE IF EXISTS bench`).Exec()
	if err != nil {
		panic(err)
	}

	err = session.Query(fmt.Sprintf(`CREATE KEYSPACE bench
	WITH replication = {
		'class' : 'SimpleStrategy',
		'replication_factor' : %d
	}`, *rf)).Exec()

	if err != nil {
		return gserrors.Newf(err, "create keyspace bench error")
	}

	return nil
}

func createSession(cluster *gocql.ClusterConfig) *gocql.Session {
	session, err := cluster.CreateSession()
	if err != nil {
		gserrors.Panicf(err, "create session error")
	}

	return session
}

func createTable(cluster *gocql.ClusterConfig) error {

	session := createSession(cluster)

	defer session.Close()

	if err := session.Query(`CREATE TABLE bench.SQID_TABLE(name varchar primary key,id   int)`).Exec(); err != nil {
		gserrors.Newf(err, "create SQID_TABLE error")
	}

	return nil
}

func main() {

	flag.Parse()

	defer func() {
		if e := recover(); e != nil {
			applog.E("%s", e)
		}

		gslogger.Join()
	}()

	cluster := gocql.NewCluster(strings.Split(*hosts, "|")...)

	cluster.ProtoVersion = 4
	cluster.CQLVersion = "3.3.1"

	if err := createKeySpace(cluster); err != nil {
		panic(err)
	}

	if err := createTable(cluster); err != nil {
		panic(err)
	}

	cluster.Keyspace = "bench"

	counter := uint32(0)

	for i := 0; i < *conns; i++ {

		name := fmt.Sprintf("test%d", i)

		go func() {
			session := createSession(cluster)
			defer session.Close()

			if err := session.Query(`INSERT INTO bench.SQID_TABLE (name,id) VALUES (?,?)`, name, 0).Exec(); err != nil {
				panic(err)
			}

			for i := 0; ; i++ {
				if err := session.Query(`UPDATE bench.SQID_TABLE SET id=? WHERE name = ?`, i, name).Exec(); err != nil {
					panic(err)
				}

				atomic.AddUint32(&counter, 1)

				<-time.After(*duration)
			}
		}()
	}

	for _ = range time.Tick(time.Second * 2) {
		applog.I("update speed %d/s", atomic.SwapUint32(&counter, 0)/2)
	}
}
