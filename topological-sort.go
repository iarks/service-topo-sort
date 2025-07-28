// topo.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Metadata for each service
type ServiceMetadata struct {
	Repository     string `json:"repository"`
	PathToManifest string `json:"pathToManifest"`
	PathToDevlocal string `json:"pathToDevlocal"`
	Branch         string `json:"branch"`
}

// Input manifest structure
type Manifest struct {
	DefaultBranch           string
	DependencyAdjacencyList map[string][]string        `json:"dependencyAdjacencyList"`
	Services                map[string]ServiceMetadata `json:"services"`
}

// Output structure for deployment order
type DeployDependency struct {
	ServiceName string   `yml:"serviceName"`
	Repository  string   `yml:"repository"`
	Manifest    string   `yml:"manifest"`
	DevLocal    string   `yml:"devLocal"`
	DependsOn   []string `yml:"dependsOn"`
	Branch      string   `yml:"branch"`
}

type FinalDeploymentList struct {
	DeploymentOrder         []DeployDependency
	DependencyAdjacencyList map[string][]string // the dependency list which was used to generate the original deployment order
}

func topoSort(graph map[string][]string) ([]string, error) {
	type state struct {
		node     string
		expanded bool
	}

	visited := make(map[string]bool)
	onPath := make(map[string]bool) // cycle detection
	var result []string

	for node := range graph {

		// if the node is already visited, skip it
		if visited[node] {
			continue
		}

		// create a local stack to save the state of the adjacent nodes
		stack := []state{{node, false}}

		// while the stack is not empty, pop each item from the stack and then perform dfs on that node
		for len(stack) > 0 {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			// if the top node is visited, skip it
			if visited[top.node] {
				continue
			}

			// if the top has been seen, skip it
			if top.expanded {
				// All children processed
				visited[top.node] = true
				onPath[top.node] = false
				result = append(result, top.node)
				continue
			}

			// Push back to stack for post-processing
			stack = append(stack, state{top.node, true})

			if onPath[top.node] {
				return nil, fmt.Errorf("❌ cycle detected at service: %s", top.node)
			}
			onPath[top.node] = true

			for _, dep := range graph[top.node] {
				if !visited[dep] {
					stack = append(stack, state{dep, false})
				}
			}
		}
	}

	return result, nil
}

func main() {
	// Read the manifest file
	data, err := os.ReadFile("dependency-manifest.json")
	if err != nil {
		log.Fatalf("❌ Failed to read dependency-manifest.json: %v", err)
	}

	// Parse JSON
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		log.Fatalf("❌ Invalid JSON in manifest: %v", err)
	}

	// Build dependency graph
	graph := manifest.DependencyAdjacencyList

	// Perform topological sort
	sorted, err := topoSort(graph)
	if err != nil {
		log.Fatalf("🚫 Topological sort failed: %v", err)
	}

	// Output: dependencies first → reverse post-order
	fmt.Println("✅ Build/Deploy Order (dependencies first):")

	var deploymentOrder []DeployDependency

	for i, service := range sorted {
		meta, exists := manifest.Services[service]
		if !exists {
			log.Printf("⚠️ Warning: No metadata found for service '%s', using defaults", service)
			meta = ServiceMetadata{}
		}

		dependsOn := graph[service]

		branch := strings.TrimSpace(meta.Branch)
		if branch == "" {
			branch = manifest.DefaultBranch
		}

		deploymentOrder = append(deploymentOrder, DeployDependency{
			ServiceName: service,
			Repository:  strings.TrimSpace(meta.Repository),
			Manifest:    meta.PathToManifest,
			DevLocal:    meta.PathToDevlocal,
			DependsOn:   dependsOn,
			Branch:      branch,
		})

		// Print human-readable format
		fmt.Printf("%d.\t%s\n", i+1, service)
		fmt.Printf("\t\tRepo: %s\n", deploymentOrder[i].Repository)
		fmt.Printf("\t\tManifest: %s\n", deploymentOrder[i].Manifest)
		fmt.Printf("\t\tDevLocal: %s\n", deploymentOrder[i].DevLocal)
		fmt.Printf("\t\tDepends on: %s\n", strings.Join(dependsOn, ", "))
	}

	finalDeployList := FinalDeploymentList{
		DeploymentOrder:         deploymentOrder,
		DependencyAdjacencyList: manifest.DependencyAdjacencyList,
	}

	// Convert to JSON
	jsonData, err := yaml.Marshal(&finalDeployList)
	if err != nil {
		log.Fatalf("❌ Failed to generate JSON: %v", err)
	}

	// Save to file
	err = os.WriteFile("deployment-order.yml", jsonData, 0644)
	if err != nil {
		log.Fatalf("❌ Failed to write deployment-order.json: %v", err)
	}

	fmt.Println("✅ deployment-order.json has been saved!")
}
