package main

import (
	"math"
	"sync"
)

// ===== Link-State runtime (LSR helpers) =====

// Global in-memory LSDB: router -> (neighbor -> cost)
var (
	lsdb   = map[string]map[string]int{}
	lsdbMu sync.RWMutex
)

// Ensure the node's own LSA is present in the DB (from its local neighbor table).
func ensureLocalLSA(node *Node) {
	lsdbMu.Lock()
	defer lsdbMu.Unlock()

	if _, ok := lsdb[node.ID]; !ok {
		cp := make(map[string]int, len(node.neighbors))
		for k, v := range node.neighbors {
			cp[k] = v
		}
		lsdb[node.ID] = cp
	}
}

// Insert/update the LSA for "origin" in the DB. Returns true if anything changed.
func updateLSA(origin string, neighbors map[string]int) bool {
	lsdbMu.Lock()
	defer lsdbMu.Unlock()

	old, ok := lsdb[origin]
	changed := !ok

	if ok && !changed {
		if len(old) != len(neighbors) {
			changed = true
		} else {
			for k, v := range neighbors {
				if ov, ok := old[k]; !ok || ov != v {
					changed = true
					break
				}
			}
			if !changed {
				for k := range old {
					if _, ok := neighbors[k]; !ok {
						changed = true
						break
					}
				}
			}
		}
	}

	if changed {
		cp := make(map[string]int, len(neighbors))
		for k, v := range neighbors {
			cp[k] = v
		}
		lsdb[origin] = cp
	}
	return changed
}

// Basic loop detection using headers (last routers visited).
func containsHeader(headers []string, id string) bool {
	for _, h := range headers {
		if h == id {
			return true
		}
	}
	return false
}

// On forward: remove the first header and append "self" at the end (max 3 elements).
func rotateHeaders(headers []string, self string) []string {
	if headers == nil {
		return []string{self}
	}
	out := make([]string, 0, 3)
	if len(headers) > 0 {
		out = append(out, headers[1:]...)
	}
	out = append(out, self)
	if len(out) > 3 {
		out = out[len(out)-3:]
	}
	return out
}

// Build an undirected graph from the LSDB.
func buildGraphFromLSDB() map[string]map[string]int {
	lsdbMu.RLock()
	defer lsdbMu.RUnlock()

	g := make(map[string]map[string]int)
	for u, nbrs := range lsdb {
		if _, ok := g[u]; !ok {
			g[u] = make(map[string]int)
		}
		for v, c := range nbrs {
			g[u][v] = c
			if _, ok := g[v]; !ok {
				g[v] = make(map[string]int)
			}
			// mirror the edge in both directions (if mismatch, keep the smaller cost)
			if cur, ok := g[v][u]; !ok || c < cur {
				g[v][u] = c
			}
		}
	}
	return g
}

// O(V^2) Dijkstra over the graph built from LSDB.
func runDijkstra(graph map[string]map[string]int, source string) (map[string]int, map[string]string) {
	dist := map[string]int{}
	prev := map[string]string{}
	visited := map[string]bool{}

	// initialize distances
	for n := range graph {
		dist[n] = math.MaxInt32
	}
	if _, ok := graph[source]; !ok {
		graph[source] = map[string]int{}
	}
	dist[source] = 0

	for len(visited) < len(graph) {
		// pick the unvisited node with minimum distance
		cur := ""
		min := math.MaxInt32
		for n, d := range dist {
			if !visited[n] && d < min {
				cur, min = n, d
			}
		}
		if cur == "" {
			break
		}
		visited[cur] = true

		// relax neighbors
		for v, c := range graph[cur] {
			if visited[v] || dist[cur] == math.MaxInt32 {
				continue
			}
			alt := dist[cur] + c
			if alt < dist[v] {
				dist[v] = alt
				prev[v] = cur
			}
		}
	}
	return dist, prev
}

// Return the next hop from "src" towards "dst" (or "" if no path).
func computeNextHop(graph map[string]map[string]int, src, dst string) string {
	if src == dst {
		return ""
	}
	dist, prev := runDijkstra(graph, src)
	if d, ok := dist[dst]; !ok || d == math.MaxInt32 {
		return ""
	}
	cur := dst
	for {
		p, ok := prev[cur]
		if !ok {
			return ""
		}
		if p == src {
			return cur
		}
		cur = p
	}
}
