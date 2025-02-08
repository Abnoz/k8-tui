package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newServiceAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sa",
		Short: "Manage service accounts",
		Long:  `Create, delete, and list service accounts in your Kubernetes cluster.`,
	}

	cmd.AddCommand(newSAListCmd())
	cmd.AddCommand(newSACreateCmd())
	cmd.AddCommand(newSADeleteCmd())

	return cmd
}

func newSAListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List service accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return err
			}

			sas, err := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Service accounts in namespace %s:\n", namespace)
			for _, sa := range sas.Items {
				fmt.Printf("- %s\n", sa.Name)
			}
			return nil
		},
	}
}

func newSACreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a service account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("service account name is required")
			}

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}

			sa, err = clientset.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Service account %s created in namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name of the service account")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newSADeleteCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a service account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("service account name is required")
			}

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			err = clientset.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Service account %s deleted from namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name of the service account")
	cmd.MarkFlagRequired("name")
	return cmd
}
