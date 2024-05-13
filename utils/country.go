package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

const MaxQueryCount int = 20

var client = http.Client{
	// Timeout: time.Duration(time.Second * 3),
}

func GetCountry(ip string) (country string, err error) {
	respJson := make(map[string]interface{}, 1)
	// url := "https://ip.useragentinfo.com/json?ip="
	// url := "https://api.country.is/"
	url := "http://ip-api.com/json/"
	// resp, err := client.Get(fmt.Sprintf("https://api.country.is/%s", ip))
	resp, err := client.Get(fmt.Sprintf("%s%s?fields=country,countryCode,query", url, ip))

	if nil != err {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if nil != err {
		return
	}

	fmt.Printf("ip: %s, body: %s\n", ip, string(body))

	err = json.Unmarshal(body, &respJson)
	if nil != err || nil == respJson["country"] {
		// log.Println(ip)
		return
	}
	country = respJson["countryCode"].(string)

	return
}

func GetCountryWithCurl(ip string) (country string, err error) {
	respJson := make(map[string]interface{}, 1)
	// url := "https://ip.useragentinfo.com/json?ip="
	url := "http://ip-api.com/json/"
	// resp, err := client.Get(fmt.Sprintf("https://api.country.is/%s", ip))
	cmd := exec.Command("curl", fmt.Sprintf("%s%s?fields=country,countryCode,query", url, ip))
	cmd.Wait()
	data, err := cmd.Output()
	if nil != err {
		return
	}

	err = json.Unmarshal(data, &respJson)
	if nil != err {
		return
	}
	country = respJson["countryCode"].(string)

	return
}

func GetCountryBatch(ips ...string) (country_map map[string]string, err error) {
	url := "http://ip-api.com/batch?fields=country,countryCode,query"
	size := len(ips)
	// bytes.NewBuffer(ips)``
	country_map = map[string]string{}

	for i := 0; i < size; i += MaxQueryCount {
		var country_maps []map[string]string

		_ips := ips[i:min(size, i+MaxQueryCount)]
		for _, _ip := range _ips {
			country_map[_ip] = ""
		}
		data, err := json.Marshal(_ips)
		if nil != err {
			continue
		}

		resp, _ := http.Post(url, "text/plain;charset=UTF-8", bytes.NewBuffer(data))
		body, err := io.ReadAll(resp.Body)
		if nil != err {
			continue
		}
		err = json.Unmarshal(body, &country_maps)
		if nil != err {
			continue
		}

		// log.Println(string(body))

		for _, item := range country_maps {
			ip := item["query"]
			countryCode := item["countryCode"]
			country_map[ip] = countryCode
		}
	}
	return
}
