package cassandra

import (
	"time"

	"github.com/gschat/gocql"
)

var session *gocql.Session

// init the cassandra test env
func init() {
	cluster := gocql.NewCluster("192.168.88.2:7000", "192.168.88.3:7000", "192.168.88.4:7000")

	cluster.Keyspace = "bench"

	cluster.Timeout = 5 * time.Second

	var err error

	session, err = cluster.CreateSession()

	if err != nil {
		panic(err)
	}
}
