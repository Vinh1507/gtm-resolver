package main

import (
	"encoding/json"
	"fmt"
	gtm_etcd "go-resolver/etcd"
	"go-resolver/initializers"
	"go-resolver/models"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/miekg/dns"
)

// Biến đếm số lần truy vấn cho từng region
var (
	region1Count = 0
	region2Count = 0
)

func checkHealth(url string) bool {
	client := http.Client{
		Timeout: 2 * time.Second, // Timeout sau 2 giây
	}
	resp, err := client.Get(url)

	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func getRegionIP(domain string) string {
	var gtmInfo models.Domain
	key := fmt.Sprintf("resource/gtm/%s", domain)
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
	region1IP := gtmInfo.DataCenters[0].IP
	region1Port := gtmInfo.DataCenters[0].Port
	region1Url := gtmInfo.DataCenters[0].HealthCheckUrl

	region2IP := gtmInfo.DataCenters[1].IP
	region2Port := gtmInfo.DataCenters[1].Port
	region2Url := gtmInfo.DataCenters[1].HealthCheckUrl

	region1HealthURL := fmt.Sprintf("http://%s:%d%s", region1IP, region1Port, region1Url)
	region2HealthURL := fmt.Sprintf("http://%s:%d%s", region2IP, region2Port, region2Url)

	fmt.Println("REGION2", region2HealthURL)

	// Các hằng số để xác định tỉ lệ phân phối
	region1Ratio := gtmInfo.DataCenters[0].Weight
	region2Ratio := gtmInfo.DataCenters[1].Weight

	// Health check của từng region
	region1Healthy := checkHealth(region1HealthURL)
	region2Healthy := checkHealth(region2HealthURL)

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
		return region1IP
	} else if !region1Healthy && region2Healthy {
		fmt.Println("Return region2\n==============\n")
		return region2IP
	} else if region1Healthy && region2Healthy {
		if region1RatioWeight < region2RatioWeight {
			region1Count++
			fmt.Println("Return region1\n==============\n")
			return region1IP
		}
		fmt.Println("Return region2\n==============\n")
		region2Count++
		return region2IP
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
