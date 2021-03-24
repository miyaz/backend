package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
)

var (
	revision string
	buildAt  string
	logger   *logrus.Logger
)

func init() {
	logger = &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.JSONFormatter{},
		Level:     logrus.DebugLevel,
		Hooks:     make(logrus.LevelHooks),
	}
}

func main() {
	logger.Infof("revision: %s, buildAt: %s", revision, buildAt)

	http.HandleFunc("/", handler)
	srv := &http.Server{Addr: ":9000"}
	logger.Fatalln(srv.ListenAndServe())
}

func handler(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	ip := readUserIP(r)

	logger := logger.WithFields(logrus.Fields{"host": host, "ip": ip})
	logger.Debugf("%s %s?%s %s", r.Method, r.URL.Path, r.URL.RawQuery, r.Proto)
	fmt.Fprint(w, fmt.Sprintf("host: %s\nip: %s\n\n", host, ip))

	headers := headerToArray(r.Header)
	sort.Strings(headers)
	for _, header := range headers {
		fmt.Fprint(w, header+"\n")
	}
}

func readUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

func headerToArray(header http.Header) (res []string) {
	for name, values := range header {
		for _, value := range values {
			res = append(res, fmt.Sprintf("%s: %s", name, value))
		}
	}
	return
}
