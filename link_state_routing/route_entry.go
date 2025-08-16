package main

type RouteEntry struct {
	Destination string
	NextHop     string
	Cost        int
	Valid       bool
}

type RoutingTable struct {
	routes map[string]*RouteEntry
}

func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		routes: make(map[string]*RouteEntry),
	}
}

func (rt *RoutingTable) AddRoute(dest, nextHop string, cost int) {
	rt.routes[dest] = &RouteEntry{
		Destination: dest,
		NextHop:     nextHop,
		Cost:        cost,
		Valid:       true,
	}
}

func (rt *RoutingTable) GetRoute(dest string) (*RouteEntry, bool) {
	route, exists := rt.routes[dest]
	return route, exists && route.Valid
}

func (rt *RoutingTable) RemoveRoute(dest string) {
	delete(rt.routes, dest)
}

func (rt *RoutingTable) GetAllRoutes() map[string]*RouteEntry {
	return rt.routes
}

func (rt *RoutingTable) Clear() {
	rt.routes = make(map[string]*RouteEntry)
}
