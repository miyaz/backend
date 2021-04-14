package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"sync"
)

// DataStore ... Variables that use mutex
type DataStore struct {
	*sync.RWMutex
	server    ServerInfo
	validator map[string]*regexp.Regexp
}

// ServerInfo ... information of server
type ServerInfo struct {
	Name string
	IP   string
	AZ   string
}

// QueryString ... QueryString Values
type QueryString struct {
	Sleep      string `json:"sleep,omitempty"`
	Size       string `json:"size,omitempty"`
	Status     string `json:"status,omitempty"`
	IfHost     string `json:"ifhost,omitempty"`
	IfAZ       string `json:"ifaz,omitempty"`
	IfServerIP string `json:"ifserverip,omitempty"`
	IfTargetIP string `json:"iftargetip,omitempty"`
	IfProxy1IP string `json:"ifproxy1ip,omitempty"`
	IfProxy2IP string `json:"ifproxy2ip,omitempty"`
	IfClientIP string `json:"ifclientip,omitempty"`
}

//  re := regexp.MustCompile("^[0-9]+([-,]?[0-9]+)*$")
//  fmt.Println(re.FindString(",3103"))

var store = &DataStore{&sync.RWMutex{}, ServerInfo{}, newValidator()}

func newValidator() map[string]*regexp.Regexp {
	const (
		regexpHostname = "^([a-zA-Z0-9-.]+)$"
		regexpStatus   = "^(200|400|403|404|500|502|503|504)$"
		regexpNumRange = "^([0-9]+)(?:-([0-9]+))?$"
		regexpNumComma = "^([0-9]+)(?:,([0-9]+))*$" // 2個以上はFindStringSubmatchで取得不可のためmatchしたらstrings.Split
		regexpAZone    = "^([a-z]{2}-[a-z]+-[1-9][a-d])$"
		regexpIPv6     = "^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$"
		regexpIPv4     = "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
	)
	validator := map[string]*regexp.Regexp{}
	validator["sleep"] = regexp.MustCompile(regexpNumRange)
	validator["size"] = regexp.MustCompile(regexpNumRange)
	validator["status"] = regexp.MustCompile(regexpStatus)
	validator["ifhost"] = regexp.MustCompile(regexpHostname)
	validator["ifaz"] = regexp.MustCompile(regexpAZone)
	validator["ifserverip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["iftargetip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["ifproxy1ip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["ifproxy2ip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	validator["ifclientip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
	return validator
}

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
	fmt.Fprintf(w, "hostname: %s\nserverip: %s\n\n", store.server.Name, store.server.IP)
	fmt.Fprintf(w, "path: %s\nquerystring: %s\n", r.URL.EscapedPath(), r.URL.Query().Encode())

	fmt.Fprintf(w, "[QueryString]\n")
	for _, kv := range sortkeyValues(r.URL.Query()) {
		fmt.Fprintf(w, "  %s = %s\n", kv.key, kv.value)
	}

	qs := validateQueryString(r.URL.Query())
	s, _ := json.Marshal(qs)
	fmt.Fprintf(w, "\n%s\n", string(s))
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

func validateQueryString(inQs map[string][]string) *QueryString {
	qs := &QueryString{}
	for _, kv := range sortkeyValues(inQs) {
		if re, ok := store.validator[kv.key]; ok {
			if len(re.FindStringSubmatch(kv.value)) > 0 {
				fmt.Printf("  valid %s = %s\n", kv.key, kv.value)
			} else {
				fmt.Printf("invalid %s = %s\n", kv.key, kv.value)
			}
		}
	}

	return qs
}
