package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"
)

func validate(addr string) error {
	host, portstr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid IP address: %s", addr)
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		return fmt.Errorf("invalid port: %v", portstr)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %v", port)
	}
	valid := net.ParseIP(host)
	if valid == nil {
		return fmt.Errorf("invalid IP: %v", host)
	}
	return nil
}

func main() {
	addr := flag.String("addr", "127.0.0.1:80", "combined ip:port address to connect to")
	max_ttl := flag.Int("ttl", 64, "maximum TTL value to use")
	timeout_val := flag.Int("timeout", 2, "number of seconds before timeout")
	flag.Parse()

	if err := validate(*addr); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	timeout := time.Duration(*timeout_val) * time.Second
	fmt.Printf("Traceroute to '%s'\n", *addr)

	for ttl := 0; ttl <= *max_ttl; ttl++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		// testing Control for control over time to live
		dialer := &net.Dialer{
			Control: func(network, address string, c syscall.RawConn) error {
				var err error
				err = c.Control(func(fd uintptr) {
					err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
				})
				return err
			},
			Timeout: timeout,
		}

		start := time.Now()
		conn, err := dialer.DialContext(ctx, "tcp", *addr)
		elapsed := time.Since(start)

		fmt.Printf("%2d ", ttl)

		if err != nil {
			// in theory this is where we could listen for the return - ICMP usually responds with
			// some version of Timeout Exceeded and then print who was in the chain.
			fmt.Println("Timeout out")
		} else {
			fmt.Printf("%s (%s) %s\n", conn.RemoteAddr(), conn.RemoteAddr().String(), elapsed)
			if err := conn.Close(); err != nil {
				fmt.Println("error closing connection", err)
			}
			break // break the ttl loop cause we're here.
		}
	}
}
