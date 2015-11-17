package cassandra

import (
	"time"

	"github.com/gschat/gocql"
)

var session *gocql.Session

// init the cassandra test env
func init() {
	cluster := gocql.NewCluster("218.90.199.5:7000", "218.90.199.7:7000", " 218.90.199.8:7000")

	cluster.Keyspace = "bench"

	cluster.Timeout = 5 * time.Second

	var err error

	session, err = cluster.CreateSession()

	if err != nil {
		panic(err)
	}
}
