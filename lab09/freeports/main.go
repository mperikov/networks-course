package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	ip := flag.String("ip", "", "target IP address")
	from := flag.Int("from", 0, "start port (inclusive)")
	to := flag.Int("to", 0, "end port (inclusive)")
	timeout := flag.Duration("timeout", 200*time.Millisecond, "connection timeout (e.g. 200ms)")
	flag.Parse()

	if err := validateInput(*ip, *from, *to); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("IP: %s\n", *ip)
	fmt.Printf("Range: %d-%d\n", *from, *to)
	fmt.Println("Available ports:")

	hasAvailable := false
	for port := *from; port <= *to; port++ {
		if isPortAvailable(*ip, port, *timeout) {
			hasAvailable = true
			fmt.Println(port)
		}
	}

	if !hasAvailable {
		fmt.Println("none")
	}
}

func validateInput(ip string, from, to int) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}
	if from < 1 || to < 1 || from > 65535 || to > 65535 {
		return errors.New("ports must be in range 1..65535")
	}
	if from > to {
		return errors.New("start port must be less than or equal to end port")
	}
	return nil
}

func isPortAvailable(ip string, port int, timeout time.Duration) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err == nil {
		_ = conn.Close()
		return true
	}

	return false
}
