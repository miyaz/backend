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
	Count     int64 `json:"count"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
	Reachable bool  `json:"reachable"`
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
	ni.UpdatedAt = time.Now().UnixNano()
}
func (ni *NodeInfo) getCreatedAt() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.CreatedAt
}
func (ni *NodeInfo) setCreatedAt(_time int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.CreatedAt = _time
}
func (ni *NodeInfo) getUpdatedAt() int64 {
	ni.RLock()
	defer ni.RUnlock()
	return ni.UpdatedAt
}
func (ni *NodeInfo) setUpdatedAt(_time int64) {
	ni.Lock()
	defer ni.Unlock()
	ni.UpdatedAt = _time
}
func (ni *NodeInfo) isReachable() bool {
	ni.RLock()
	defer ni.RUnlock()
	return ni.Reachable
}
func (ni *NodeInfo) setReachable(r bool) {
	ni.Lock()
	defer ni.Unlock()
	ni.Reachable = r
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
	ni.UpdatedAt = time.Now().UnixNano()
}

// Syncer ... Latest Data for Syncer
type Syncer struct {
	*sync.RWMutex
	SyncedAt int64             `json:"synced_at"`
	Nodes    map[int]*NodeInfo `json:"nodes"`
}

func (s *Syncer) setNow() {
	s.Lock()
	defer s.Unlock()
	s.SyncedAt = time.Now().UnixNano()
}
func (s *Syncer) getSyncedAt() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.SyncedAt
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
	&NodeInfo{&sync.RWMutex{}, 0, time.Now().UnixNano(), time.Now().UnixNano(), true},
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

	initSyncer()
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
		Node: NodeInfo{
			CreatedAt: store.node.getCreatedAt(),
			UpdatedAt: store.node.getUpdatedAt(),
			Count:     store.node.getCount(),
		},
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
			if node.getUpdatedAt() < inNode.getUpdatedAt() {
				syncer.Nodes[inPort].setCount(inNode.getCount())
				syncer.Nodes[inPort].setCreatedAt(inNode.getCreatedAt())
				syncer.Nodes[inPort].setUpdatedAt(inNode.getUpdatedAt())
				syncer.Nodes[inPort].setReachable(inNode.isReachable())
			}
		} else {
			syncer.Lock()
			inNode.setReachable(true)
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
	//ticker := time.NewTicker(time.Duration(sleep) * time.Millisecond)
	//defer ticker.Stop()
	for {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		//select {
		//case <-ticker.C:
		syncer.RLock()
		minUpdatedAt := time.Now().UnixNano()
		now := time.Now().UnixNano()
		destPort := listenPort
		for inPort := range syncer.Nodes {
			curUpdatedAt := syncer.Nodes[inPort].getUpdatedAt()
			if minUpdatedAt > curUpdatedAt && syncer.Nodes[inPort].isReachable() {
				minUpdatedAt = curUpdatedAt
				destPort = inPort
			}
		}
		syncer.RUnlock()
		if destPort == listenPort {
			continue
		}
		delta := (float64)(now-minUpdatedAt) / 1000000000
		syncer.Nodes[destPort].setReachable(execSyncer("http://localhost:"+strconv.Itoa(destPort)+"/syncer/", true))
		//clear()
		fmt.Println(string(getSyncerJSON()))
		fmt.Printf("port %d : %d - %d = %d (%f sec)\n", destPort, now, minUpdatedAt, now-minUpdatedAt, delta)
		//}
	}
}
func initSyncer() {
	nodes := getNodeList()
	reachableNodes := getReachableNodeList(nodes)
	syncer.Lock()
	defer syncer.Unlock()
	for i := 0; i < len(reachableNodes); i++ {
		portNum, _ := strconv.Atoi(reachableNodes[i])
		syncer.Nodes[portNum] = &NodeInfo{&sync.RWMutex{}, 0, time.Now().UnixNano(), time.Now().UnixNano(), true}
	}
}

func execSyncer(url string, merge bool) bool {
	c := &http.Client{
		Timeout: 500 * time.Millisecond,
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(getSyncerJSON()))
	if err != nil {
		fmt.Printf("failed to http.NewRequest: %v", err)
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	resp, err := c.Do(req)
	if err != nil {
		//fmt.Printf("failed to c.Do: %v", err)
		return false
	}
	defer resp.Body.Close()

	byteArray, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("failed to ioutil.ReadAll: %v", err)
	}
	wkSyncer := Syncer{} // RWMutex や map にアクセスしなければ初期化は不要
	if err := json.Unmarshal(byteArray, &wkSyncer); err != nil {
		fmt.Printf("failed to json.Unmarshal: %v", err)
	}
	if merge {
		mergeSyncer(&wkSyncer)
	}
	return true
}

func getNodeList() []string {
	// TODO: get eni list and crawle eni list
	var params []string
	for i := 0; i < 100; i++ {
		params = append(params, strconv.Itoa(9000+i))
	}
	return params
}

func getReachableNodeList(nodes []string) (reachableNodes []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	limiter := make(chan struct{}, 10)
	for i := 0; i < len(nodes); i++ {
		port := nodes[i]
		wg.Add(1)
		go func() {
			limiter <- struct{}{}
			defer wg.Done()
			//<-limiter
			reachable := execSyncer("http://localhost:"+port+"/syncer/", false)
			<-limiter
			mu.Lock()
			defer mu.Unlock()
			if reachable {
				reachableNodes = append(reachableNodes, port)
			}
		}()
	}
	wg.Wait()
	return
}
