package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Stats represents memory statistics for linux
type Stats struct {
	Total, Used, Buffers, Cached, Free, Available, Active, Inactive,
	SwapTotal, SwapUsed, SwapCached, SwapFree uint64
}

func main() {
	fmt.Println("Memory usage % at 1 second intervals:")

	for i := 0; ; i++ {
		if i < 20 {
			go consumeMemory()
		}
		file, err := os.Open("/proc/meminfo")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		var memory Stats
		memStats := map[string]*uint64{
			"MemTotal":     &memory.Total,
			"MemFree":      &memory.Free,
			"MemAvailable": &memory.Available,
			"Buffers":      &memory.Buffers,
			"Cached":       &memory.Cached,
			"Active":       &memory.Active,
			"Inactive":     &memory.Inactive,
			"SwapCached":   &memory.SwapCached,
			"SwapTotal":    &memory.SwapTotal,
			"SwapFree":     &memory.SwapFree,
		}
		for scanner.Scan() {
			line := scanner.Text()
			i := strings.IndexRune(line, ':')
			if i < 0 {
				continue
			}
			fld := line[:i]
			if ptr := memStats[fld]; ptr != nil {
				val := strings.TrimSpace(strings.TrimRight(line[i+1:], "kB"))
				if v, err := strconv.ParseUint(val, 10, 64); err == nil {
					*ptr = v * 1024
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		memory.SwapUsed = memory.SwapTotal - memory.SwapFree
		memory.Used = memory.Total - memory.Available

		fmt.Printf("%d : %4d (%d/%d) %d\n", i, memory.Used*100.0/memory.Total, memory.Used, memory.Total, runtime.NumGoroutine())
		time.Sleep(time.Second)
	}
}

func consumeMemory() {
	buf := NewBuffer()
	for i := 0; i < 10000000; i++ {
		buf.Append("Hello").Append(", ").Append("world.")
		buf.Append(LF).Append("I have a ").Append("pen.")
	}
	randInt := rand.Intn(30)
	time.Sleep(time.Duration(randInt) * time.Second)
	runtime.GC()
}

const LF = "\n"

type StringBuffer struct {
	buf  bytes.Buffer
	size int
	err  error
}

func (s *StringBuffer) Append(text string) *StringBuffer {
	if s.err != nil {
		return s // Nothing to do.
	}
	size, err := s.buf.WriteString(text)
	s.size += size
	s.err = err
	return s
}

func (s *StringBuffer) String() string {
	return s.buf.String()
}

func NewBuffer() StringBuffer {
	return StringBuffer{}
}
