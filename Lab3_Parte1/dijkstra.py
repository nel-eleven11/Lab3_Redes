# Link-State (Dijkstra) para una red no dirigida.
# - Lee una topología desde archivo (formato: "u v costo" por línea).
# - Imprime la tabla de iteraciones (N', D(v), p(v)).
# - Construye la tabla de reenvío (next hop por destino).

from math import inf
import argparse
from typing import Dict, Tuple, List

Graph = Dict[str, Dict[str, float]]

def parse_topology(path: str) -> Graph:
    g: Graph = {}
    with open(path, "r", encoding="utf-8") as f:
        for raw in f:
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            a, b, w = line.split()
            w = float(w) if "." in w else int(w)
            g.setdefault(a, {})[b] = w
            g.setdefault(b, {})[a] = w     # no dirigida
    return g

def dijkstra(graph: Graph, source: str):
    # asegura que todos los nodos existen en el dict
    for u in list(graph.keys()):
        for v, w in graph[u].items():
            graph.setdefault(v, {})
            if u not in graph[v] or graph[v][u] != w:
                graph[v][u] = w

    nodes = sorted(graph.keys())
    dist = {v: inf for v in nodes}
    prev = {v: None for v in nodes}
    visited = set()

    # inicialización (como en Kurose): N' = {source}, D(neighbor)=c(source,neighbor)
    dist[source] = 0
    for v, w in graph[source].items():
        dist[v] = w
        prev[v] = source

    def snapshot():
        return {
            "N_prime": "".join(sorted(visited)),
            "D": {v: (dist[v], prev[v]) for v in nodes},
        }

    steps = []
    visited.add(source)
    steps.append(snapshot())

    while len(visited) < len(nodes):
        # elegir w fuera de N' con D(w) mínimo
        w = min((v for v in nodes if v not in visited), key=lambda v: dist[v])
        visited.add(w)
        # relajar vecinos de w
        for v, cwv in graph[w].items():
            if v in visited:
                continue
            alt = dist[w] + cwv
            if alt < dist[v]:
                dist[v] = alt
                prev[v] = w
        steps.append(snapshot())

    return dist, prev, steps, nodes

def reconstruct_path(prev: Dict[str, str], source: str, dest: str):
    path = []
    node = dest
    while node is not None:
        path.append(node)
        if node == source:
            break
        node = prev[node]
    if path[-1] != source:
        return None
    return list(reversed(path))

def forwarding_table(prev: Dict[str, str], source: str, dist: Dict[str, float]):
    table = {}
    for dest in prev:
        if dest == source:
            continue
        if dist.get(dest, inf) == inf:
            table[dest] = None
            continue
        node = dest
        while prev[node] is not None and prev[node] != source:
            node = prev[node]
        table[dest] = node if node != source else dest
    return table

def format_steps_table(steps, nodes):
    headers = ["step", "N'"] + [f"D({v}), p({v})" for v in nodes]
    rows = []
    for i, s in enumerate(steps):
        row = {"step": str(i), "N'": s["N_prime"]}
        for v in nodes:
            d, p = s["D"][v]
            sd = "∞" if d == inf else str(int(d) if abs(d - int(d)) < 1e-9 else d)
            sp = "" if p is None else str(p)
            row[f"D({v}), p({v})"] = f"{sd}, {sp}".rstrip(", ")
        rows.append(row)

    # ancha simple
    widths = {h: max(len(h), max(len(r[h]) for r in rows)) for h in headers}
    line = " | ".join(f"{h:<{widths[h]}}" for h in headers)
    sep  = "-+-".join("-"*widths[h] for h in headers)
    out = [line, sep]
    for r in rows:
        out.append(" | ".join(f"{r[h]:<{widths[h]}}" for h in headers))
    return "\n".join(out)

def format_forwarding_table(fwd, source):
    headers = ["Destination", "Next-hop"]
    rows = []
    for dest in sorted(fwd):
        nh = fwd[dest]
        rows.append({
            "Destination": dest,
            "Next-hop": "unreachable" if nh is None else f"({source}, {nh})"
        })
    widths = {h: max(len(h), max(len(r[h]) for r in rows)) for h in headers}
    line = " | ".join(f"{h:<{widths[h]}}" for h in headers)
    sep  = "-+-".join("-"*widths[h] for h in headers)
    out = [line, sep]
    for r in rows:
        out.append(" | ".join(f"{r[h]:<{widths[h]}}" for h in headers))
    return "\n".join(out)

def main():
    ap = argparse.ArgumentParser(description="Dijkstra (Link-State) para topologías no dirigidas.")
    ap.add_argument("-t", "--topology", required=True, help="Archivo con líneas 'u v costo'")
    ap.add_argument("-s", "--source", required=True, help="Nodo fuente")
    ap.add_argument("--no-steps", action="store_true", help="No imprimir la tabla de iteraciones")
    args = ap.parse_args()

    g = parse_topology(args.topology)
    dist, prev, steps, nodes = dijkstra(g, args.source)

    if not args.no_steps:
        print("\n== Tabla de iteraciones ==")
        print(format_steps_table(steps, nodes))

    fwd = forwarding_table(prev, args.source, dist)
    print("\n== Forwarding table (next-hop por destino) ==")
    print(format_forwarding_table(fwd, args.source))

    print("\n== Caminos mínimos desde", args.source, "==")
    for dest in sorted(nodes):
        if dest == args.source: 
            continue
        if dist[dest] == inf:
            print(f"{args.source} -> {dest}: unreachable")
        else:
            path = reconstruct_path(prev, args.source, dest)
            cost = int(dist[dest]) if abs(dist[dest]-int(dist[dest])) < 1e-9 else dist[dest]
            print(f"{args.source} -> {dest}  costo={cost}  camino={'-'.join(path)}")

if __name__ == "__main__":
    main()
