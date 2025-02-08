package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newRoleBindingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rolebinding",
		Short: "Manage role bindings",
		Long:  `Create, delete, and list role bindings in your Kubernetes cluster.`,
	}

	cmd.AddCommand(newRoleBindingListCmd())
	cmd.AddCommand(newRoleBindingCreateCmd())
	cmd.AddCommand(newRoleBindingDeleteCmd())

	return cmd
}

func newRoleBindingListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List role bindings",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return err
			}

			rbs, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Role bindings in namespace %s:\n", namespace)
			for _, rb := range rbs.Items {
				fmt.Printf("- %s (Role: %s)\n", rb.Name, rb.RoleRef.Name)
				for _, subject := range rb.Subjects {
					fmt.Printf("  Subject: %s (%s)\n", subject.Name, subject.Kind)
				}
			}
			return nil
		},
	}
}

func newRoleBindingCreateCmd() *cobra.Command {
	var (
		name           string
		role           string
		serviceAccount string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a role binding",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || role == "" || serviceAccount == "" {
				return fmt.Errorf("name, role, and serviceaccount are required")
			}

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			// Parse service account namespace and name
			parts := strings.Split(serviceAccount, ":")
			if len(parts) != 2 {
				return fmt.Errorf("serviceaccount must be in format namespace:name")
			}
			saNamespace, saName := parts[0], parts[1]

			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     role,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      saName,
						Namespace: saNamespace,
					},
				},
			}

			rb, err = clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), rb, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Role binding %s created in namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name of the role binding")
	cmd.Flags().StringVar(&role, "role", "", "name of the role to bind")
	cmd.Flags().StringVar(&serviceAccount, "serviceaccount", "", "service account in format namespace:name")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("role")
	cmd.MarkFlagRequired("serviceaccount")
	return cmd
}

func newRoleBindingDeleteCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a role binding",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("role binding name is required")
			}

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			err = clientset.RbacV1().RoleBindings(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Role binding %s deleted from namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name of the role binding")
	cmd.MarkFlagRequired("name")
	return cmd
}
