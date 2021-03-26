package main

import (
	"fmt"
	"log"
	"runtime"
	"time"
)

func main() {
	startInterval := float64(1000)
	quit := make(chan bool)

	go func() {
		ticker := time.NewTicker(time.Duration(startInterval) * time.Millisecond)
		ratio := 0.95
		//counter := 1.0

		for {
			select {
			case <-ticker.C:
				startInterval *= ratio
				if startInterval > 1.0 {
					log.Println("ticker accelerating to " + fmt.Sprint(startInterval) + " ms")
					ticker.Stop()
					ticker = time.NewTicker(time.Duration(startInterval) * time.Millisecond)
					//counter++
					if startInterval > 1000 {
						ratio = 0.95
					}
				} else {
					ratio = 1.05
				}
				done := make(chan int)
				go placeLoad(done)
				go stopTimer(done)
			case <-quit:
				ticker.Stop()
				log.Println("..ticker stopped!")
				return
			}
		}
	}()

	time.Sleep(60 * time.Second)

	log.Println("stopping ticker...")
	quit <- true

	time.Sleep(500 * time.Millisecond)
}

func stopTimer(done chan int) {
	time.Sleep(10 * time.Millisecond)
	close(done)
}

func placeLoad(done chan int) {
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
