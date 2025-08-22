package main

import "math"

type DijkstraResult struct {
	Distances map[string]int
	Previous  map[string]string
}

func RunDijkstra(topology *TopologyDB, source string) *DijkstraResult {
	distances := make(map[string]int)
	previous := make(map[string]string)
	visited := make(map[string]bool)

	// Initialize distances
	for router := range topology.GetAllLSAs() {
		distances[router] = math.MaxInt32
	}
	distances[source] = 0

	for len(visited) < len(topology.GetAllLSAs()) {
		// Find unvisited node with minimum distance
		current := ""
		minDist := math.MaxInt32

		for node, dist := range distances {
			if !visited[node] && dist < minDist {
				current = node
				minDist = dist
			}
		}

		if current == "" {
			break // No more reachable nodes
		}

		visited[current] = true

		// Check neighbors
		if lsa, exists := topology.GetLSA(current); exists {
			for neighbor, cost := range lsa.Neighbors {
				if !visited[neighbor] {
					alt := distances[current] + cost
					if alt < distances[neighbor] {
						distances[neighbor] = alt
						previous[neighbor] = current
					}
				}
			}
		}
	}

	return &DijkstraResult{
		Distances: distances,
		Previous:  previous,
	}
}
