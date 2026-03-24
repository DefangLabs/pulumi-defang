package common

import "github.com/DefangLabs/pulumi-defang/provider/compose"

func TopologicalSort(nodes compose.Services) []string {
	sorted := make([]string, 0, len(nodes))
	visited := make(map[string]struct{}, len(nodes))

	var visit func(name string)
	visit = func(name string) {
		if _, ok := visited[name]; ok {
			return
		}
		visited[name] = struct{}{}
		for dep := range nodes[name].DependsOn {
			visit(dep)
		}
		sorted = append(sorted, name)
	}

	for name := range nodes {
		visit(name)
	}

	return sorted
}
