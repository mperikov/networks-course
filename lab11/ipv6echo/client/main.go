package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	addr := flag.String("addr", "[::1]:9000", "server address (IPv6)")
	flag.Parse()

	conn, err := net.Dial("tcp6", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	in := bufio.NewScanner(os.Stdin)
	out := bufio.NewScanner(conn)

	for in.Scan() {
		line := in.Text()
		if _, err := conn.Write([]byte(line + "\n")); err != nil {
			log.Fatal(err)
		}
		if !out.Scan() {
			log.Fatal(out.Err())
		}
		fmt.Println(strings.TrimSpace(out.Text()))
	}
}
