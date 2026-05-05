package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	ip, mask, err := getIPv4AndMask()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("IP: %s\n", ip)
	fmt.Printf("Mask: %s\n", mask)
}

func getIPv4AndMask() (string, string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}

			ipv4 := ipNet.IP.To4()
			if ipv4 == nil {
				continue
			}

			mask := net.IP(ipNet.Mask).String()
			return ipv4.String(), mask, nil
		}
	}

	return "", "", fmt.Errorf("active IPv4 address not found")
}
