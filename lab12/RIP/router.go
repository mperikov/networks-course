package main

import (
	"fmt"
)

type RouteEntry struct {
	Source      string
	Destination string
	NextHop     string
	Metric      int
}

type RoutingTable map[string]RouteEntry

type Router struct {
	IP              string
	Neighbors       []string
	Table           RoutingTable
	ReceivedUpdates []RouteEntry
}

func (r *Router) InitMinimalTable() {
	r.Table = make(RoutingTable)
	r.ReceivedUpdates = nil
	for _, neighbor := range r.Neighbors {
		r.Table[neighbor] = RouteEntry{
			Source:      r.IP,
			Destination: neighbor,
			NextHop:     neighbor,
			Metric:      1,
		}
	}
}

func (r *Router) SendTableToNeighbors(routers map[string]*Router) {
	for _, neighborIP := range r.Neighbors {
		neighbor := routers[neighborIP]
		r.sendTableToNeighbor(neighbor)
	}
}

func (r *Router) sendTableToNeighbor(neighbor *Router) {
	for _, route := range r.Table {
		neighbor.ReceivedUpdates = append(neighbor.ReceivedUpdates, route)
	}
}

func (r *Router) ProcessReceivedUpdates() {
	for _, entry := range r.ReceivedUpdates {
		if entry.Destination == r.IP {
			continue
		}
		if entry.NextHop == r.IP {
			// Split horizon
			continue
		}
		entry.NextHop = entry.Source
		entry.Source = r.IP
		entry.Metric++
		cur, ok := r.Table[entry.Destination]
		if !ok || cur.Metric > entry.Metric {
			r.Table[entry.Destination] = entry
		}
	}
	r.ReceivedUpdates = nil
}

func (r *Router) PrintTable(header string) {
	fmt.Println(header)
	fmt.Println("[Source IP]\t[Destination IP]\t[Next Hop]\t[Metric]")

	for _, route := range r.Table {
		fmt.Printf("%s\t%s\t\t%s\t%d\n",
			route.Source, route.Destination, route.NextHop, route.Metric)
	}
	fmt.Println()
}
