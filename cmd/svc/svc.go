package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jingyuanliang/conntest/pkg/version"
)

var (
	network string
	address string
)

func init() {
	flag.StringVar(&network, "network", "tcp", "network")
	flag.StringVar(&address, "address", "", "address")

	flag.Parse()
}

var cnt atomic.Int64

func tcp() {
	l, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("[err] %v\n", err)
	}
	defer l.Close()

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

			_, err := io.Copy(c, c)
			if err != nil {
				log.Printf("[c:err] %v\n", err)
			}
		}(conn)
	}
}

func udp() {
	conn, err := net.ListenPacket(network, address)
	if err != nil {
		log.Fatalf("[err] %v\n", err)
	}
	defer conn.Close()

	buf := make([]byte, 65536)
	for {
		rn, addr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("[r:err] %v\n", err)
			continue
		}

		wn, err := conn.WriteTo(buf[:rn], addr)
		if err != nil {
			log.Printf("[w:err] %v\n", err)
			continue
		}
		if rn != wn {
			log.Printf("[rw] %d != %d\n", rn, wn)
			continue
		}

		cnt.Add(1)
	}
}

func main() {
	log.SetPrefix(fmt.Sprintf("[pid:%d] ", os.Getpid()))
	log.Printf("version: %s\n", version.Version)

	go func() {
		for range time.Tick(time.Second * 1) {
			log.Printf("[conn] %d\n", cnt.Load())
		}
	}()

	if strings.HasPrefix(network, "tcp") {
		tcp()
	} else if strings.HasPrefix(network, "udp") {
		udp()
	} else {
		log.Fatalf("[err] unknown network %s\n", network)
	}
}
