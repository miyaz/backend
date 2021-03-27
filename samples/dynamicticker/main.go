package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	quit := make(chan bool)
	ratio := make(chan float64)

	go cpuUsageController(ratio, quit)

	time.Sleep(10 * time.Second)
	for i := 0; i < 20; i++ {
		ratio <- 0.7
		time.Sleep(1 * time.Second)
	}
	time.Sleep(10 * time.Second)
	for i := 0; i < 20; i++ {
		ratio <- 1.5
		time.Sleep(2 * time.Second)
	}

	log.Println("stopping ticker...")
	quit <- true

	time.Sleep(500 * time.Millisecond)
}

func cpuUsageController(ratio chan float64, quit chan bool) {
	interval := float64(1000)
	prevInterval := interval
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			done := make(chan int)
			go placeLoad(done)
			go stopTimer(done)
		case newRatio := <-ratio:
			if prevInterval != newRatio*interval {
				interval *= newRatio
				prevInterval = interval
				if interval > 1.0 {
					ticker.Stop()
					ticker = time.NewTicker(time.Duration(interval) * time.Millisecond)
					log.Println("ticker changing to " + fmt.Sprint(interval) + " ms")
				} else {
					log.Println("ticker no changing, cause be less than 1.0 ms")
				}
			}
		case <-quit:
			ticker.Stop()
			log.Println("..ticker stopped!")
			return
		}
	}
}

func stopTimer(done chan int) {
	time.Sleep(500 * time.Millisecond)
	close(done)
}

func placeLoad(done chan int) {
	//for i := 0; i < runtime.NumCPU(); i++ {
	go func() {
		x := 0
		for {
			select {
			case <-done:
				return
			default:
				x++
			}
		}
	}()
	//}
}
