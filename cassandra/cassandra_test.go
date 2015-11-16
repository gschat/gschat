package cassandra

import (
	"time"

	"github.com/gschat/gocql"
)

var session *gocql.Session

// init the cassandra test env
func init() {
	cluster := gocql.NewCluster("218.90.199.5:9160", "218.90.199.7:9160", " 218.90.199.8:9160")

	cluster.Keyspace = "bench"

	cluster.Timeout = 5 * time.Second

	var err error

	session, err = cluster.CreateSession()

	if err != nil {
		panic(err)
	}
}
