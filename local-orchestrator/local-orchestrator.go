// local-deploy.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// === Input Structures ===

type Override struct {
	ServiceName  string `json:"serviceName"`
	Branch       string `json:"branch,omitempty"`
	ManifestPath string `json:"manifestPath,omitempty"`
	DevLocal     string `json:"devLocal,omitempty"`
	Skip         bool   `json:"skip"`
	ForceDeploy  bool   `json:"forceDeploy"`
}

type LocalConfig struct {
	ServiceName         string     `json:"serviceName"`
	DependencyOverrides []Override `json:"dependencyOverrides"`
}

// Master manifest includes both order and graph
type MasterManifest struct {
	DeploymentOrder         []DeployableService `json:"DeploymentOrder"`
	DependencyAdjacencyList map[string][]string `json:"DependencyAdjacencyList"`
}

type DeployableService struct {
	ServiceName string   `json:"serviceName"`
	Repository  string   `json:"repository"`
	Manifest    string   `json:"manifest"`
	DevLocal    string   `json:"devLocal"`
	DependsOn   []string `json:"dependsOn"`
	Branch      string   `json:"branch"`
	OriginalIdx int      `json:"-"` // not serialized, used for sorting
}

func main() {
	// Read master manifest
	masterData, err := os.ReadFile("deployment-order.json")
	if err != nil {
		log.Fatalf("❌ Failed to read deployment-order.json: %v", err)
	}

	var master MasterManifest
	if err := json.Unmarshal(masterData, &master); err != nil {
		log.Fatalf("❌ Invalid master JSON: %v", err)
	}

	// Build lookup maps
	serviceMap := make(map[string]DeployableService)
	originalIndex := make(map[string]int)
	for i, svc := range master.DeploymentOrder {
		cleanSvc := DeployableService{
			ServiceName: svc.ServiceName,
			Repository:  strings.TrimSpace(svc.Repository),
			Manifest:    svc.Manifest,
			DevLocal:    svc.DevLocal,
			DependsOn:   svc.DependsOn,
			Branch:      svc.Branch,
			OriginalIdx: i,
		}
		serviceMap[svc.ServiceName] = cleanSvc
		originalIndex[svc.ServiceName] = i
	}

	graph := master.DependencyAdjacencyList

	// Read local config
	localData, err := os.ReadFile("local-config.json")
	if err != nil {
		log.Fatalf("❌ Failed to read local-config.json: %v", err)
	}

	var local LocalConfig
	if err := json.Unmarshal(localData, &local); err != nil {
		log.Fatalf("❌ Invalid local config JSON: %v", err)
	}

	// find all the services which are dependent on this serivce

	// Build override map
	overrideMap := make(map[string]Override)
	for _, o := range local.DependencyOverrides {
		overrideMap[o.ServiceName] = o
	}

	// Set of services to deploy
	deploySet := make(map[string]bool)

	// Step 1: Add root service and its transitive dependencies
	root := local.ServiceName
	if _, exists := serviceMap[root]; !exists {
		log.Fatalf("❌ Root service '%s' not found in master list", root)
	}
	deploySet[root] = true
	addTransitiveDeps(root, graph, deploySet)

	// Step 2: Add force-deploy services and their deps
	for _, override := range overrideMap {
		if override.ForceDeploy {
			deploySet[override.ServiceName] = true
			addTransitiveDeps(override.ServiceName, graph, deploySet)
		}
	}

	// Step 3: Apply skip (remove from deploySet)
	for _, override := range overrideMap {
		if override.Skip {
			delete(deploySet, override.ServiceName)
		}
	}

	// Step 4: Build final list in master order
	var finalList []DeployableService
	for _, svc := range master.DeploymentOrder {
		if deploySet[svc.ServiceName] {
			override, hasOverride := overrideMap[svc.ServiceName]

			// Apply overrides
			branch := svc.Branch // default from master
			manifest := svc.Manifest
			devLocal := svc.DevLocal

			if hasOverride {
				if override.Branch != "" {
					branch = override.Branch
				}
				if override.ManifestPath != "" {
					manifest = override.ManifestPath
				}
				if override.DevLocal != "" {
					devLocal = override.DevLocal
				}
			}

			finalList = append(finalList, DeployableService{
				ServiceName: svc.ServiceName,
				Repository:  svc.Repository,
				Branch:      branch,
				Manifest:    manifest,
				DevLocal:    devLocal,
				DependsOn:   svc.DependsOn,
				OriginalIdx: svc.OriginalIdx,
			})
		}
	}

	// Step 5: Print result
	fmt.Println("🚀 Final Deployment Plan (in topological order):")
	for i, svc := range finalList {
		fmt.Printf("%d. %s\n", i+1, svc.ServiceName)
		fmt.Printf("   Repo: %s\n", svc.Repository)
		fmt.Printf("   Branch: %s\n", svc.Branch)
		fmt.Printf("   Manifest: %s\n", svc.Manifest)
		fmt.Printf("   DevLocal: %s\n", svc.DevLocal)
		fmt.Println()
	}

	// Optional: Save to file
	outputData, err := json.MarshalIndent(finalList, "", "  ")
	if err != nil {
		log.Fatalf("❌ Failed to generate output JSON: %v", err)
	}
	_ = os.WriteFile("local-deployment-plan.json", outputData, 0644)
	fmt.Println("✅ Saved to local-deployment-plan.json")
}

// Recursively add all dependencies
func addTransitiveDeps(service string, graph map[string][]string, set map[string]bool) {
	for _, dep := range graph[service] {
		if !set[dep] {
			set[dep] = true
			addTransitiveDeps(dep, graph, set)
		}
	}
}
