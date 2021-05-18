package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

func main() {

	var params []string
	for i := 0; i < 100; i++ {
		params = append(params, strconv.Itoa(9000+i))
	}
	workers := crawle(params)
	fmt.Println(workers)
}

func crawle(inList []string) (outList []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	limiter := make(chan struct{}, 10)
	for i := 0; i < len(inList); i++ {
		addr := inList[i]
		wg.Add(1)
		go func(i int) {
			limiter <- struct{}{}
			defer wg.Done()
			//<-limiter
			resp := checkWorker(i+1, addr)
			<-limiter
			mu.Lock()
			defer mu.Unlock()
			outList = append(outList, resp)
		}(i)
	}
	wg.Wait()
	return
}

func checkWorker(i int, param string) string {
	randInt := rand.Intn(1000)
	time.Sleep(time.Duration(randInt) * time.Millisecond)
	fmt.Printf("%d: %d\n", i, randInt)
	return strconv.Itoa(randInt)
}
