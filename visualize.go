package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/k8s-admin-cli/visualizer"
	"github.com/spf13/cobra"
)

func newVisualizeCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "visualize",
		Short: "Visualize Kubernetes resource dependencies",
		Long: `Create a visual graph of Kubernetes resource dependencies in your cluster.
This will show relationships between:
- Deployments and Services
- Deployments and ConfigMaps
- Deployments and Secrets
- Services and their selected Pods
The output will be saved as a PNG file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes client: %v", err)
			}

			if outputDir == "" {
				outputDir = "."
			}

			// Create timestamp-based filename
			timestamp := time.Now().Format("2006-01-02-150405")
			filename := fmt.Sprintf("k8s-dependencies-%s.png", timestamp)
			outputPath := filepath.Join(outputDir, filename)

			viz := visualizer.New(clientset, namespace)
			if err := viz.CreateGraph(outputPath); err != nil {
				return fmt.Errorf("failed to create dependency graph: %v", err)
			}

			fmt.Printf("Successfully created dependency graph: %s\n", outputPath)
			fmt.Println("\nThe graph shows:")
			fmt.Println("- Blue boxes: Deployments")
			fmt.Println("- Green ovals: Services")
			fmt.Println("- Yellow notes: ConfigMaps")
			fmt.Println("- Pink notes: Secrets")
			fmt.Println("\nArrows indicate relationships between resources.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "directory to save the visualization (default: current directory)")
	return cmd
}
