package main

import (
	"bufio"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const (
	letterBytes   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	respSize      = 10240
)

func main() {
	http.HandleFunc("/", handler)
	srv := &http.Server{Addr: ":9000"}
	log.Fatalln(srv.ListenAndServe())

}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s?%s %s", r.Method, r.URL.Path, r.URL.RawQuery, r.Proto)
	//fmt.Fprint(w, host+"\n")
	fw := bufio.NewWriter(w)
	src := rand.New(rand.NewSource(time.Now().UnixNano()))

	loopCount := respSize / 100
	remainder := respSize % 100
	for i := 0; i < loopCount; i++ {
		fw.Write(randBytes(src, 99))
		fw.Write([]byte("\n"))
	}
	if remainder != 0 {
		fw.Write(randBytes(src, remainder))
	}

	err := fw.Flush()
	if err != nil {
		log.Fatalln(err)
	}
}

func randBytes(src *rand.Rand, n int) []byte {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}
