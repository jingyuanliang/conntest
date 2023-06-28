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

	"github.com/jingyuanliang/conntest/pkg/version"
)

const (
	deadline = time.Second * 10
	steady   = time.Second * 10
)

var (
	cnt atomic.Int64

	firstErr   atomic.Int64
	firstErrCh chan int64
)

func talk(c net.Conn) {
	cnt.Add(1)
	defer cnt.Add(-1)
	defer c.Close()

	buf := []byte("x")
	for range time.Tick(time.Second * 1) {
		err := c.SetWriteDeadline(time.Now().Add(deadline))
		if err != nil {
			firstErrCh <- cnt.Load()
			log.Printf("[wd:err] %v\n", err)
			break
		}
		_, err = c.Write(buf)
		if err != nil {
			firstErrCh <- cnt.Load()
			log.Printf("[w:err] %v\n", err)
			break
		}

		err = c.SetReadDeadline(time.Now().Add(deadline))
		if err != nil {
			firstErrCh <- cnt.Load()
			log.Printf("[rd:err] %v\n", err)
			break
		}
		_, err = c.Read(buf)
		if err != nil {
			firstErrCh <- cnt.Load()
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
	log.Printf("version: %s\n", version.Version)

	firstErr.Store(-1)
	firstErrCh = make(chan int64)
	go func() {
		firstErr.Store(<-firstErrCh)
		for {
			<-firstErrCh
		}
	}()

	go func() {
		var topCnt int64
		topTime := time.Now()
		for range time.Tick(time.Second * 1) {
			c := cnt.Load()
			log.Printf("[conn] %d\n", c)
			if c > topCnt {
				topCnt = c
				topTime = time.Now()
			} else if time.Since(topTime) > steady {
				log.Printf("[complete] top-conn %d, first-err %d\n", topCnt, firstErr.Load())
				os.Exit(0)
			}
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
