package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DataStore ... Variables that use mutex
type DataStore struct {
	host HostInfo
	node NodeInfo
}

// HostInfo ... information of host
type HostInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// NodeInfo ... information of node
type NodeInfo struct {
	*sync.RWMutex
	Count    int64 `json:"count"`
	UpdateAt int64 `json:"update_at"`
}

func (ni *NodeInfo) getCount() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Count
}
func (ni *NodeInfo) countUp() {
	ni.Lock()
	defer ni.Unlock()
	ni.Count++
	ni.UpdateAt = time.Now().UnixNano()
}
func (ni *NodeInfo) getUpdateAt() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.UpdateAt
}
func (ni *NodeInfo) setNow() {
	ni.Lock()
	defer ni.Unlock()
	ni.UpdateAt = time.Now().UnixNano()
}

// RequestInfo ... information of request
type RequestInfo struct {
	Path     string            `json:"path"`
	Query    string            `json:"querystring,omitempty"`
	Header   map[string]string `json:"header"`
	ClientIP string            `json:"clientip"`
	TargetIP string            `json:"targetip"`
	Node     NodeInfo          `json:"node"`
}

// ResponseInfo ... information of response
type ResponseInfo struct {
	Host    HostInfo    `json:"host"`
	Node    NodeInfo    `json:"node"`
	Request RequestInfo `json:"request"`
}

var store = &DataStore{
	HostInfo{},
	NodeInfo{&sync.RWMutex{}, 0, time.Now().UnixNano()},
}
var listenPort int
var nodes = map[int]*NodeInfo{}

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
	flag.IntVar(&listenPort, "port", 9000, "listen port")
	flag.Parse()
	fmt.Println("Listen Port : ", listenPort)

	nodes[listenPort] = &store.node
	store.host.Name, _ = os.Hostname()
	store.host.IP = getIPAddress()
	http.HandleFunc("/", topHandler)
	http.HandleFunc("/syncer/", syncerHandler)
	srv := &http.Server{Addr: ":" + strconv.Itoa(listenPort)}
	log.Fatalln(srv.ListenAndServe())
}

func topHandler(w http.ResponseWriter, r *http.Request) {
	//w.WriteHeader(http.StatusNotFound)
	store.node.countUp()
	reqInfo := RequestInfo{
		Path:   r.URL.EscapedPath(),
		Query:  r.URL.Query().Encode(),
		Header: combineValues(r.Header),
		Node:   store.node,
	}
	reqInfo.setIPAddresse(r)

	nodesStr, _ := json.MarshalIndent(nodes, "", "  ")
	fmt.Fprintf(w, "\n%s\n", string(nodesStr))

	s, _ := json.MarshalIndent(reqInfo, "", "  ")
	fmt.Fprintf(w, "\n%s\n", string(s))

	inputStr := []byte(`{ "8000": { "count": 6, "update_at": 1619351214434379100 } }`)
	var inputNodes map[int]NodeInfo
	if err := json.Unmarshal(inputStr, &inputNodes); err != nil {
		panic(err)
	}
	inputNodes[listenPort] = store.node
	n, _ := json.MarshalIndent(inputNodes, "", "  ")
	fmt.Fprintf(w, "\n%s\n", string(n))
}

func clear() {
	cmd := exec.Command("clear") //Linux example, its tested
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func combineValues(input map[string][]string) map[string]string {
	output := map[string]string{}
	for key := range input {
		output[key] = strings.Join(input[key], ", ")
	}
	return output
}

func syncerHandler(w http.ResponseWriter, r *http.Request) {
	s := "syncer"
	fmt.Fprintf(w, "\n%s\n", string(s))
}

func (reqInfo *RequestInfo) setIPAddresse(r *http.Request) {
	//reqInfo.TargetIP = extractIPAddress(r.Host)
	reqInfo.TargetIP = r.Host
	reqInfo.ClientIP = extractIPAddress(r.RemoteAddr)
}

func extractIPAddress(ipport string) string {
	var ipaddr string
	if strings.HasPrefix(ipport, "[") {
		ipaddr = strings.Join(strings.Split(ipport, ":")[:len(strings.Split(ipport, ":"))-1], ":")
		ipaddr = strings.Trim(ipaddr, "[]")
	} else {
		ipaddr = strings.Split(ipport, ":")[0]
	}
	return ipaddr
}
