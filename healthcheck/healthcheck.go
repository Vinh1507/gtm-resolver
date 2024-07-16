package gtm_healthcheck

import (
	"encoding/json"
	"fmt"
	gtm_etcd "go-resolver/etcd"
	"go-resolver/models"
	"net/http"
	"sync"
	"time"
)

func updateDataCenterStatus(dataCenter models.DataCenter, wg *sync.WaitGroup) {
	defer wg.Done()
	dataCenterKey := fmt.Sprintf("resource/datacenter/%s", dataCenter.Name)
	dataCenterJsonData, err := json.Marshal(dataCenter)
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
		fmt.Printf("Health check failed for %s: %v\n", url, err)
		dataCenter.Status = "stop"
	} else {
		defer resp.Body.Close()
	}

	if err == nil && resp.StatusCode == http.StatusOK {
		dataCenter.Status = "running"
	} else {
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
