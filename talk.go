package main

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var cnt atomic.Int64

func talk(c net.Conn) {
	cnt.Add(1)
	defer cnt.Add(-1)
	defer c.Close()

	buf := []byte("x")
	for range time.Tick(time.Second * 1) {
		_, err := c.Write(buf)
		if err != nil {
			log.Printf("[w:err] %v\n", err)
			break
		}

		_, err = c.Read(buf)
		if err != nil {
			log.Printf("[r:err] %v\n", err)
			break
		}
	}
}

func implicit() {
	for {
		conn, err := net.Dial(os.Args[1], os.Args[2])
		if err != nil {
			log.Printf("[d:err] %v\n", err)
			time.Sleep(time.Second * 1)
			continue
		}

		go talk(conn)
	}
}

func explicit() {
	addr, err := netip.ParseAddr(os.Args[3])
	if err != nil {
		log.Fatalf("[err] %v", err)
	}

	begin, err := strconv.Atoi(os.Args[4])
	if err != nil {
		log.Fatalf("[err] %v", err)
	}

	end, err := strconv.Atoi(os.Args[5])
	if err != nil {
		log.Fatalf("[err] %v", err)
	}

	for {
		for i := begin; i <= end; i++ {
			ap := netip.AddrPortFrom(addr, uint16(i))
			dialer := net.Dialer{
				LocalAddr: net.TCPAddrFromAddrPort(ap),
			}

			conn, err := dialer.Dial(os.Args[1], os.Args[2])
			if err != nil {
				if !strings.Contains(err.Error(), "bind: address already in use") {
					log.Printf("[d:err] [%s] %v\n", ap, err)
					time.Sleep(time.Second * 1)
				}
				continue
			}

			go talk(conn)
		}

		log.Printf("[loop]\n")
		time.Sleep(time.Second * 1)
	}
}

func main() {
	log.SetPrefix(fmt.Sprintf("[pid:%d] ", os.Getpid()))

	go func() {
		for range time.Tick(time.Second * 1) {
			log.Printf("[conn] %d\n", cnt.Load())
		}
	}()

	switch len(os.Args) {
	case 3:
		implicit()
	case 6:
		explicit()
	default:
		log.Fatalf("[args] %v\n", os.Args)
	}
}
