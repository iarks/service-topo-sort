package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the structure of the YAML file
type Config struct {
	DependencyAdjacencyList map[string]ServiceDependencies `yaml:"dependencyAdjacencyList"`
}

type ServiceDependencies struct {
	DependsOn []string `yaml:"dependsOn"`
}

// UnionFind represents the Union-Find data structure
type UnionFind struct {
	parent map[string]string
}

// InitialiseUnionFind initializes a new UnionFind structure
func InitialiseUnionFind(nodes []string) *UnionFind {
	uf := &UnionFind{
		parent: make(map[string]string),
	}

	// set all node as the parent of itself. This is the starting point of an union find
	for _, node := range nodes {
		uf.parent[node] = node
	}
	return uf
}

// Find finds the root of x with path compression
func (uf *UnionFind) Find(x string) string {

	// if the parent of x is x itself, we have reached the top most root, if not, we recursively find the parent of this node
	if uf.parent[x] != x {
		uf.parent[x] = uf.Find(uf.parent[x]) // path compression
	}
	return uf.parent[x]
}

// Union merges the sets containing x and y
func (uf *UnionFind) Union(x, y string) {
	rootX := uf.Find(x) // find the root of X
	rootY := uf.Find(y) // find the root of Y

	// if the 2 roots are different, we make one of them the root of another
	// Why?
	// Because this is the union function. Union means we are combining these 2 nodes to be part of the same set
	if rootX != rootY {
		uf.parent[rootY] = rootX
	}
}

// GetGroups returns a map of root -> list of nodes in that group
func (uf *UnionFind) GetGroups() map[string]string {

	groups := make(map[string]string)

	for node := range uf.parent {
		root := uf.Find(node)
		groups[node] = root
	}
	return groups
}

func main() {

	// Define command-line flags
	inputFile := flag.String("i", "", "Input YAML file path")
	outputFile := flag.String("o", "", "Output YAML file path")

	flag.Parse()

	// Validate required flags
	if *inputFile == "" {
		log.Fatal("At least input (-i) flag is required")
	}

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", *inputFile, err)
	}

	// read the file content into a struct
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	// Collect all unique service names
	allServices := make(map[string]bool)
	// first traverse all the roots
	for service := range config.DependencyAdjacencyList {
		allServices[service] = true
	}
	// then traverse all the indivisual "DependsOn" values
	for _, deps := range config.DependencyAdjacencyList {
		for _, dep := range deps.DependsOn {
			allServices[dep] = true
		}
	}
	// convert the map into a list.
	// Qn:	Why do we first add stuff to a map and then to a list?
	// Ans:	A service can be a dependency of more than 1 services. If we directly add stuff to a list, there may be repeated stuff
	serviceList := make([]string, 0, len(allServices))
	for svc := range allServices {
		serviceList = append(serviceList, svc)
	}

	// Initialize Union-Find
	uf := InitialiseUnionFind(serviceList)

	// Union all dependencies
	for service, deps := range config.DependencyAdjacencyList {
		for _, dep := range deps.DependsOn {
			uf.Union(service, dep)
		}
	}

	// Group services by connected components
	groups := uf.GetGroups()

	// Marshal to YAML
	outputData, err := yaml.Marshal(&groups)
	if err != nil {
		log.Fatalf("Failed to marshal output YAML: %v", err)
	}

	if *outputFile != "" {
		// Write to output file if the output file is provided
		err = os.WriteFile(*outputFile, outputData, 0644)
		if err != nil {
			log.Fatalf("Failed to write output file %s: %v", *outputFile, err)
		}
	} else {

		// if output file not provided print the value

		fmt.Println("# Connected Components:")

		for root, members := range groups {
			fmt.Printf("%s: %v\n", root, members)
		}
	}
}
