package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	icmpCode       = 0
	packetInterval = time.Second
	replyTimeout   = time.Second
	payloadSize    = 16
)

var protocolICMPv4 = ipv4.ICMPTypeEcho.Protocol()

type pingStats struct {
	transmitted int
	received    int
	sumRTT      time.Duration
	minRTT      time.Duration
	maxRTT      time.Duration
}

func main() {
	host := flag.String("host", "", "target host or IPv4 address")
	count := flag.Int("count", 10, "number of echo requests to send")
	ttl := flag.Int("ttl", -1, "IPv4 TTL for outgoing packets (1-255); omit for OS default")
	flag.Parse()

	if err := validateInput(*host, *count, *ttl); err != nil {
		log.Fatal(err.Error())
	}

	dstIP, err := resolveIPv4(context.Background(), *host)
	if err != nil {
		log.Fatal(err.Error())
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer conn.Close()

	if *ttl != -1 {
		p4 := conn.IPv4PacketConn()
		if p4 == nil {
			log.Fatal("IPv4PacketConn is not available on this connection")
		}
		if err := p4.SetTTL(*ttl); err != nil {
			log.Fatalf("set TTL: %v", err)
		}
	}

	pid := os.Getpid() & 0xffff
	sequence := 1

	if *ttl != -1 {
		fmt.Printf("PING %s (%s), ttl=%d\n", *host, dstIP.String(), *ttl)
	} else {
		fmt.Printf("PING %s (%s)\n", *host, dstIP.String())
	}
	var stats pingStats
	for i := 0; i < *count; i++ {
		rtt, peer, icmpLen, checksumValid, err := sendAndReceive(conn, dstIP, pid, sequence, &stats)
		if err != nil {
			fmt.Printf("icmp_seq=%d error=%v — %s\n", sequence, err, stats.rttSummary())
		} else {
			if stats.received == 0 {
				stats.minRTT = rtt
				stats.maxRTT = rtt
			} else {
				if rtt < stats.minRTT {
					stats.minRTT = rtt
				}
				if rtt > stats.maxRTT {
					stats.maxRTT = rtt
				}
			}
			stats.received++
			stats.sumRTT += rtt

			peerStr := peer.String()
			if checksumValid {
				fmt.Printf("%d bytes from %s: icmp_seq=%d time=%s — %s\n",
					icmpLen, peerStr, sequence, rtt.Round(time.Microsecond), stats.rttSummary())
			} else {
				fmt.Printf("%d bytes from %s: icmp_seq=%d time=%s (checksum: invalid) — %s\n",
					icmpLen, peerStr, sequence, rtt.Round(time.Microsecond), stats.rttSummary())
			}
		}
		sequence++

		if i < *count-1 {
			time.Sleep(packetInterval)
		}
	}

	fmt.Printf("\n--- %s (%s) ping statistics ---\n", *host, dstIP.String())
	fmt.Printf("%d packets transmitted, %d received, %.1f%% packet loss\n",
		stats.transmitted, stats.received, stats.lossPercent())
}

func validateInput(host string, count, ttl int) error {
	if host == "" {
		return fmt.Errorf("host is required, use -host")
	}
	if count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}
	if ttl != -1 && (ttl < 1 || ttl > 255) {
		return fmt.Errorf("ttl must be between 1 and 255, or -1 (default) for OS default")
	}
	return nil
}

func resolveIPv4(ctx context.Context, host string) (net.IP, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPv4 address found for host %q", host)
	}
	ip4 := ips[0].To4()
	if ip4 == nil {
		return nil, fmt.Errorf("no IPv4 address found for host %q", host)
	}
	return ip4, nil
}

