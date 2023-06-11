package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"
)

var cnt atomic.Int64

func main() {
	log.SetPrefix(fmt.Sprintf("[pid:%d] ", os.Getpid()))

	l, err := net.Listen(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatalf("[err] %v\n", err)
	}
	defer l.Close()

	go func() {
		for range time.Tick(time.Second * 1) {
			log.Printf("[conn] %d\n", cnt.Load())
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("[err] %v\n", err)
			time.Sleep(time.Second * 1)
			continue
		}

		go func(c net.Conn) {
			cnt.Add(1)
			defer cnt.Add(-1)
			defer c.Close()

			io.Copy(c, c)
		}(conn)
	}
}
