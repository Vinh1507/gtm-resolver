package main

import (
	"encoding/json"
	"fmt"
	gtm_etcd "go-resolver/etcd"
	gtm_healthcheck "go-resolver/healthcheck"
	"go-resolver/initializers"
	"go-resolver/models"
	"log"
	"net"
	"os"

	"github.com/miekg/dns"
)

// Biến đếm số lần truy vấn cho từng region
var (
	region1Count = 0
	region2Count = 0
)

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

	// Các hằng số để xác định tỉ lệ phân phối
	region1Ratio := dataCenters[0].Weight
	region2Ratio := dataCenters[1].Weight

	// Health check của từng region
	region1Healthy := dataCenters[0].Status == "running"
	region2Healthy := dataCenters[1].Status == "running"

	// Tính toán tỉ lệ dựa trên số lần truy vấn
	totalRequests := region1Count + region2Count
	var region1RatioWeight, region2RatioWeight float64
	if totalRequests > 0 {
		region1RatioWeight = float64(region1Count) / float64(region1Ratio)
		region2RatioWeight = float64(region2Count) / float64(region2Ratio)

	}

	fmt.Printf("region1: %d, region2: %d\n", region1Count, region2Count)

	// Quyết định region dựa trên tỉ lệ
	if region1Healthy && !region2Healthy {
		fmt.Println("Return region1\n==============\n")
		return dataCenters[0].IP
	} else if !region1Healthy && region2Healthy {
		fmt.Println("Return region2\n==============\n")
		return dataCenters[1].IP
	} else if region1Healthy && region2Healthy {
		if region1RatioWeight < region2RatioWeight {
			region1Count++
			fmt.Println("Return region1\n==============\n")
			return dataCenters[0].IP
		}
		fmt.Println("Return region2\n==============\n")
		region2Count++
		return dataCenters[1].IP
	}

	return "" // Không có region nào khả dụng hoặc không đủ dữ liệu để quyết định
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	ttl := uint32(60)

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
}
