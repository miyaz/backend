package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

// DataStore ... Variables that use mutex
type DataStore struct {
	host      HostInfo
	resource  ResourceInfo
	validator map[string]*regexp.Regexp
}

// HostInfo ... information of host
type HostInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	AZ   string `json:"az,omitempty"`
}

// ResourceInfo ... information of os resource
type ResourceInfo struct {
	CPU    ResourceUsage `json:"cpu"`
	Memory ResourceUsage `json:"memory"`
}

// ResourceUsage ... information of os resource usage
type ResourceUsage struct {
	*sync.RWMutex
	Target  float64 `json:"target"`
	Current float64 `json:"current"`
}

func (ru *ResourceUsage) getTarget() float64 {
	ru.RLock()
	defer ru.RUnlock()
	return ru.Target
}
func (ru *ResourceUsage) setTarget(value float64) {
	if ru.getTarget() != value {
		ru.Lock()
		defer ru.Unlock()
		ru.Target = value
	}
}
func (ru *ResourceUsage) getCurrent() float64 {
	ru.RLock()
	defer ru.RUnlock()
	return ru.Current
}
func (ru *ResourceUsage) setCurrent(value float64) {
	if ru.getCurrent() != value {
		ru.Lock()
		defer ru.Unlock()
		ru.Current = value
	}
}

// RequestInfo ... information of request
type RequestInfo struct {
	Path     string            `json:"path"`
	Query    string            `json:"querystring,omitempty"`
	Header   map[string]string `json:"header"`
	ClientIP string            `json:"clientip"`
	Proxy1IP string            `json:"proxy1ip,omitempty"`
	Proxy2IP string            `json:"proxy2ip,omitempty"`
	TargetIP string            `json:"targetip"`
}

// Direction ... information of directions
type Direction struct {
	Input  *QueryString `json:"input"`
	Action *QueryString `json:"action"`
}

// ResponseInfo ... information of response
type ResponseInfo struct {
	Host      HostInfo     `json:"host"`
	Resource  ResourceInfo `json:"resource"`
	Request   RequestInfo  `json:"request"`
	Direction Direction    `json:"direction"`
}

var store = &DataStore{
	HostInfo{},
	ResourceInfo{ResourceUsage{&sync.RWMutex{}, 0, 0}, ResourceUsage{&sync.RWMutex{}, 0, 0}},
	newValidator(),
}

// QueryString ... QueryString Values
type QueryString struct {
	CPU          string `json:"cpu,omitempty"`
	Memory       string `json:"memory,omitempty"`
	Sleep        string `json:"sleep,omitempty"`
	Size         string `json:"size,omitempty"`
	Status       string `json:"status,omitempty"`
	existsAction bool
	needsAction  bool
	IfClientIP   string `json:"ifclientip,omitempty"`
	IfProxy1IP   string `json:"ifproxy1ip,omitempty"`
	IfProxy2IP   string `json:"ifproxy2ip,omitempty"`
	IfTargetIP   string `json:"iftargetip,omitempty"`
	IfHostIP     string `json:"ifhostip,omitempty"`
	IfHost       string `json:"ifhost,omitempty"`
	IfAZ         string `json:"ifaz,omitempty"`
}

func (qs *QueryString) setValue(key, value string) {
	switch key {
	case "cpu":
		qs.CPU = value
	case "memory":
		qs.Memory = value
	case "sleep":
		qs.Sleep = value
	case "size":
		qs.Size = value
	case "status":
		qs.Status = value
	case "ifclientip":
		qs.IfClientIP = value
	case "ifproxy1ip":
		qs.IfProxy1IP = value
	case "ifproxy2ip":
		qs.IfProxy2IP = value
	case "iftargetip":
		qs.IfTargetIP = value
	case "ifhostip":
		qs.IfHostIP = value
	case "ifhost":
		qs.IfHost = value
	case "ifaz":
		qs.IfAZ = value
	}
}