func sendAndReceive(conn *icmp.PacketConn, dstIP net.IP, pid, sequence int, stats *pingStats) (rtt time.Duration, peer net.Addr, icmpLen int, checksumValid bool, err error) {
	request, err := buildEchoRequest(pid, sequence)
	if err != nil {
		return 0, nil, 0, false, err
	}

	start := time.Now()
	if _, err := conn.WriteTo(request, &net.IPAddr{IP: dstIP}); err != nil {
		return 0, nil, 0, false, err
	}
	stats.transmitted++

	if err := conn.SetReadDeadline(time.Now().Add(replyTimeout)); err != nil {
		return 0, nil, 0, false, err
	}

	replyBuf := make([]byte, 1500)
	for {
		n, p, err := conn.ReadFrom(replyBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return 0, nil, 0, false, fmt.Errorf("request timeout")
			}
			return 0, nil, 0, false, err
		}

		reply := replyBuf[:n]
		replyICMP := extractICMPMessage(reply)
		parsed, err := icmp.ParseMessage(protocolICMPv4, replyICMP)
		if err != nil {
			continue
		}

		rtt = time.Since(start)

		switch parsed.Type {
		case ipv4.ICMPTypeEchoReply:
			echoBody, ok := parsed.Body.(*icmp.Echo)
			if !ok || echoBody.ID != pid || echoBody.Seq != sequence {
				continue
			}
			checksumValid = validateICMPChecksum(replyICMP)
			return rtt, p, len(replyICMP), checksumValid, nil

		default:
			encap, errLabel, ok := icmpErrorEncapsulation(parsed.Type, parsed.Body)
			if !ok || !validateICMPChecksum(replyICMP) || !encapsulationMatchesEcho(encap, pid, sequence) {
				continue
			}
			return 0, p, len(replyICMP), true, fmt.Errorf("from %s: %s, code %d", p.String(), errLabel, parsed.Code)
		}
	}
}

func icmpErrorEncapsulation(typ icmp.Type, body icmp.MessageBody) (data []byte, label string, ok bool) {
	switch typ {
	case ipv4.ICMPTypeDestinationUnreachable:
		du, ok := body.(*icmp.DstUnreach)
		if !ok {
			return nil, "", false
		}
		return du.Data, "destination unreachable", true
	case ipv4.ICMPTypeTimeExceeded:
		te, ok := body.(*icmp.TimeExceeded)
		if !ok {
			return nil, "", false
		}
		return te.Data, "time exceeded", true
	case ipv4.ICMPTypeParameterProblem:
		pp, ok := body.(*icmp.ParamProb)
		if !ok {
			return nil, "", false
		}
		return pp.Data, "parameter problem", true
	default:
		return nil, "", false
	}
}

func (s *pingStats) lossPercent() float64 {
	if s.transmitted == 0 {
		return 0
	}
	return 100.0 * float64(s.transmitted-s.received) / float64(s.transmitted)
}

func (s *pingStats) rttSummary() string {
	if s.received == 0 {
		return "rtt min/avg/max = —/—/— ms"
	}
	minMs := rttToMs(s.minRTT)
	avgMs := rttToMs(s.sumRTT / time.Duration(s.received))
	maxMs := rttToMs(s.maxRTT)
	return fmt.Sprintf("rtt min/avg/max = %.3f/%.3f/%.3f ms", minMs, avgMs, maxMs)
}

func rttToMs(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

func buildEchoRequest(id, seq int) ([]byte, error) {
	return newICMPEchoRequest(id, seq).Marshal(nil)
}

func newICMPEchoRequest(id, seq int) *icmp.Message {
	payload := make([]byte, payloadSize)
	binary.BigEndian.PutUint64(payload[:8], uint64(time.Now().UnixNano()))
	for i := 8; i < len(payload); i++ {
		payload[i] = byte(i)
	}
	return &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: icmpCode,
		Body: &icmp.Echo{
			ID:   id,
			Seq:  seq,
			Data: payload,
		},
	}
}

func extractICMPMessage(packet []byte) []byte {
	h, err := ipv4.ParseHeader(packet)
	if err != nil {
		return packet
	}
	if h.Version != ipv4.Version || h.Protocol != protocolICMPv4 || len(packet) < h.Len {
		return packet
	}
	return packet[h.Len:]
}


func validateICMPChecksum(packet []byte) bool {
	return ICMPChecksum(packet) == 0
}

func ICMPChecksum(data []byte) uint16 {
	var sum uint32

	for i := 0; i+1 < len(data); i += 2 {
		word := uint32(binary.BigEndian.Uint16(data[i : i+2]))
		sum += word
	}

	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	for (sum >> 16) != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

func encapsulationMatchesEcho(data []byte, pid, seq int) bool {
	h, err := ipv4.ParseHeader(data)
	if err != nil || h.Version != ipv4.Version || h.Protocol != protocolICMPv4 || len(data) < h.Len {
		return false
	}
	m, err := icmp.ParseMessage(protocolICMPv4, data[h.Len:])
	if err != nil || m.Type != ipv4.ICMPTypeEcho {
		return false
	}
	echo, ok := m.Body.(*icmp.Echo)
	if !ok {
		return false
	}
	return echo.ID == pid && echo.Seq == seq
}
