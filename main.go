package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/k8s-admin-cli/ui"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeconfig string
	namespace  string
	rootCmd    *cobra.Command
	tui        bool
)

func main() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	rootCmd = &cobra.Command{
		Use:   "k8s-admin",
		Short: "Kubernetes administration CLI tool",
		Long:  `A command line tool for managing Kubernetes permissions, service accounts, and administrative tasks.`,
		Run: func(cmd *cobra.Command, args []string) {
			if tui {
				if err := ui.New().Start(); err != nil {
					fmt.Printf("Error running TUI: %v\n", err)
					os.Exit(1)
				}
			} else {
				cmd.Help()
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "path to the kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "kubernetes namespace")
	rootCmd.Flags().BoolVarP(&tui, "tui", "t", false, "start terminal user interface")

	// Add commands
	rootCmd.AddCommand(newServiceAccountCmd())
	rootCmd.AddCommand(newRoleCmd())
	rootCmd.AddCommand(newRoleBindingCmd())
	rootCmd.AddCommand(newHealthCmd())
	rootCmd.AddCommand(newResourceAnalyzerCmd())
	rootCmd.AddCommand(newVisualizeCmd())
	rootCmd.AddCommand(newPodCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getClientset() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
