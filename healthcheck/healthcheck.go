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

func updateDataCenterHistory(dataCenter models.DataCenter, historyRecord models.DataCenterHistory, wg *sync.WaitGroup) {
	defer wg.Done()
	dataCenterKey := fmt.Sprintf("resource/datacenterhistory/%s", dataCenter.Domain)
	currentDataCenterJson, err := gtm_etcd.GetEntryByKey(dataCenterKey)
	if err != nil {
		fmt.Println("Error getting Data Center:", err)
		return
	}
	var currentDataCenterHistory []models.DataCenterHistory

	if len(currentDataCenterJson.Kvs) > 0 {
		err = json.Unmarshal([]byte(currentDataCenterJson.Kvs[len(currentDataCenterJson.Kvs)-1].Value), &currentDataCenterHistory)
		if err != nil {
			log.Fatalf("Error unmarshalling Data Center History JSON: %s", err)
		}
	}
	currentDataCenterHistory = append(currentDataCenterHistory, historyRecord)
	dataCenterJsonData, err := json.Marshal(currentDataCenterHistory)
	if err != nil {
		fmt.Println("Error marshalling Data Center History to JSON:", err)
		return
	}
	gtm_etcd.PutEntry(dataCenterKey, string(dataCenterJsonData))
}
func updateDataCenterStatus(dataCenter models.DataCenter, responseCode int, wg *sync.WaitGroup) {
	defer wg.Done()
	dataCenterKey := fmt.Sprintf("resource/datacenter/%s_%s", dataCenter.Domain, dataCenter.Name)
	currentDataCenterJson, err := gtm_etcd.GetEntryByKey(dataCenterKey)
	if err != nil {
		fmt.Println("Error getting Data Center:", err)
		return
	}
	var currentDataCenter models.DataCenter

	if len(currentDataCenterJson.Kvs) > 0 {
		err = json.Unmarshal([]byte(currentDataCenterJson.Kvs[len(currentDataCenterJson.Kvs)-1].Value), &currentDataCenter)
		if err != nil {
			log.Fatalf("Error unmarshalling currentDataCenter JSON: %s", err)
		}
	}

	if currentDataCenter.Status == "" || currentDataCenter.Status != dataCenter.Status {
		historyRecord := models.DataCenterHistory{
			DataCenterName: dataCenter.Name,
			HealthCheckUrl: dataCenter.HealthCheckUrl,
			Domain:         dataCenter.Domain,
			Status:         dataCenter.Status,
			ResponseCode:   responseCode,
			Reason:         "",
			TimeStamp:      time.Now().Format(time.RFC3339),
		}
		wg.Add(1)
		go updateDataCenterHistory(currentDataCenter, historyRecord, wg)

		if currentDataCenter.Status == "" {
			currentDataCenter = dataCenter
		} else {
			currentDataCenter.Status = dataCenter.Status
		}
		dataCenterJsonData, err := json.Marshal(currentDataCenter)
		if err != nil {
			fmt.Println("Error marshalling Data Center to JSON:", err)
			return
		}
		gtm_etcd.PutEntry(dataCenterKey, string(dataCenterJsonData))
	}
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

	var statusCode int
	if err == nil && resp != nil {
		statusCode = resp.StatusCode
	} else {
		statusCode = 0
	}
	wg.Add(1)
	go updateDataCenterStatus(dataCenter, statusCode, wg)
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
