package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

var (
	region1IP = "192.168.144.132"
	region2IP = "192.168.144.136"

	region1HealthURL = fmt.Sprintf("http://%s:8000/health", region1IP)
	region2HealthURL = fmt.Sprintf("http://%s:8000/health", region2IP)
	ttl              = uint32(60) // TTL được đặt thành 60 giây

	// Các hằng số để xác định tỉ lệ phân phối
	region1Ratio = 6
	region2Ratio = 4
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

func getRegionIP() string {
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

	for _, q := range r.Question {
		switch q.Qtype {
		case dns.TypeA:
			ip := getRegionIP()
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

func main() {
	dns.HandleFunc("live.", handleDNSRequest)

	server := &dns.Server{Addr: ":8053", Net: "udp"}
	log.Printf("Starting DNS server on port 8053\n")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n", err.Error())
	}
}