func newValidator() map[string]*regexp.Regexp {
	const (
		regexpPercent  = "^(100|[0-9]{1,2})$"
		regexpNumRange = "^([0-9]+)(?:-([0-9]+))?$"
		//regexpNumComma = "^([0-9]+)(?:,([0-9]+))*$" // 2個以上はFindStringSubmatchで取得不可のためmatchしたらstrings.Split
		regexpStatus   = "^(200|400|403|404|500|502|503|504)$"
		regexpHostname = "^([a-zA-Z0-9-.]+)$"
		regexpAZone    = "^([a-z]{2}-[a-z]+-[1-9][a-d])$"
		regexpIPv4     = "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
		regexpIPv6     = "^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$"
	)
	validator := map[string]*regexp.Regexp{}
	validator["cpu"] = regexp.MustCompile(regexpPercent)
	validator["memory"] = regexp.MustCompile(regexpPercent)
	validator["sleep"] = regexp.MustCompile(regexpNumRange)
	validator["size"] = regexp.MustCompile(regexpNumRange)
	validator["status"] = regexp.MustCompile(regexpStatus)
	validator["ifhost"] = regexp.MustCompile(regexpHostname)
	validator["ifaz"] = regexp.MustCompile(regexpAZone)
	validator["ifhostip"] = regexp.MustCompile(fmt.Sprintf("(%s|%s)", regexpIPv4, regexpIPv6))
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
	fmt.Printf("%v\n", store)
	store.host.Name, _ = os.Hostname()
	store.host.IP = getIPAddress()
	http.HandleFunc("/", handler)
	srv := &http.Server{Addr: ":9000"}
	log.Fatalln(srv.ListenAndServe())
}

func handler(w http.ResponseWriter, r *http.Request) {
	//w.WriteHeader(http.StatusNotFound)
	reqInfo := RequestInfo{
		Path:   r.URL.EscapedPath(),
		Query:  r.URL.Query().Encode(),
		Header: combineValues(r.Header),
	}
	reqInfo.setIPAddresse(r)
	respInfo := ResponseInfo{
		Host: store.host,
		Resource: ResourceInfo{
			CPU:    ResourceUsage{Target: store.resource.CPU.getTarget(), Current: store.resource.CPU.getCurrent()},
			Memory: ResourceUsage{Target: store.resource.Memory.getTarget(), Current: store.resource.Memory.getCurrent()},
		},
		Request:   reqInfo,
		Direction: Direction{},
	}

	inputQs := reqInfo.validateQueryString(r.URL.Query())
	actionQs := inputQs.evaluate(&reqInfo)
	respInfo.Direction.Input = inputQs
	respInfo.Direction.Action = actionQs
	s, _ := json.MarshalIndent(respInfo, "", "  ")
	fmt.Fprintf(w, "\n%s\n", string(s))
}

func (reqInfo *RequestInfo) validateQueryString(mapQs map[string][]string) *QueryString {
	qs := &QueryString{needsAction: true}
	for key, value := range combineValues(mapQs) {
		if re, ok := store.validator[key]; ok {
			if len(re.FindStringSubmatch(value)) > 0 {
				qs.setValue(key, value)
				if strings.HasPrefix(key, "if") {
					if reqInfo.getActualValue(key) != value {
						qs.needsAction = false
					}
				} else {
					qs.existsAction = true
					fmt.Printf("  valid %s = %s\n", key, value)
				}
			} else {
				fmt.Printf("invalid %s = %s\n", key, value)
			}
		}
	}
	return qs
}

func combineValues(input map[string][]string) map[string]string {
	output := map[string]string{}
	for key := range input {
		output[key] = strings.Join(input[key], ", ")
	}
	return output
}

func (reqInfo *RequestInfo) setIPAddresse(r *http.Request) {
	reqInfo.TargetIP = extractIPAddress(r.Host)
	xff := splitXFF(r.Header.Get("X-Forwarded-For"))
	if len(xff) == 0 {
		reqInfo.ClientIP = extractIPAddress(r.RemoteAddr)
	} else {
		reqInfo.ClientIP = xff[0]
	}
	if len(xff) >= 2 {
		reqInfo.Proxy1IP = xff[1]
	}
	if len(xff) >= 3 {
		reqInfo.Proxy2IP = xff[2]
	}
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

func splitXFF(xffStr string) []string {
	if xffStr == "" {
		return []string{}
	}
	xff := strings.Split(xffStr, ",")
	for i := range xff {
		xff[i] = strings.TrimSpace(xff[i])
	}
	return xff
}

func (qs *QueryString) evaluate(reqInfo *RequestInfo) *QueryString {
	actionQs := &QueryString{}
	if !qs.existsAction {
		return actionQs
	}
	if !qs.needsAction {
		return actionQs
	}
	// action evaluation

	return qs
}

func (reqInfo *RequestInfo) getActualValue(key string) (ret string) {
	switch key {
	case "ifclientip":
		ret = reqInfo.ClientIP
	case "ifproxy1ip":
		ret = reqInfo.Proxy1IP
	case "ifproxy2ip":
		ret = reqInfo.Proxy2IP
	case "iftargetip":
		ret = reqInfo.TargetIP
	case "ifhostip":
		ret = store.host.IP
	case "ifhost":
		ret = store.host.Name
	case "ifaz":
		ret = store.host.AZ
	}
	return
}
