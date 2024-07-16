package initializers

import (
	"log"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var Etcd_cli *clientv3.Client

func ConnectToEtcd() {

	var err error

	enpoint := os.Getenv("ETCD_SERVER_ENDPOINT")
	Etcd_cli, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{enpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
}
