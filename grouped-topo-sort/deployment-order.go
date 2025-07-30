// topo.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Output structure for deployment order
type DeployDependency struct {
	ServiceName string   `yaml:"serviceName"`
	Repository  string   `yaml:"repository"`
	Manifest    string   `yaml:"manifest"`
	DevLocal    string   `yaml:"devLocal"`
	DependsOn   []string `yaml:"dependsOn"`
	Branch      string   `yaml:"branch"`
}

type FinalDeploymentList struct {
	DeploymentOrder []DeployDependency
}

func main() {

	// Define command-line flags
	unionFile := flag.String("u", "./union.yml", "YAML file containing the union file")
	deploymentOrderFile := flag.String("d", "./deployment-order.yml", "file containing the topologically sorted graph nodes")

	flag.Parse()

	if *unionFile == "" || *deploymentOrderFile == "" {
		fmt.Printf("unionfile: %s\n", *unionFile)
		fmt.Printf("deployment file: %s\n", *deploymentOrderFile)
		log.Fatalf("Files required")
	}

	// Read the union file into a mf
	data, err := os.ReadFile(*unionFile)
	if err != nil {
		log.Fatalf("❌ Failed to read %s. %v", *unionFile, err)
	}
	var union = make(map[string]string, 0)
	if err := yaml.Unmarshal(data, &union); err != nil {
		log.Fatalf("❌ Invalid YML in union file: %v", err)
	}
	// fmt.Printf("union: %v", union)

	// read the toposort file into a list of nodes
	data, err = os.ReadFile(*deploymentOrderFile)
	if err != nil {
		log.Fatalf("❌ Failed to read %s. %v", *deploymentOrderFile, err)
	}

	var deploymentOrder = FinalDeploymentList{
		DeploymentOrder: make([]DeployDependency, 0),
	}
	if err := yaml.Unmarshal(data, &deploymentOrder); err != nil {
		log.Fatalf("❌ Invalid YML in union file: %v", err)
	}
	// fmt.Printf("deployment order: %v", deploymentOrder)

	var finalDeploymentOrder = make(map[string][]DeployDependency)

	for _, service := range deploymentOrder.DeploymentOrder {
		var serviceName = service.ServiceName
		var root = union[serviceName]
		finalDeploymentOrder[root] = append(finalDeploymentOrder[root], service)
	}
	// fmt.Printf("deployment order: %v", finalDeploymentOrder)

	for _, service := range finalDeploymentOrder {
		for _, s := range service {
			fmt.Printf("%s\n", s)
		}
		fmt.Println("---------------------------------")
	}
}
