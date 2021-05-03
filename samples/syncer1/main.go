package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
	*sync.RWMutex
	host *HostInfo
	node *NodeInfo
}

// HostInfo ... information of host
type HostInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// NodeInfo ... information of node
type NodeInfo struct {
	*sync.RWMutex
	Count int64 `json:"count"`
	Time  int64 `json:"time"`
}

func (ni *NodeInfo) getCount() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Count
}
func (ni *NodeInfo) setCount(cnt int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.Count = cnt
}
func (ni *NodeInfo) countUp() {
	ni.Lock()
	defer ni.Unlock()
	ni.Count++
	ni.Time = time.Now().UnixNano()
}
func (ni *NodeInfo) getTime() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Time
}
func (ni *NodeInfo) getClone() *NodeInfo {
	ni.RLock()
	defer ni.RUnlock()
	node := *ni
	return &node
}
func (ni *NodeInfo) setNow() {
	ni.Lock()
	defer ni.Unlock()
	ni.Time = time.Now().UnixNano()
}

// Syncer ... Latest Data for Syncer
type Syncer struct {
	*sync.RWMutex
	Time  int64             `json:"time"`
	Nodes map[int]*NodeInfo `json:"nodes"`
}

func (s *Syncer) setNow() {
	s.Lock()
	defer s.Unlock()
	s.Time = time.Now().UnixNano()
}
func (s *Syncer) getTime() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.Time
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
	&sync.RWMutex{},
	&HostInfo{},
	&NodeInfo{&sync.RWMutex{}, 0, time.Now().UnixNano()},
}
var listenPort int
var syncer = Syncer{&sync.RWMutex{}, time.Now().UnixNano(), map[int]*NodeInfo{}}

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

	syncer.Nodes[listenPort] = store.node
	store.host.Name, _ = os.Hostname()
	store.host.IP = getIPAddress()
	go loopSyncer()

	http.HandleFunc("/", topHandler)
	http.HandleFunc("/syncer/", syncerHandler)
	srv := &http.Server{
		Addr:        ":" + strconv.Itoa(listenPort),
		IdleTimeout: 65 * time.Second,
	}
	log.Fatalln(srv.ListenAndServe())
}

func topHandler(w http.ResponseWriter, r *http.Request) {
	//w.WriteHeader(http.StatusNotFound)
	//w.Header().Set("Connection", "close")
	reqInfo := RequestInfo{
		Path:   r.URL.EscapedPath(),
		Query:  r.URL.Query().Encode(),
		Header: combineValues(r.Header),
		Node:   NodeInfo{Time: store.node.getTime(), Count: store.node.getCount()},
	}
	reqInfo.setIPAddresse(r)
	s, _ := json.MarshalIndent(reqInfo, "", "  ")
	fmt.Fprintf(w, "\n%s\n", string(s))

	syncer.RLock()
	defer syncer.RUnlock()
	syncerJSON, _ := json.MarshalIndent(syncer, "", "  ")
	fmt.Fprintf(w, "\n%s\n", string(syncerJSON))
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
	store.node.countUp()
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(http.StatusBadRequest)
	case http.MethodPost:
		body := r.Body
		defer body.Close()
		buf := new(bytes.Buffer)
		io.Copy(buf, body)
		//wkSyncer := Syncer{&sync.RWMutex{}, time.Now().UnixNano(), map[int]NodeInfo{}}
		wkSyncer := Syncer{} // RWMutex や map にアクセスしなければ初期化は不要
		if err := json.Unmarshal(buf.Bytes(), &wkSyncer); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Bad Request\n")
			fmt.Printf("failed to json.MarshalIndent: %v", err)
		} else {
			mergeSyncer(&wkSyncer)
			fmt.Fprintln(w, string(getSyncerJSON()))
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "Method not allowed.\n")
	}
}

func mergeSyncer(inSyncer *Syncer) {
	if inSyncer == nil || inSyncer.Nodes == nil {
		return
	}
	for inPort, inNode := range inSyncer.Nodes {
		if inPort == listenPort {
			continue
		}
		if inNode.RWMutex == nil {
			inNode.RWMutex = &sync.RWMutex{}
		}
		if node, ok := syncer.Nodes[inPort]; ok {
			if node.getTime() < inNode.getTime() {
				syncer.Nodes[inPort].setCount(inNode.getCount())
				syncer.Nodes[inPort].setNow()
			}
		} else {
			syncer.Lock()
			syncer.Nodes[inPort] = inNode
			syncer.Unlock()
		}
	}
}

func getSyncerJSON() []byte {
	updateSyncer()
	syncer.RLock()
	defer syncer.RUnlock()
	syncerJSON, err := json.MarshalIndent(syncer, "", "  ")
	if err != nil {
		fmt.Printf("failed to json.MarshalIndent: %v", err)
		return []byte{}
	}
	return syncerJSON
}
func updateSyncer() {
	syncer.setNow()
	store.node.setNow()
	syncer.Lock()
	defer syncer.Unlock()
	syncer.Nodes[listenPort] = store.node.getClone()
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

func loopSyncer() {
	sleep := 500
	for {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		syncer.RLock()
		//index := rand.Intn(len(syncer.Nodes))
		//count := 0
		minTime := time.Now().UnixNano()
		now := time.Now().UnixNano()
		destPort := listenPort
		for inPort := range syncer.Nodes {
			//if count == index {
			//	destPort = inPort
			//}
			//count++
			curTime := syncer.Nodes[inPort].getTime()
			if minTime > curTime {
				minTime = curTime
				destPort = inPort
			}
		}
		syncer.RUnlock()
		delta := (float64)(now-minTime) / 1000000000
		if destPort != listenPort {
			fmt.Printf("port %d : %d - %d = %d (%f sec)\n", destPort, now, minTime, now-minTime, delta)
			execSyncer("http://localhost:" + strconv.Itoa(destPort) + "/syncer/")
			//clear()
			fmt.Println(string(getSyncerJSON()))
			fmt.Println(destPort)
		}
	}
}

func execSyncer(url string) {
	c := &http.Client{
		Timeout: 500 * time.Millisecond,
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(getSyncerJSON()))
	if err != nil {
		fmt.Printf("failed to http.NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("failed to c.Do: %v", err)
	} else {
		defer resp.Body.Close()
		byteArray, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("failed to ioutil.ReadAll: %v", err)
		}
		wkSyncer := Syncer{} // RWMutex や map にアクセスしなければ初期化は不要
		if err := json.Unmarshal(byteArray, &wkSyncer); err != nil {
			fmt.Printf("failed to json.Unmarshal: %v", err)
		}
		mergeSyncer(&wkSyncer)
	}
}
