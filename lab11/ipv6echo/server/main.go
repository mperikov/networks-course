package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net"
	"strings"
	"syscall"
)

func main() {
	addr := flag.String("addr", "[::]:9000", "IPv6 listen address")
	flag.Parse()

	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_V6ONLY, 1)
			})
		},
	}
	ln, err := lc.Listen(context.Background(), "tcp6", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("listening on %s (tcp6)", *addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		reply := strings.ToUpper(sc.Text()) + "\n"
		if _, err := conn.Write([]byte(reply)); err != nil {
			return
		}
	}
}
