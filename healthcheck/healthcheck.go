package gtm_healthcheck

import (
	"encoding/json"
	"fmt"
	gtm_etcd "go-resolver/etcd"
	"go-resolver/models"
	"log"
	"net/http"
	"sync"
	"time"
)

func updateDataCenterStatus(dataCenter models.DataCenter, wg *sync.WaitGroup) {
	defer wg.Done()
	dataCenterKey := fmt.Sprintf("resource/datacenter/%s_%s", dataCenter.Domain, dataCenter.Name)
	currentDataCenterJson, err := gtm_etcd.GetEntryByKey(dataCenterKey)
	if err != nil {
		fmt.Println("Error getting Data Center:", err)
		return
	}
	var currentDataCenter models.DataCenter
	err = json.Unmarshal([]byte(currentDataCenterJson.Kvs[len(currentDataCenterJson.Kvs)-1].Value), &currentDataCenter)
	if err != nil {
		log.Fatalf("Error unmarshalling currentDataCenter JSON: %s", err)
	}

	currentDataCenter.Status = dataCenter.Status
	dataCenterJsonData, err := json.Marshal(currentDataCenter)
	if err != nil {
		fmt.Println("Error marshalling Data Center to JSON:", err)
		return
	}
	gtm_etcd.PutEntry(dataCenterKey, string(dataCenterJsonData))
}
func healthCheck(dataCenter models.DataCenter, wg *sync.WaitGroup) {
	defer wg.Done()

	regionIP := dataCenter.IP
	regionPort := dataCenter.Port
	regionUrl := dataCenter.HealthCheckUrl
	url := fmt.Sprintf("http://%s:%d%s", regionIP, regionPort, regionUrl)

	resp, err := http.Get(url)
	if err != nil {
		// fmt.Printf("Health check failed for %s: %v\n", url, err)
		dataCenter.Status = "stop"
	} else {
		defer resp.Body.Close()
	}

	if err == nil && resp.StatusCode == http.StatusOK {
		// fmt.Printf("Health check OK for %s\n", url)
		dataCenter.Status = "running"
	} else {
		// fmt.Printf("Health check Failed for %s\n", url)
		dataCenter.Status = "stop"
	}

	wg.Add(1)
	go updateDataCenterStatus(dataCenter, wg)
}

func checkServers(dataCenters []models.DataCenter) {
	for {
		var wg sync.WaitGroup
		for _, dataCenter := range dataCenters {
			wg.Add(1)
			go healthCheck(dataCenter, &wg)

		}
		wg.Wait()
		time.Sleep(5 * time.Second)
	}
}

func StartCheckHealth() {
	dataCenters := make([]models.DataCenter, 0)
	entries, err := gtm_etcd.GetEntryByPrefix("resource/datacenter/")
	if err != nil || len(entries.Kvs) == 0 {
		fmt.Println("Cannot get gtm info")
		return
	}

	for _, entry := range entries.Kvs {
		var dc models.DataCenter
		jsonStr := entry.Value
		json.Unmarshal([]byte(jsonStr), &dc)
		dataCenters = append(dataCenters, dc)
	}

	fmt.Println(dataCenters)
	checkServers(dataCenters)
}
