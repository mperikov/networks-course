package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
)

type routerConfig struct {
	IP        string   `json:"ip"`
	Neighbors []string `json:"neighbors"`
}

type networkConfig struct {
	Routers []routerConfig `json:"routers"`
}

func main() {
	steps := flag.Int("steps", 4, "count of simulation steps")
	flag.Parse()

	routers, err := loadRouters("network.json")
	if err != nil {
		log.Fatal(err)
	}

	runSimulation(routers, *steps)
}

func loadRouters(path string) (map[string]*Router, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg networkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.Routers) == 0 {
		return nil, fmt.Errorf("config contains no routers")
	}

	routers := make(map[string]*Router, len(cfg.Routers))
	for _, rc := range cfg.Routers {
		if rc.IP == "" {
			return nil, fmt.Errorf("router with empty IP in config")
		}
		if _, exists := routers[rc.IP]; exists {
			return nil, fmt.Errorf("duplicate router IP: %s", rc.IP)
		}
		routers[rc.IP] = &Router{
			IP:        rc.IP,
			Neighbors: append([]string(nil), rc.Neighbors...),
		}
	}

	for _, rc := range cfg.Routers {
		for _, neighbor := range rc.Neighbors {
			if _, exists := routers[neighbor]; !exists {
				return nil, fmt.Errorf("unknown neighbor %s for router %s", neighbor, rc.IP)
			}
		}
	}

	return routers, nil
}

func runSimulation(routers map[string]*Router, steps int) {
	fmt.Println("=== Minimal routing tables ===")
	for _, router := range routers {
		router.InitMinimalTable()
	}
	printRouters(routers, 1)

	for step := range steps - 1 {
		makeStep(routers)
		if step+2 < steps {
			fmt.Printf("=== Routing tables after %d step ===\n", step+2)
			printRouters(routers, step+2)
		}
	}

	fmt.Println("=== Final routing tables ===")
	printFinalRouters(routers)
}

func makeStep(routers map[string]*Router) {
	for _, router := range routers {
		router.ReceivedUpdates = nil
	}
	for _, router := range routers {
		router.SendTableToNeighbors(routers)
	}
	for _, router := range routers {
		router.ProcessReceivedUpdates()
	}
}

func routerIPs(routers map[string]*Router) []string {
	ips := make([]string, 0, len(routers))
	for ip := range routers {
		ips = append(ips, ip)
	}
	sort.Strings(ips)
	return ips
}

func printRouters(routers map[string]*Router, step int) {
	for _, ip := range routerIPs(routers) {
		routers[ip].PrintTable(fmt.Sprintf("Simulation step %d of router %s", step, ip))
	}
}

func printFinalRouters(routers map[string]*Router) {
	for _, ip := range routerIPs(routers) {
		routers[ip].PrintTable(fmt.Sprintf("Final state of router %s table:", ip))
	}
}
