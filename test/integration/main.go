package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"k8s-dojo/pkg/cluster"
	"k8s-dojo/pkg/engine"
	"k8s-dojo/pkg/k8s"
	"k8s-dojo/pkg/scenario"
)

func main() {
	scenarioID := flag.String("scenario", "image-pull-backoff", "ID of the scenario to run")
	flag.Parse()

	fmt.Println("=== K8s-Dojo Integration Test ===")
	fmt.Printf("Testing Scenario: %s\n\n", *scenarioID)

	// 1. Ensure Cluster Exists
	fmt.Println("1. Connecting to existing cluster...")
	cm := cluster.NewManager()

	exists, err := cm.ClusterExists()
	if err != nil {
		log.Fatalf("Failed to check cluster existence: %v", err)
	}
	if !exists {
		log.Fatal("Cluster does not exist. Please run ./k8s-dojo first to create it.")
	}
	fmt.Println("   ✅ Connected to cluster")

	// 2. Create K8s Client
	fmt.Println("2. Creating K8s client...")
	// We need kubeconfig. NewManager provided it, but here we can easier just use default loading or trick it.
	// Actually k8s.NewClientFromKubeconfig takes string content.
	// In real app, main passes it. Here we need to retrieve it.
	// Kind exports kubeconfig.

	// Simplification: We'll re-export kubeconfig using Kind command or just assume default default if we were running outside.
	// But our pkg/k8s uses explicit config.
	// Let's use the cluster manager to get it.
	cfg, err := cm.EnsureCluster(cluster.SupportedVersions()[0]) // This might be slow if it validates too much?
	// EnsureCluster returns kubeconfig string.
	if err != nil {
		log.Fatalf("Failed to get kubeconfig: %v", err)
	}

	client, err := k8s.NewClientFromKubeconfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	v, err := client.Clientset.Discovery().ServerVersion()
	if err != nil {
		log.Fatalf("Failed to get server version: %v", err)
	}
	fmt.Printf("   ✅ Client created (Server: %s)\n", v.String())

	// 3. Initialize Engine
	fmt.Println("3. Initializing game engine...")
	reg := scenario.NewRegistry(client.Clientset)
	eng := engine.NewEngine(reg)
	fmt.Printf("   ✅ Engine ready (%d scenarios available)\n", reg.Count())

	if reg.Get(*scenarioID) == nil {
		log.Fatalf("Scenario %s not found", *scenarioID)
	}

	// 4. Start Scenario
	fmt.Printf("4. Starting '%s' scenario...\n", *scenarioID)
	ctx := context.Background()
	if err := eng.StartScenario(ctx, *scenarioID); err != nil {
		log.Fatalf("Failed to start scenario: %v", err)
	}
	fmt.Println("   ✅ Scenario started")

	// 5. Wait a bit
	fmt.Println("5. Waiting for resources to be created...")
	time.Sleep(5 * time.Second)

	// 6. Check (Should fail initial check)
	fmt.Println("6. Validating initial state (should NOT be solved)...")
	res, err := eng.Check(ctx)
	if err != nil {
		log.Fatalf("Check failed: %v", err)
	}

	if res.Solved {
		log.Fatal("❌ Scenario solved immediately? That shouldn't happen.")
	} else {
		fmt.Printf("   ✅ Correct: Not solved yet (%s)\n", res.Message)
	}

	// 7. Cleanup
	fmt.Println("7. Cleaning up scenario...")
	if err := eng.Cleanup(ctx); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}
	fmt.Println("   ✅ Cleanup complete")

	fmt.Println("\n=== All Tests Passed! ===")
}
