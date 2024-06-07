package utils

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	// "github.com/peanut996/CloudflareWarpSpeedTest/task"
)

const (
	defaultOutput         = "warp.csv"
	maxDelay              = 9999 * time.Millisecond
	minDelay              = 0 * time.Millisecond
	maxLossRate   float32 = 1.0
)

var (
	InputMaxDelay    = maxDelay
	InputMinDelay    = minDelay
	InputMaxLossRate = maxLossRate
	Output           = defaultOutput
	PrintNum         = 10
)

var (
	IPCountryMap map[string]string
	wg           = &sync.WaitGroup{}
)

func NoPrintResult() bool {
	return PrintNum == 0
}

func noOutput() bool {
	return Output == "" || Output == " "
}

type PingData struct {
	IP       *net.UDPAddr
	Sended   int
	Received int
	Delay    time.Duration
	Country  string
}

type CloudflareIPData struct {
	*PingData
	lossRate float32
}

func (cf *CloudflareIPData) getCountry() string {
	ip := cf.IP.IP.String()
	if cf.Country != "" {
		return cf.Country
	}

	cn, err := GetCountry(ip)
	if err != nil {
		log.Fatal(err)
	}
	cf.Country = cn
	return cn
}

func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.lossRate == 0 {
		pingLost := cf.Sended - cf.Received
		cf.lossRate = float32(pingLost) / float32(cf.Sended)
	}
	return cf.lossRate
}

func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 3)
	result[0] = cf.IP.String()
	result[1] = strconv.FormatFloat(float64(cf.getLossRate())*100, 'f', 0, 32) + "%"
	result[2] = strconv.FormatFloat(cf.Delay.Seconds()*1000, 'f', 2, 32)
	result = append(result, cf.Country)
	// result = append(result, "")
	// result = append(result, cf.getCountry())
	return result
}

func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	fp, err := os.Create(Output)
	if err != nil {
		log.Fatalf("Create file [%s] failed：%v", Output, err)
		return
	}
	defer fp.Close()
	w := csv.NewWriter(fp)
	_ = w.Write([]string{"IP:Port", "Loss", "Latency", "Country"})
	_ = w.WriteAll(convertToString(data))

	w.Flush()
}

func ExportAddresses(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	fp, err := os.Create("addressesapi.txt")
	if err != nil {
		log.Fatalf("Create file [%s] failed：%v", Output, err)
		return
	}
	defer fp.Close()
	w := csv.NewWriter(fp)

	size := len(data)

	// log.Println(convertToString(data))
	cached_ips := map[string]bool{}
	count := 0
	for idx, d := range data {
		fmt.Printf("idx: %d, size: %d\n", idx, size)
		if count == PrintNum {
			break
		}

		ip := d.IP.IP.String()
		if cached_ips[ip] == false {
			cached_ips[ip] = true
			record := fmt.Sprintf("%s:%d#%s", ip, 2052, d.getCountry())
			w.Write([]string{record})
			count++
		} else {
			fmt.Printf("already record %s \n", ip)
		}
	}
	w.Flush()
}

func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

type PingDelaySet []CloudflareIPData

func (s PingDelaySet) FilterDelay() (data PingDelaySet) {
	if InputMaxDelay > maxDelay || InputMinDelay < minDelay {
		return s
	}
	if InputMaxDelay == maxDelay && InputMinDelay == minDelay {
		return s
	}
	for _, v := range s {
		if v.Delay > InputMaxDelay {
			break
		}
		if v.Delay < InputMinDelay {
			continue
		}
		data = append(data, v)
	}
	return
}

func (s PingDelaySet) FilterLossRate() (data PingDelaySet) {
	if InputMaxLossRate >= maxLossRate {
		return s
	}
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate {
			break
		}
		data = append(data, v)
	}
	return
}

func (s PingDelaySet) Len() int {
	return len(s)
}
func (s PingDelaySet) Less(i, j int) bool {
	iRate, jRate := s[i].getLossRate(), s[j].getLossRate()
	if iRate != jRate {
		return iRate < jRate
	}
	return s[i].Delay < s[j].Delay
}
func (s PingDelaySet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func handler(delay CloudflareIPData, ch <-chan string) {
	ip := <-ch
	defer wg.Done()
	if IPCountryMap[ip] != "" {
		delay.Country = IPCountryMap[ip]
	} else {
		cn, err := GetCountry(ip)
		if nil != err {
			fmt.Errorf(err.Error())
		}
		delay.Country = cn
	}
}

func (s PingDelaySet) FetchCountries() {
	ch := make(chan string, 3)
	cached_ip := map[string]bool{}
	for _, delay := range s {
		ip := delay.IP.IP.String()
		if cached_ip[ip] == true {
			continue
		}
		ch <- ip
		wg.Add(1)
		go handler(delay, ch)
	}

	wg.Wait()
}

func (s PingDelaySet) Print() {
	if NoPrintResult() {
		return
	}
	if len(s) <= 0 {
		fmt.Println("\n[Info] The total number of IP addresses in the complete speed test results is 0, so skipping the output.")
		return
	}
	s.FetchCountries()
	// log.Println(country_map)
	dataString := convertToString(s)
	if len(dataString) < PrintNum {
		PrintNum = len(dataString)
	}
	headFormat := "\n%-24s%-9s%-10s%-10s\n"
	dataFormat := "%-25s%-8s%-10s%-10s\n"
	for i := 0; i < PrintNum; i++ {
		if len(dataString[i][0]) > 15 {
			headFormat = "\n%-44s%-9s%-10s%-10s\n"
			dataFormat = "%-45s%-8s%-10s%-10s\n"
		}
	}
	fmt.Printf(headFormat, "IP:Port", "Loss", "Latency", "Country")

	// cached := map[string]bool{}
	for i, idx := 0, 0; ; {
		if i == PrintNum {
			break
		}

		ip := strings.Split(dataString[idx][0], ":")[0]
		country := s[idx].Country
		cached := map[string]bool{}
		if !cached[ip] {
			cached[ip] = true
			fmt.Printf(dataFormat, dataString[idx][0], dataString[idx][1], dataString[idx][2], country)
			i += 1
		}

		idx += 1
	}
	if !noOutput() {
		fmt.Printf("\nComplete speed test results have been written to the %v file.\n", Output)
	}
}
