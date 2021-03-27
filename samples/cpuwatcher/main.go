package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	fmt.Println("CPU usage % at 1 second intervals:")
	store := New()
	closeCh := make(chan bool)
	go procStatParser(store, closeCh)
	for i := 1; ; i++ {
		fmt.Printf("%d : %6.3f\n", i, store.Get("cpu"))
		time.Sleep(time.Millisecond * 400)
	}
}

func procStatParser(ds *DataStore, closeCh chan bool) {
	t := time.NewTicker(time.Duration(500) * time.Millisecond)
	defer t.Stop()
	var prevIdleTime, prevTotalTime uint64
	for i := 0; ; i++ {
		select {
		case <-t.C: //タイマーイベント
			file, err := os.Open("/proc/stat")
			if err != nil {
				log.Fatal(err)
			}
			scanner := bufio.NewScanner(file)
			scanner.Scan()
			firstLine := scanner.Text()[5:] // get rid of cpu plus 2 spaces
			file.Close()
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
			split := strings.Fields(firstLine)
			idleTime, _ := strconv.ParseUint(split[3], 10, 64)
			totalTime := uint64(0)
			for _, s := range split {
				u, _ := strconv.ParseUint(s, 10, 64)
				totalTime += u
			}
			if i > 0 {
				deltaIdleTime := idleTime - prevIdleTime
				deltaTotalTime := totalTime - prevTotalTime
				cpuUsage := (1.0 - float64(deltaIdleTime)/float64(deltaTotalTime)) * 100.0
				ds.Set("cpu", cpuUsage)
			}
			prevIdleTime = idleTime
			prevTotalTime = totalTime
			break
		case <-closeCh:
			return
		}
	}
}

/*
	done := make(chan int, runtime.NumCPU())

  for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}
}

func resourceEater(closeCh chan bool, ticker int) {
	t := time.NewTicker(time.Duration(ticker) * time.Millisecond)
	defer t.Stop()
  done := make(chan int)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}
	//time.Sleep(time.Second * 10)
	//close(done)
}
*/

type DataStore struct {
	sync.Mutex
	resource ResourceData
}
type ResourceData struct {
	CPUUsage    float64
	MempryUsage float64
	CPULoad     float64
}

func New() *DataStore {
	return &DataStore{
		resource: ResourceData{
			CPUUsage: 100.0,
		},
	}
}
func (ds *DataStore) Set(key string, value float64) {
	ds.Lock()
	defer ds.Unlock()
	ds.resource.CPUUsage = value
}
func (ds *DataStore) Get(key string) float64 {
	ds.Lock()
	defer ds.Unlock()
	return ds.resource.CPUUsage
}
