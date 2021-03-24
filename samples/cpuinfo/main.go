package main

import (
	"fmt"
	"log"
	"time"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

func main() {

	for {
		cpuinfo, err := linuxproc.ReadCPUInfo("/proc/cpuinfo")
		if err != nil {
			log.Fatal("cpuinfo read fail")
		}
		fmt.Printf("%v\n", cpuinfo)
		time.Sleep(time.Second)
	}
}
