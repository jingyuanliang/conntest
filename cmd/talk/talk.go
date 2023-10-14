package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jingyuanliang/conntest/pkg/version"
)

var (
	timeout  time.Duration
	deadline time.Duration
	steady   time.Duration

	network string
	address string
	bind    string
	begin   int
	end     int
)

func init() {
	flag.DurationVar(&timeout, "timeout", time.Second*10, "timeout for connection")
	flag.DurationVar(&deadline, "deadline", time.Second*10, "deadline for read/write")
	flag.DurationVar(&steady, "steady", time.Second*0, "terminate if stay unchanged for")

	flag.StringVar(&network, "network", "tcp", "network of svc")
	flag.StringVar(&address, "address", "", "address of svc")
	flag.StringVar(&bind, "bind", "", "address to bind talk")
	flag.IntVar(&begin, "begin", 0, "start of port range to bind")
	flag.IntVar(&end, "end", 0, "end of port range to bind")

	flag.Parse()
}

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
		if deadline != 0 {
			err := c.SetWriteDeadline(time.Now().Add(deadline))
			if err != nil {
				firstErrCh <- cnt.Load()
				log.Printf("[wd:err] %v\n", err)
				break
			}
		}
		_, err := c.Write(buf)
		if err != nil {
			firstErrCh <- cnt.Load()
			log.Printf("[w:err] %v\n", err)
			break
		}

		if deadline != 0 {
			err := c.SetReadDeadline(time.Now().Add(deadline))
			if err != nil {
				firstErrCh <- cnt.Load()
				log.Printf("[rd:err] %v\n", err)
				break
			}
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
		dialer := net.Dialer{
			Timeout: timeout,
		}

		conn, err := dialer.Dial(network, address)
		if err != nil {
			log.Printf("[d:err] %v\n", err)
			time.Sleep(time.Second * 1)
			continue
		}

		go talk(conn)
	}
}

func explicit() {
	addr, err := netip.ParseAddr(bind)
	if err != nil {
		log.Fatalf("[err] %v", err)
	}

	for {
		for i := begin; i <= end; i++ {
			ap := netip.AddrPortFrom(addr, uint16(i))
			dialer := net.Dialer{
				Timeout:   timeout,
				LocalAddr: net.TCPAddrFromAddrPort(ap),
			}

			conn, err := dialer.Dial(network, address)
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
			} else if steady != 0 && time.Since(topTime) > steady {
				log.Printf("[complete] top-conn %d, first-err %d\n", topCnt, firstErr.Load())
				os.Exit(0)
			}
		}
	}()

	if bind == "" {
		implicit()
	} else {
		explicit()
	}
}
