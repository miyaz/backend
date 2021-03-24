package main

import (
	"fmt"
	"log"
	"time"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

func main() {
	for {
		vmstat, err := linuxproc.ReadVMStat("/proc/vmstat")
		if err != nil {
			log.Fatal("vmstat read fail")
		}
		fmt.Printf("%v\n", vmstat)
		time.Sleep(time.Second)
	}
}
