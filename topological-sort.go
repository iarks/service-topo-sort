// topo.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type ServiceMetadata struct {
	Repository     string `json:"repository"`
	PathToManifest string `json:"pathToManifest"`
	PathToDevlocal string `json:"pathToDevlocal"`
}

type Manifest struct {
	DependencyAdjacencyList map[string][]string        `json:"dependencyAdjacencyList"`
	Services                map[string]ServiceMetadata `json:"services"`
}

var sorted []string

type DeployDependency struct {
	ServiceName string   `json:"serviceName"`
	Repository  string   `json:"repository"`
	Manifest    string   `json:"manifest"`
	DevLocal    string   `json:"devLocal"`
	DependsOn   []string `json:"dependsOn"`
}

func dfs(node string, visited map[string]bool, temp map[string]bool, graph map[string][]string) error {

	// if the current node has been visited already, return
	if visited[node] {
		return nil
	}

	// // if the node has been seen before in the same path, this is a cycle
	if temp[node] {
		return fmt.Errorf("❌ cycle detected at service: %s", node)
	}

	temp[node] = true

	// dfs for each node in the dependency list
	for _, dep := range graph[node] {
		if err := dfs(dep, visited, temp, graph); err != nil {
			return err
		}
	}

	temp[node] = false
	visited[node] = true
	sorted = append(sorted, node)
	return nil
}

func main() {
	// Read file
	data, err := os.ReadFile("dependency-manifest.json")
	if err != nil {
		log.Fatalf("❌ Failed to read manifest.json: %v", err)
	}

	// convert file into json
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		log.Fatalf("❌ Invalid JSON: %v", err)
	}

	// Build reverse dependency graph for topo sort
	graph := manifest.DependencyAdjacencyList

	// Track visited nodes
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	// Visit all nodes
	for service := range graph {
		if !visited[service] {
			if err := dfs(service, visited, temp, graph); err != nil {
				log.Fatalf("🚫 Topo sort failed: %v", err)
			}
		}
	}

	// Output: dependencies first → reverse post-order
	fmt.Println("✅ Build/Deploy Order (dependencies first):")

	var deploymentOrder []DeployDependency

	for i := 0; i < len(sorted); i++ {
		service := sorted[i]
		meta := manifest.Services[service]
		repository := meta.Repository
		manifestPath := meta.PathToManifest
		dependsOn := manifest.DependencyAdjacencyList[service]

		deploymentOrder = append(deploymentOrder, DeployDependency{ServiceName: service, Manifest: manifestPath, Repository: repository, DevLocal: meta.PathToDevlocal, DependsOn: dependsOn})
	}

	for i := 0; i < len(deploymentOrder); i++ {
		fmt.Printf("%d.\t%s\n", i+1, deploymentOrder[i].ServiceName)
		fmt.Printf("\t\tRepo: %s\n", deploymentOrder[i].Repository)
		fmt.Printf("\t\tManifest: %s\n", deploymentOrder[i].Manifest)
		fmt.Printf("\t\tDevLocal: %s\n", deploymentOrder[i].DevLocal)
		fmt.Printf("\t\tDepends on: %s\n", strings.Join(deploymentOrder[i].DependsOn, ", "))
	}

	jsonData, err := json.MarshalIndent(deploymentOrder, "", "  ")
	if err != nil {
		log.Fatalf("❌ Failed to generate JSON: %v", err)
	}

	fmt.Println(string(jsonData))

	// Write JSON to file
	err = os.WriteFile("deployment-order.json", jsonData, 0644)
	if err != nil {
		log.Fatalf("❌ Failed to write to file: %v", err)
	}

	fmt.Println("✅ deployment-order.json has been saved!")
}
