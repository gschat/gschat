package cassandra

import (
	"testing"
	"time"

	"github.com/gschat/gocql"
)

var cluster = gocql.NewCluster("192.168.88.2", "192.168.88.3", "192.168.88.4")

func init() {
	cluster.ProtoVersion = 4
	cluster.CQLVersion = "3.3.1"
}

func TestCreateKeySpace(t *testing.T) {
	cluster.Keyspace = "system"

	cluster.Timeout = 60 * time.Second

	var err error

	session, err := cluster.CreateSession()

	if err != nil {
		t.Fatal(err)
	}

	defer session.Close()

	err = session.Query(`DROP KEYSPACE IF EXISTS bench`).Exec()
	if err != nil {
		panic(err)
	}

	err = session.Query(`CREATE KEYSPACE bench
	WITH replication = {
		'class' : 'SimpleStrategy',
		'replication_factor' : 2
	}`).Exec()

	if err != nil {
		t.Fatal(err)
	}
}

func createSession() *gocql.Session {
	cluster.Keyspace = "bench"

	session, err := cluster.CreateSession()
	if err != nil {
		panic(err)
	}

	return session
}

func TestCreateTable(t *testing.T) {
	session := createSession()
	defer session.Close()

	if err := session.Query(
		`CREATE TABLE bench.SQID_TABLE(
			name varchar primary key,
			id   int
		 )`,
	).Exec(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkUpdate(b *testing.B) {
	b.StopTimer()
	session := createSession()

	if err := session.Query(`INSERT INTO bench.SQID_TABLE (name,id)
		VALUES (?,?)`, "test", 0).Exec(); err != nil {
		b.Fatal("insert:", err)
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if err := session.Query(`UPDATE bench.SQID_TABLE SET id=? WHERE name = ?`, i, "test").Exec(); err != nil {
			b.Fatal("update:", err)
		}
	}

	b.StopTimer()
	session.Close()
	b.StartTimer()
}
