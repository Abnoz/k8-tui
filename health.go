package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newHealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check cluster health",
		Long:  `View cluster health information including node status and resource utilization.`,
	}

	cmd.AddCommand(newNodeStatusCmd())
	cmd.AddCommand(newPodDistributionCmd())
	cmd.AddCommand(newResourceUtilizationCmd())

	return cmd
}

func newNodeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nodes",
		Short: "Check node status",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return err
			}

			nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			fmt.Println("Node Status:")
			for _, node := range nodes.Items {
				ready := "NotReady"
				for _, condition := range node.Status.Conditions {
					if condition.Type == "Ready" {
						if condition.Status == "True" {
							ready = "Ready"
						}
						break
					}
				}
				fmt.Printf("- %s: %s\n", node.Name, ready)
				fmt.Printf("  Version: %s\n", node.Status.NodeInfo.KubeletVersion)
				fmt.Printf("  OS: %s\n", node.Status.NodeInfo.OperatingSystem)
			}
			return nil
		},
	}
}

func newPodDistributionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pods",
		Short: "View pod distribution",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return err
			}

			pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			nodePodCount := make(map[string]int)
			for _, pod := range pods.Items {
				nodePodCount[pod.Spec.NodeName]++
			}

			fmt.Printf("Pod distribution in namespace %s:\n", namespace)
			for node, count := range nodePodCount {
				fmt.Printf("- Node %s: %d pods\n", node, count)
			}
			return nil
		},
	}
}

func newResourceUtilizationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resources",
		Short: "View resource utilization",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return err
			}

			nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			fmt.Println("Resource Utilization:")
			for _, node := range nodes.Items {
				allocatable := node.Status.Allocatable
				capacity := node.Status.Capacity

				fmt.Printf("Node: %s\n", node.Name)
				fmt.Printf("  CPU:\n")
				fmt.Printf("    Capacity: %v\n", capacity.Cpu())
				fmt.Printf("    Allocatable: %v\n", allocatable.Cpu())
				fmt.Printf("  Memory:\n")
				fmt.Printf("    Capacity: %v\n", capacity.Memory())
				fmt.Printf("    Allocatable: %v\n", allocatable.Memory())
				fmt.Printf("  Pods:\n")
				fmt.Printf("    Capacity: %v\n", capacity.Pods())
				fmt.Printf("    Allocatable: %v\n", allocatable.Pods())
			}
			return nil
		},
	}
}
