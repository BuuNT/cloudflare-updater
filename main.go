package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
)

var (
	cloudflare = "https://api.cloudflare.com"
	httpbin    = "https://httpbin.org"
	re         = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`)
)

type config struct {
	Authorization string        `json:"authorization"`
	ZoneID        string        `json:"zoneID"`
	ZoneName      string        `json:"zoneName"`
	Proxied       bool          `json:"proxied"`
	Type          string        `json:"type"`
	Period        time.Duration `json:"period"`
}

func loadConfig(fname string) *config {
	file, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalln(err)
	}
	cfg := &config{}
	err = json.Unmarshal(file, &cfg)
	if err != nil {
		log.Fatalln(err)
	}
	return cfg

}

func getCurrentIP() string {
	client := http.Client{}
	req, err := http.NewRequest("GET", httpbin+"/ip", nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header = http.Header{
		"Content-Type": []string{"application/json"},
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	ip := string(re.Find(body))
	return ip

}

func getDNSRecord(config *config) (string, string) {
	client := http.Client{}
	req, err := http.NewRequest(
		"GET", cloudflare+"/client/v4/zones/"+config.ZoneID+
			"/dns_records?name="+config.ZoneName+"&type="+config.Type, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header = http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + config.Authorization},
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Fatalln(err)
	}
	recordID := string(regexp.MustCompile(`[a-f0-9]{32}`).Find(body))
	recordContent := string(re.Find(body))
	return recordID, recordContent
}

func updateDNSRecord(config *config, recordID, newIP string) string {
	client := http.Client{}
	stringJSON := fmt.Sprintf(
		`{"type":"%s", "name":"%s","proxied":%t, "content":"%s"}`,
		config.Type, config.ZoneName, config.Proxied, newIP)
	req, err := http.NewRequest(
		"PUT", cloudflare+"/client/v4/zones/"+config.ZoneID+
			"/dns_records/"+recordID, bytes.NewBuffer([]byte(stringJSON)))
	if err != nil {
		log.Fatalln(err)
	}

	req.Header = http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + config.Authorization},
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	return string(body)

}

func main() {
	cfg := loadConfig("config.json")
	for {
		currentIP := getCurrentIP()
		recordID, recordContent := getDNSRecord(cfg)
		if currentIP != "" && recordContent != "" {
			if currentIP == recordContent {
				log.Println("your IP address is the same, no update needed")
			} else {
				log.Println(updateDNSRecord(cfg, recordID, currentIP))
			}
		} else {
			log.Println("Cannot get IP Address or DNS Record")
		}
		time.Sleep(cfg.Period * time.Second)
	}
}
