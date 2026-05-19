package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"time"
)

const (
	probesPerHop = 3
	maxTTL       = 64
	probeTimeout = time.Second
)

func main() {
	host := flag.String("host", "", "destination host or IPv4")
	probes := flag.Int("n", probesPerHop, "probes per hop")
	names := flag.Bool("names", false, "show hosts names")
	flag.Parse()

	dst, err := net.ResolveIPAddr("ip4", *host)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	id := os.Getpid() & 0xffff
	payload := make([]byte, 32)
	fmt.Printf("traceroute to %s (%s), max ttl %d, %d probes per hop\n", *host, dst.IP, maxTTL, *probes)

	for ttl := 1; ttl <= maxTTL; ttl++ {
		setTTL(conn, ttl)
		var hop net.IP
		done := false
		ms := make([]float64, *probes)
		ok := make([]bool, *probes)
		for p := 0; p < *probes; p++ {
			peer, rtt, typ, got := probe(conn, dst.IP, id, ttl<<8|p, payload)
			ok[p] = got
			if !got {
				continue
			}
			ms[p] = rtt
			if hop == nil {
				hop = peer
			}
			if typ == 0 {
				done = true
			}
		}
		fmt.Printf("%2d  ", ttl)
		if hop == nil {
			for range ok {
				fmt.Print("  *")
			}
			fmt.Println()
			continue
		}
		if *names {
			if ns, _ := net.LookupAddr(hop.String()); len(ns) > 0 {
				fmt.Printf("%s (%s)", trimDot(ns[0]), hop)
			} else {
				fmt.Print(hop)
			}
		} else {
			fmt.Print(hop)
		}
		for p := range ok {
			if ok[p] {
				fmt.Printf("  %.3f ms", ms[p])
			} else {
				fmt.Print("  *")
			}
		}
		fmt.Println()
		if done || hop.Equal(dst.IP) {
			break
		}
	}
}

func setTTL(c net.PacketConn, ttl int) {
	sc, _ := c.(interface {
		SyscallConn() (syscall.RawConn, error)
	})
	raw, _ := sc.SyscallConn()
	raw.Control(func(fd uintptr) {
		syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	})
}

func probe(conn net.PacketConn, dst net.IP, id, seq int, payload []byte) (net.IP, float64, byte, bool) {
	conn.WriteTo(echo(id, seq, payload), &net.IPAddr{IP: dst})
	t0 := time.Now()
	conn.SetReadDeadline(time.Now().Add(probeTimeout))
	buf := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return nil, 0, 0, false
		}
		rid, rseq, ok := parseReply(buf[:n])
		if ok && rid == id && rseq == seq {
			return addr.(*net.IPAddr).IP, float64(time.Since(t0).Microseconds()) / 1000, buf[0], true
		}
	}
}

func echo(id, seq int, data []byte) []byte {
	b := make([]byte, 8+len(data))
	b[0] = 8
	binary.BigEndian.PutUint16(b[4:], uint16(id))
	binary.BigEndian.PutUint16(b[6:], uint16(seq))
	copy(b[8:], data)
	binary.BigEndian.PutUint16(b[2:], checksum(b))
	return b
}

func parseReply(p []byte) (id, seq int, ok bool) {
	if len(p) < 8 {
		return
	}
	if p[0] == 0 {
		return int(binary.BigEndian.Uint16(p[4:])), int(binary.BigEndian.Uint16(p[6:])), true
	}
	if p[0] != 11 || len(p) < 28 {
		return
	}
	body := p[8:]
	ihl := int(body[0]&0x0f) * 4
	if len(body) < ihl+8 || body[ihl] != 8 {
		return
	}
	inner := body[ihl:]
	return int(binary.BigEndian.Uint16(inner[4:])), int(binary.BigEndian.Uint16(inner[6:])), true
}

func checksum(b []byte) uint16 {
	var s uint32
	for i := 0; i+1 < len(b); i += 2 {
		s += uint32(binary.BigEndian.Uint16(b[i : i+2]))
	}
	if len(b)%2 == 1 {
		s += uint32(b[len(b)-1]) << 8
	}
	for s > 0xffff {
		s = (s & 0xffff) + (s >> 16)
	}
	return ^uint16(s)
}

func trimDot(s string) string {
	if s != "" && s[len(s)-1] == '.' {
		return s[:len(s)-1]
	}
	return s
}
