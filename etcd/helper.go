package gtm_etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"go-resolver/initializers"
	"go-resolver/models"
	"log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func PutEntry(key string, value string) error {
	cli := initializers.Etcd_cli

	if cli == nil {
		log.Fatal("Etcd isn't connected!")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	_, err := cli.Put(ctx, key, value)
	cancel()
	return err
}

func GetEntryByKey(key string) (*clientv3.GetResponse, error) {
	cli := initializers.Etcd_cli

	if cli == nil {
		log.Fatal("Etcd isn't connected!")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	resp, err := cli.Get(ctx, key)
	cancel()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return resp, err
}

func GetEntryByPrefix(prefix string) (*clientv3.GetResponse, error) {
	cli := initializers.Etcd_cli

	if cli == nil {
		log.Fatal("Etcd isn't connected!")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatal(err)
	}

	return resp, err
}

func DeleteEntry(key string) error {
	cli := initializers.Etcd_cli

	if cli == nil {
		log.Fatal("Etcd isn't connected!")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	_, err := cli.Delete(ctx, key)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func UpdateDataCenterStatus(dataCenter models.DataCenter, wg *sync.WaitGroup) {
	defer wg.Done()
	dataCenterKey := fmt.Sprintf("resource/datacenter/%s_%s", dataCenter.Domain, dataCenter.Name)

	dataCenterJsonData, err := json.Marshal(dataCenter)
	if err != nil {
		fmt.Println("Error marshalling Data Center to JSON:", err)
		return
	}
	PutEntry(dataCenterKey, string(dataCenterJsonData))
}
