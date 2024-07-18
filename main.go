package main

import (
	"encoding/json"
	"fmt"
	gtm_etcd "go-resolver/etcd"
	"go-resolver/geo_location"
	gtm_healthcheck "go-resolver/healthcheck"
	"go-resolver/initializers"
	"go-resolver/models"
	"log"
	"net"
	"os"
	"sync"

	"github.com/miekg/dns"
)

var wg sync.WaitGroup

func getRegionIP(domain string) string {
	var gtmInfo models.Domain
	key := fmt.Sprintf("resource/domain/%s", domain)
	fmt.Println(key)
	entries, err := gtm_etcd.GetEntryByKey(key)

	if err != nil || len(entries.Kvs) == 0 {
		fmt.Println("Cannot get gtm info")
		return ""
	}

	jsonStr := entries.Kvs[len(entries.Kvs)-1].Value
	err = json.Unmarshal([]byte(jsonStr), &gtmInfo)

	if err != nil {
		fmt.Println("Cannot unmarshal gtm info")
		return ""
	}
	dataCenterKeys := gtmInfo.DataCenters
	dataCenters := make([]models.DataCenter, 0)

	for _, dataCenterKey := range dataCenterKeys {
		entries, err := gtm_etcd.GetEntryByKey(dataCenterKey)
		if err != nil {
			fmt.Printf("Failed to get data center %s", dataCenterKey)
		}
		jsonStr := entries.Kvs[len(entries.Kvs)-1].Value
		var dataCenter models.DataCenter
		err = json.Unmarshal([]byte(jsonStr), &dataCenter)

		if err != nil {
			fmt.Println("Cannot unmarshal data center info")
			return ""
		}
		dataCenters = append(dataCenters, dataCenter)
	}

	var selectedDataCenter models.DataCenter
	var minimumWeightRating float64 = 1000000000
	for _, dataCenter := range dataCenters {
		if dataCenter.Status == "running" {
			fmt.Println(dataCenter)
			regionWeightRating := float64(dataCenter.Count) / float64(dataCenter.Weight)
			if regionWeightRating < minimumWeightRating {
				minimumWeightRating = regionWeightRating
				selectedDataCenter = dataCenter
			}
		}
	}
	if selectedDataCenter.IP != "" {
		selectedDataCenter.Count += 1
		wg.Add(1)
		go gtm_etcd.UpdateDataCenterStatus(selectedDataCenter, &wg)
		return selectedDataCenter.IP
	}
	return "" // Không có region nào khả dụng hoặc không đủ dữ liệu để quyết định
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	ttl := uint32(60)
	sourceIP := w.RemoteAddr().(*net.UDPAddr).IP
	regionLocation, cityLocation, continent := geo_location.LookupGeoLocation(sourceIP.String())
	fmt.Println("GEO LOCATION: ", sourceIP, regionLocation, cityLocation, continent)

	for _, q := range r.Question {
		domain := q.Name
		if len(domain) > 0 {
			domain = domain[:len(domain)-1]
		}
		switch q.Qtype {
		case dns.TypeA:
			ip := getRegionIP(domain)
			if ip == "" {
				// Nếu không có region nào khả dụng, trả về lỗi
				msg.SetRcode(r, dns.RcodeServerFailure)
				w.WriteMsg(msg)
				return
			}
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    ttl, // TTL được đặt thành 60 giây
				},
				A: net.ParseIP(ip),
			}
			msg.Answer = append(msg.Answer, rr)
		}
	}

	w.WriteMsg(msg)
}

func init() {
	initializers.LoadEnvVariables()
	initializers.ConnectToEtcd()
	go gtm_healthcheck.StartCheckHealth()
}

func main() {
	dns.HandleFunc("live.", handleDNSRequest)

	port := os.Getenv("PORT")
	server := &dns.Server{Addr: port, Net: "udp"}
	log.Printf("Starting DNS server on port %s\n", port)

	err := server.ListenAndServe()

	if err != nil {
		log.Fatalf("Failed to start server: %s\n", err.Error())
	}
	wg.Wait()
}
