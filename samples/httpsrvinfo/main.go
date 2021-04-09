package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
)

// DataStore ... Variables that use mutex
type DataStore struct {
	*sync.RWMutex
	server  ServerInfo
	current ResourceData
	target  ResourceData
}

// ServerInfo ... information of server
type ServerInfo struct {
	Name string
	IP   string
	AZ   string
}

// ResourceData ... OS Resource Data
type ResourceData struct {
	CPUUsage float64 `json:"cpu"`
	MemUsage float64 `json:"mem"`
}

// QueryString ... QueryString Values
type QueryString struct {
	Sleep      string `json:"sleep"`
	Size       string `json:"size"`
	Status     string `json:"status"`
	IfHost     string `json:"ifhost"`
	IfAZ       string `json:"ifaz"`
	IfTargetIP string `json:"iftargetip"`
	IfLBNodeIP string `json:"iflbnodeip"`
	IfClientIP string `json:"ifclientip"`
}

// HandleInfo ... handle info
type HandleInfo struct {
	MSleep           int64  `json:"sleep"`
	ResponceSize     int64  `json:"size"`
	StatusCode       int64  `json:"status"`
	AvailabilityZone string `json:"az"`
	TargetIP         string `json:"targetip"`
	LBNodeIP         string `json:"lbnodeip"`
	ClientIP         string `json:"clientip"`
}

var store = &DataStore{&sync.RWMutex{}, ServerInfo{}, ResourceData{}, ResourceData{}}

func getIPAddress() string {
	var currentIP string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println("Current IP address : ", ipnet.IP.String())
				currentIP = ipnet.IP.String()
			}
		}
	}
	return currentIP
}

func main() {
	store.Lock()
	store.server.Name, _ = os.Hostname()
	store.server.IP = getIPAddress()
	store.Unlock()
	http.HandleFunc("/", handler)
	srv := &http.Server{Addr: ":9000"}
	log.Fatalln(srv.ListenAndServe())
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "host: %s\nip: %s\n\n", store.server.Name, store.server.IP)
	fmt.Fprintf(w, "path:  %s\nquerystring: %s\n", r.URL.EscapedPath(), r.URL.Query().Encode())

	fmt.Fprintf(w, "[Headers]\n")
	for _, kv := range sortkeyValues(r.Header) {
		fmt.Fprintf(w, "%s = %s\n", kv.key, kv.value)
	}
	fmt.Fprintf(w, "[QueryString]\n")
	for _, kv := range sortkeyValues(r.URL.Query()) {
		fmt.Fprintf(w, "%s = %s\n", kv.key, kv.value)
	}
	fmt.Fprintf(w, "%v %v", store.server, store.current)
}

type keyValue struct {
	key   string
	value string
}

func sortkeyValues(input map[string][]string) (output []keyValue) {
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if len(keys) > 1 {
			sort.Strings(input[key])
		}
		for _, val := range input[key] {
			output = append(output, keyValue{key: key, value: val})
		}
	}
	return
}

func (ds *DataStore) setCPU(value float64) {
	ds.Lock()
	defer ds.Unlock()
	ds.current.CPUUsage = value
}
func (ds *DataStore) getCPU() float64 {
	ds.RLock()
	defer ds.RUnlock()
	return ds.current.CPUUsage
}

func (ds *DataStore) setMemory(value float64) {
	ds.Lock()
	defer ds.Unlock()
	ds.current.MemUsage = value
}
func (ds *DataStore) getMemory() float64 {
	ds.RLock()
	defer ds.RUnlock()
	return ds.current.MemUsage
}
