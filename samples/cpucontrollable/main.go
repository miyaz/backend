package main

import (
	"runtime"
	"time"
)

func main() {
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

	time.Sleep(time.Second * 10)
	close(done)
}

func moveLoop(moveCh chan int, closeCh chan bool, mover, ticker int) {
  t := time.NewTicker(time.Duration(ticker) * time.Millisecond)
  defer t.Stop()
  for {
    select {
    case <-t.C: //タイマーイベント
      moveCh <- mover
      break
    case <-closeCh:
      return
    }
  }
}

