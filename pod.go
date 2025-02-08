package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newPodCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pod",
		Short: "Manage pods",
		Long:  `Create, delete, and list pods in your Kubernetes cluster.`,
	}

	cmd.AddCommand(newPodListCmd())
	cmd.AddCommand(newPodCreateCmd())
	cmd.AddCommand(newPodDeleteCmd())

	return cmd
}

func newPodListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientset, err := getClientset()
			if err != nil {
				return err
			}

			pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Pods in namespace %s:\n", namespace)
			for _, pod := range pods.Items {
				fmt.Printf("- %s (Status: %s)\n", pod.Name, pod.Status.Phase)
				if len(pod.Spec.Containers) > 0 {
					container := pod.Spec.Containers[0]
					fmt.Printf("  Image: %s\n", container.Image)
					if len(container.Ports) > 0 {
						fmt.Printf("  Ports: ")
						for _, port := range container.Ports {
							fmt.Printf("%d/%s ", port.ContainerPort, port.Protocol)
						}
						fmt.Println()
					}
				}
			}
			return nil
		},
	}
}

func newPodCreateCmd() *cobra.Command {
	var (
		name           string
		image          string
		cpu            string
		memory         string
		envVars        []string
		labels         []string
		ports          []string
		volumeMounts   []string
		configMapNames []string
		secretNames    []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pod",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("pod name is required")
			}
			if image == "" {
				return fmt.Errorf("container image is required")
			}

			// Parse environment variables
			envSlice := []corev1.EnvVar{}
			for _, env := range envVars {
				parts := strings.Split(env, "=")
				if len(parts) != 2 {
					return fmt.Errorf("invalid environment variable format: %s", env)
				}
				envSlice = append(envSlice, corev1.EnvVar{
					Name:  parts[0],
					Value: parts[1],
				})
			}

			// Parse labels
			labelMap := make(map[string]string)
			for _, label := range labels {
				parts := strings.Split(label, "=")
				if len(parts) != 2 {
					return fmt.Errorf("invalid label format: %s", label)
				}
				labelMap[parts[0]] = parts[1]
			}

			// Parse ports
			portSlice := []corev1.ContainerPort{}
			for _, port := range ports {
				var containerPort int32
				var protocol string
				if strings.Contains(port, "/") {
					parts := strings.Split(port, "/")
					fmt.Sscanf(parts[0], "%d", &containerPort)
					protocol = strings.ToUpper(parts[1])
				} else {
					fmt.Sscanf(port, "%d", &containerPort)
					protocol = "TCP"
				}
				portSlice = append(portSlice, corev1.ContainerPort{
					ContainerPort: containerPort,
					Protocol:      corev1.Protocol(protocol),
				})
			}

			// Create volumes and volume mounts
			volumes := []corev1.Volume{}
			volumeMountSlice := []corev1.VolumeMount{}

			// Add ConfigMap volumes
			for i, configMapName := range configMapNames {
				volumeName := fmt.Sprintf("config-volume-%d", i)
				volumes = append(volumes, corev1.Volume{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMapName,
							},
						},
					},
				})
				volumeMountSlice = append(volumeMountSlice, corev1.VolumeMount{
					Name:      volumeName,
					MountPath: fmt.Sprintf("/etc/config/%s", configMapName),
				})
			}

			// Add Secret volumes
			for i, secretName := range secretNames {
				volumeName := fmt.Sprintf("secret-volume-%d", i)
				volumes = append(volumes, corev1.Volume{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretName,
						},
					},
				})
				volumeMountSlice = append(volumeMountSlice, corev1.VolumeMount{
					Name:      volumeName,
					MountPath: fmt.Sprintf("/etc/secrets/%s", secretName),
				})
			}

			// Parse custom volume mounts
			for _, mount := range volumeMounts {
				parts := strings.Split(mount, ":")
				if len(parts) != 2 {
					return fmt.Errorf("invalid volume mount format: %s", mount)
				}
				volumeName := parts[0]
				mountPath := parts[1]
				volumeMountSlice = append(volumeMountSlice, corev1.VolumeMount{
					Name:      volumeName,
					MountPath: mountPath,
				})
			}

			// Create resource requirements
			resources := corev1.ResourceRequirements{}
			if cpu != "" || memory != "" {
				resources.Limits = corev1.ResourceList{}
				resources.Requests = corev1.ResourceList{}

				if cpu != "" {
					cpuResource, err := resource.ParseQuantity(cpu)
					if err != nil {
						return fmt.Errorf("invalid CPU format: %v", err)
					}
					resources.Limits[corev1.ResourceCPU] = cpuResource
					resources.Requests[corev1.ResourceCPU] = cpuResource
				}

				if memory != "" {
					memoryResource, err := resource.ParseQuantity(memory)
					if err != nil {
						return fmt.Errorf("invalid memory format: %v", err)
					}
					resources.Limits[corev1.ResourceMemory] = memoryResource
					resources.Requests[corev1.ResourceMemory] = memoryResource
				}
			}

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels:    labelMap,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         name,
							Image:        image,
							Env:          envSlice,
							Ports:        portSlice,
							VolumeMounts: volumeMountSlice,
							Resources:    resources,
						},
					},
					Volumes: volumes,
				},
			}

			pod, err = clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Pod %s created in namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the pod")
	cmd.Flags().StringVar(&image, "image", "", "Container image to use")
	cmd.Flags().StringVar(&cpu, "cpu", "", "CPU resource limit (e.g., '200m' or '0.2')")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory resource limit (e.g., '128Mi' or '1Gi')")
	cmd.Flags().StringSliceVar(&envVars, "env", []string{}, "Environment variables (format: KEY=VALUE)")
	cmd.Flags().StringSliceVar(&labels, "label", []string{}, "Pod labels (format: KEY=VALUE)")
	cmd.Flags().StringSliceVar(&ports, "port", []string{}, "Container ports (format: PORT[/PROTOCOL])")
	cmd.Flags().StringSliceVar(&volumeMounts, "volume-mount", []string{}, "Volume mounts (format: VOLUME_NAME:MOUNT_PATH)")
	cmd.Flags().StringSliceVar(&configMapNames, "configmap", []string{}, "ConfigMap names to mount")
	cmd.Flags().StringSliceVar(&secretNames, "secret", []string{}, "Secret names to mount")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("image")

	return cmd
}

func newPodDeleteCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a pod",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("pod name is required")
			}

			clientset, err := getClientset()
			if err != nil {
				return err
			}

			err = clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			fmt.Printf("Pod %s deleted from namespace %s\n", name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name of the pod")
	cmd.MarkFlagRequired("name")
	return cmd
}
