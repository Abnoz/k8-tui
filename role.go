package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage roles",
		Long:  `Create, delete, and list roles in your Kubernetes cluster.`,
	}

	cmd.AddCommand(newRoleListCmd())
	cmd.AddCommand(newRoleCreateCmd())
	cmd.AddCommand(newRoleDeleteCmd())

	return cmd
}

func newRoleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return err
			}

			roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Roles in namespace %s:\n", namespace)
			for _, role := range roles.Items {
				fmt.Printf("- %s\n", role.Name)
			}
			return nil
		},
	}
}

func newRoleCreateCmd() *cobra.Command {
	var (
		name     string
		verbs    string
		resources string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a role",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("role name is required")
			}

			verbsList := strings.Split(verbs, ",")
			resourcesList := strings.Split(resources, ",")

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     verbsList,
						APIGroups: []string{""},
						Resources: resourcesList,
					},
				},
			}

			role, err = clientset.RbacV1().Roles(namespace).Create(context.TODO(), role, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Role %s created in namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name of the role")
	cmd.Flags().StringVar(&verbs, "verbs", "", "comma-separated list of verbs (e.g., get,list,watch)")
	cmd.Flags().StringVar(&resources, "resources", "", "comma-separated list of resources (e.g., pods,services)")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("verbs")
	cmd.MarkFlagRequired("resources")
	return cmd
}

func newRoleDeleteCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a role",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("role name is required")
			}

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			err = clientset.RbacV1().Roles(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Role %s deleted from namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name of the role")
	cmd.MarkFlagRequired("name")
	return cmd
}
