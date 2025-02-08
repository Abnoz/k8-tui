package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
	"k8s.io/client-go/tools/clientcmd"
)

type ResourceUsage struct {
	Name                string
	Namespace           string
	CurrentCPURequests  float64
	CurrentMemRequests  float64
	AverageCPUUsage    float64
	AverageMemUsage    float64
	RecommendedCPU     float64
	RecommendedMemory  float64
	PotentialCPUSavings float64
	PotentialMemSavings float64
}

// Convert bytes to MB
func bytesToMB(bytes int64) float64 {
	return float64(bytes) / (1024 * 1024)
}

// Convert millicores to cores
func millicoresToCores(millicores int64) float64 {
	return float64(millicores) / 1000
}

func newResourceAnalyzerCmd() *cobra.Command {
	var duration string
	cmd := &cobra.Command{
		Use:   "analyze-resources",
		Short: "Analyze resource usage and provide optimization recommendations",
		Long: `Analyze CPU and Memory usage patterns across pods and namespaces.
Provides recommendations for resource requests and limits based on actual usage patterns.
Helps identify over-provisioned and under-provisioned resources to optimize costs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return analyzeResources(duration)
		},
	}

	cmd.Flags().StringVarP(&duration, "duration", "d", "1h", "Duration to analyze (e.g., 1h, 24h)")
	return cmd
}

func analyzeResources(duration string) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("error building kubeconfig: %v", err)
	}

	// Create metrics client
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating metrics client: %v", err)
	}

	// Get regular clientset
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}

	// Get pods in the specified namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing pods: %v", err)
	}

	var resourceUsages []ResourceUsage

	// Analyze each pod
	for _, pod := range pods.Items {
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Warning: Could not get metrics for pod %s: %v\n", pod.Name, err)
			continue
		}

		var totalCPURequests, totalMemRequests int64
		for _, container := range pod.Spec.Containers {
			cpuRequest := container.Resources.Requests.Cpu().MilliValue()
			memRequest := container.Resources.Requests.Memory().Value()
			totalCPURequests += cpuRequest
			totalMemRequests += memRequest
		}

		var totalCPUUsage, totalMemUsage int64
		for _, container := range podMetrics.Containers {
			cpuUsage := container.Usage.Cpu().MilliValue()
			memUsage := container.Usage.Memory().Value()
			totalCPUUsage += cpuUsage
			totalMemUsage += memUsage
		}

		// Calculate recommended values (using a simple algorithm - can be made more sophisticated)
		recommendedCPU := int64(float64(totalCPUUsage) * 1.2) // 20% buffer
		recommendedMem := int64(float64(totalMemUsage) * 1.2) // 20% buffer

		usage := ResourceUsage{
			Name:                pod.Name,
			Namespace:           pod.Namespace,
			CurrentCPURequests:  millicoresToCores(totalCPURequests),
			CurrentMemRequests:  bytesToMB(totalMemRequests),
			AverageCPUUsage:    millicoresToCores(totalCPUUsage),
			AverageMemUsage:    bytesToMB(totalMemUsage),
			RecommendedCPU:     millicoresToCores(recommendedCPU),
			RecommendedMemory:  bytesToMB(recommendedMem),
			PotentialCPUSavings: millicoresToCores(totalCPURequests - recommendedCPU),
			PotentialMemSavings: bytesToMB(totalMemRequests - recommendedMem),
		}
		resourceUsages = append(resourceUsages, usage)
	}

	// Sort by potential savings
	sort.Slice(resourceUsages, func(i, j int) bool {
		return resourceUsages[i].PotentialCPUSavings > resourceUsages[j].PotentialCPUSavings
	})

	// Print report
	fmt.Printf("\nResource Optimization Report for namespace: %s\n", namespace)
	fmt.Printf("=============================================\n\n")

	var totalCPUSavings, totalMemSavings float64
	for _, usage := range resourceUsages {
		if usage.PotentialCPUSavings > 0 || usage.PotentialMemSavings > 0 {
			fmt.Printf("Pod: %s\n", usage.Name)
			fmt.Printf("  Current CPU Requests: %.3f cores\n", usage.CurrentCPURequests)
			fmt.Printf("  Average CPU Usage: %.3f cores\n", usage.AverageCPUUsage)
			fmt.Printf("  Recommended CPU: %.3f cores\n", usage.RecommendedCPU)
			fmt.Printf("  Potential CPU Savings: %.3f cores\n", usage.PotentialCPUSavings)
			fmt.Printf("  Current Memory Requests: %.1f MB\n", usage.CurrentMemRequests)
			fmt.Printf("  Average Memory Usage: %.1f MB\n", usage.AverageMemUsage)
			fmt.Printf("  Recommended Memory: %.1f MB\n", usage.RecommendedMemory)
			fmt.Printf("  Potential Memory Savings: %.1f MB\n\n", usage.PotentialMemSavings)

			totalCPUSavings += usage.PotentialCPUSavings
			totalMemSavings += usage.PotentialMemSavings
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("Total potential CPU savings: %.3f cores\n", totalCPUSavings)
	fmt.Printf("Total potential memory savings: %.1f MB\n", totalMemSavings)

	return nil
}
